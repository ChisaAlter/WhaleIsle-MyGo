//! Clean-room WASI/native shim: rasterize SVG -> PNG via resvg / usvg / tiny-skia.
//!
//! Authored from the official docs.rs API for resvg 0.47.0 (usvg + fontdb re-exported via
//! resvg::usvg) — an INDEPENDENT, permissively-licensed replacement for
//! github.com/kanrichan/resvg-go (GPL-3.0). No code from that project was used; the
//! ABI and implementation here are our own.
//!
//! The SAME source compiles two ways:
//!   - wasm32-wasip1 `cdylib` (../Cargo.toml) -> resvg.wasm, run on wazero (the default Go backend).
//!     As a WASI *reactor* module the Go/wazero host calls `_initialize` once after instantiation.
//!   - host `staticlib` (../native/Cargo.toml) -> libresvg_native.a, linked via CGo (-tags cgo_resvg).
//!
//! C ABI (handle-based), driven by the host in ../../internal/resvg:
//!   alloc(len) -> ptr ; dealloc(ptr, len)            // linear-memory buffers (align 1)
//!   renderer_new() -> *Renderer ; renderer_free(r)
//!   renderer_load_font(r, ptr, len)                  // register TTF/OTF/TTC bytes
//!   renderer_render(r, svg_ptr, svg_len, out_width, out_height, out_len*) -> png_ptr

use std::alloc::{self, Layout};
use std::ptr;
use std::slice;
use std::sync::Arc;

// resvg 0.47 re-exports tiny_skia, usvg and (via usvg) fontdb, so the types match resvg::render exactly.
use resvg::tiny_skia::{Pixmap, Transform};
use resvg::usvg::{self, fontdb, Options};

/// Opaque handle: the font database used for text-to-path. In resvg 0.47 the fontdb is supplied to the
/// parser via `Options.fontdb` (an `Arc<Database>`), and text is converted to paths during parsing.
pub struct Renderer {
    fontdb: fontdb::Database,
}

/// Allocate `len` bytes in linear memory for the host to write into.
/// Free with `dealloc(ptr, len)` using the SAME len.
#[no_mangle]
pub extern "C" fn alloc(len: usize) -> *mut u8 {
    if len == 0 {
        return ptr::null_mut();
    }
    match Layout::from_size_align(len, 1) {
        Ok(layout) => unsafe { alloc::alloc(layout) },
        Err(_) => ptr::null_mut(),
    }
}

/// Free a buffer returned by `alloc` or `renderer_render`. `len` must match the original.
#[no_mangle]
pub extern "C" fn dealloc(p: *mut u8, len: usize) {
    if p.is_null() || len == 0 {
        return;
    }
    if let Ok(layout) = Layout::from_size_align(len, 1) {
        unsafe { alloc::dealloc(p, layout) }
    }
}

#[no_mangle]
pub extern "C" fn renderer_new() -> *mut Renderer {
    Box::into_raw(Box::new(Renderer {
        fontdb: fontdb::Database::new(),
    }))
}

#[no_mangle]
pub extern "C" fn renderer_free(r: *mut Renderer) {
    if !r.is_null() {
        unsafe { drop(Box::from_raw(r)) }
    }
}

/// Register `len` bytes of font data (TTF/OTF/TTC) into the renderer's font database.
#[no_mangle]
pub extern "C" fn renderer_load_font(r: *mut Renderer, p: *const u8, len: usize) {
    if r.is_null() || p.is_null() || len == 0 {
        return;
    }
    let renderer = unsafe { &mut *r };
    let data = unsafe { slice::from_raw_parts(p, len) }.to_vec();
    renderer.fontdb.load_font_data(data);
}

/// Parse SVG (text -> paths via the renderer's fonts, done inside `Tree::from_data`), rasterize into an
/// `out_width` x `out_height` bitmap (the SVG's natural size when both are 0), encode PNG. Writes the PNG
/// length to `*out_len` and returns the PNG pointer (host reads then frees via `dealloc(ptr, *out_len)`).
/// Returns null on any failure.
#[no_mangle]
pub extern "C" fn renderer_render(
    r: *mut Renderer,
    svg_ptr: *const u8,
    svg_len: usize,
    out_width: u32,
    out_height: u32,
    out_len: *mut usize,
) -> *mut u8 {
    if r.is_null() || svg_ptr.is_null() || svg_len == 0 || out_len.is_null() {
        return ptr::null_mut();
    }
    let renderer = unsafe { &mut *r };
    let svg = unsafe { slice::from_raw_parts(svg_ptr, svg_len) };

    // 1. Parse with our fonts. resvg 0.47 takes the fontdb via Options.fontdb (Arc<Database>) and
    //    converts text to paths during from_data (no separate convert_text step).
    let mut options = Options::default();
    options.fontdb = Arc::new(renderer.fontdb.clone());
    let tree = match usvg::Tree::from_data(svg, &options) {
        Ok(t) => t,
        Err(_) => return ptr::null_mut(),
    };

    // 2. Natural size (f32, now a method). Target = caller's explicit bitmap size (0,0 -> natural, rounded).
    let size = tree.size();
    let nat_w = size.width();
    let nat_h = size.height();
    if !(nat_w > 0.0) || !(nat_h > 0.0) {
        return ptr::null_mut();
    }
    let ow = if out_width == 0 { nat_w.round().max(1.0) as u32 } else { out_width };
    let oh = if out_height == 0 { nat_h.round().max(1.0) as u32 } else { out_height };

    // 3. Render scaled to fill the ow x oh bitmap (resvg::render is a free fn in 0.47). 4. Encode PNG.
    let mut pixmap = match Pixmap::new(ow, oh) {
        Some(pm) => pm,
        None => return ptr::null_mut(),
    };
    let sx = ow as f32 / nat_w;
    let sy = oh as f32 / nat_h;
    resvg::render(&tree, Transform::from_scale(sx, sy), &mut pixmap.as_mut());

    let png = match pixmap.encode_png() {
        Ok(bytes) => bytes,
        Err(_) => return ptr::null_mut(),
    };

    // Copy into an exact-size linear-memory buffer so the host's dealloc(ptr, len) is symmetric.
    let len = png.len();
    let dst = alloc(len);
    if dst.is_null() {
        return ptr::null_mut();
    }
    unsafe {
        ptr::copy_nonoverlapping(png.as_ptr(), dst, len);
        *out_len = len;
    }
    dst
}
