package imagepreview

import (
	"bytes"
	"compress/lzw"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
)

// These checks bound the supported containers and declared image dimensions.
// They do not prove the memory behavior or correctness of a downstream codec.
func rasterInput(data []byte) error {
	if len(data) == 0 || int64(len(data)) > MaxBytes {
		return errors.New("image container is empty or exceeds the byte limit")
	}
	return nil
}

func rasterPixels(width, height uint64) (int64, error) {
	if width == 0 || height == 0 || width > uint64(MaxPixels) || height > uint64(MaxPixels)/width {
		return 0, errors.New("image dimensions exceed the pixel limit")
	}
	return int64(width * height), nil
}

// validateGIF walks every descriptor and validates LZW output without retaining
// decoded frames. Plain-text rendering and unknown extensions are unsupported.
func validateGIF(data []byte) error {
	if err := rasterInput(data); err != nil {
		return err
	}
	if len(data) < 13 || (string(data[:6]) != "GIF87a" && string(data[:6]) != "GIF89a") {
		return errors.New("invalid GIF header")
	}
	width := uint64(binary.LittleEndian.Uint16(data[6:8]))
	height := uint64(binary.LittleEndian.Uint16(data[8:10]))
	canvas, err := rasterPixels(width, height)
	if err != nil {
		return err
	}
	pos, globalColors := 13, 0
	if data[10]&0x80 != 0 {
		globalColors = 1 << ((data[10] & 7) + 1)
		if globalColors*3 > len(data)-pos {
			return errors.New("truncated GIF global color table")
		}
		pos += globalColors * 3
		if int(data[11]) >= globalColors {
			return errors.New("invalid GIF background color")
		}
	}
	frames, total := 0, int64(0)
	pendingControl, transparent := false, -1
	for pos < len(data) {
		tag := data[pos]
		pos++
		switch tag {
		case 0x3b:
			if frames == 0 || pendingControl || pos != len(data) {
				return errors.New("invalid GIF trailer")
			}
			return nil
		case 0x21:
			if pos >= len(data) {
				return errors.New("truncated GIF extension")
			}
			label := data[pos]
			pos++
			switch label {
			case 0xf9:
				if pendingControl || len(data)-pos < 6 || data[pos] != 4 || data[pos+5] != 0 {
					return errors.New("invalid GIF graphic control extension")
				}
				packed := data[pos+1]
				if packed&0xe0 != 0 || (packed>>2)&7 > 3 {
					return errors.New("unsupported GIF disposal mode")
				}
				transparent = -1
				if packed&1 != 0 {
					transparent = int(data[pos+4])
				}
				pendingControl = true
				pos += 6
			case 0xff:
				if len(data)-pos < 12 || data[pos] != 11 {
					return errors.New("invalid GIF application extension")
				}
				pos += 12
				pos, _, err = gifSubBlocks(data, pos)
				if err != nil {
					return err
				}
			case 0xfe:
				pos, _, err = gifSubBlocks(data, pos)
				if err != nil {
					return err
				}
			default:
				return errors.New("unsupported GIF extension")
			}
		case 0x2c:
			if len(data)-pos < 9 {
				return errors.New("truncated GIF image descriptor")
			}
			left := uint64(binary.LittleEndian.Uint16(data[pos:]))
			top := uint64(binary.LittleEndian.Uint16(data[pos+2:]))
			w := uint64(binary.LittleEndian.Uint16(data[pos+4:]))
			h := uint64(binary.LittleEndian.Uint16(data[pos+6:]))
			packed := data[pos+8]
			pos += 9
			pixels, err := rasterPixels(w, h)
			if err != nil {
				return err
			}
			if packed&0x18 != 0 || left+w > width || top+h > height {
				return errors.New("invalid GIF frame bounds")
			}
			frames++
			// Budget both encoded rectangles and full-canvas compositing.
			if frames > MaxFrames || total > MaxPixels-pixels || int64(frames) > MaxPixels/canvas {
				return errors.New("GIF animation exceeds the frame or pixel limit")
			}
			total += pixels
			colors := globalColors
			if packed&0x80 != 0 {
				colors = 1 << ((packed & 7) + 1)
				if colors*3 > len(data)-pos {
					return errors.New("truncated GIF local color table")
				}
				pos += colors * 3
			}
			if colors == 0 || transparent >= colors {
				return errors.New("invalid GIF frame palette")
			}
			if pos >= len(data) || data[pos] < 2 || data[pos] > 8 {
				return errors.New("invalid GIF LZW code size")
			}
			codeSize := data[pos]
			pos++
			start := pos
			var compressed int
			pos, compressed, err = gifSubBlocks(data, pos)
			if err != nil {
				return err
			}
			if compressed == 0 {
				return errors.New("empty GIF image data")
			}
			if err := validateGIFPixels(data[start:pos], int(codeSize), pixels, colors); err != nil {
				return err
			}
			pendingControl, transparent = false, -1
		default:
			return errors.New("invalid GIF block")
		}
	}
	return errors.New("missing GIF trailer")
}

func gifSubBlocks(data []byte, pos int) (int, int, error) {
	total := 0
	for pos < len(data) {
		n := int(data[pos])
		pos++
		if n == 0 {
			return pos, total, nil
		}
		if n > len(data)-pos {
			return 0, 0, errors.New("truncated GIF data sub-block")
		}
		total += n
		pos += n
	}
	return 0, 0, errors.New("unterminated GIF data sub-blocks")
}

type gifBlockReader struct {
	data      []byte
	pos       int
	remaining int
	done      bool
}

func (r *gifBlockReader) ReadByte() (byte, error) {
	if r.done {
		return 0, io.EOF
	}
	if r.remaining == 0 {
		if r.pos >= len(r.data) {
			return 0, io.ErrUnexpectedEOF
		}
		r.remaining = int(r.data[r.pos])
		r.pos++
		if r.remaining == 0 {
			r.done = true
			return 0, io.EOF
		}
	}
	if r.pos >= len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	b := r.data[r.pos]
	r.pos++
	r.remaining--
	return b, nil
}

func (r *gifBlockReader) Read(p []byte) (int, error) {
	for i := range p {
		b, err := r.ReadByte()
		if err != nil {
			return i, err
		}
		p[i] = b
	}
	return len(p), nil
}

func validateGIFPixels(blocks []byte, codeSize int, pixels int64, colors int) error {
	src := &gifBlockReader{data: blocks}
	decoder := lzw.NewReader(src, lzw.LSB, codeSize)
	defer decoder.Close()
	var buf [4096]byte
	for pixels > 0 {
		nwant := len(buf)
		if pixels < int64(nwant) {
			nwant = int(pixels)
		}
		n, err := decoder.Read(buf[:nwant])
		for _, index := range buf[:n] {
			if int(index) >= colors {
				return errors.New("GIF pixel references an invalid palette entry")
			}
		}
		pixels -= int64(n)
		if err != nil && !(err == io.EOF && pixels == 0) {
			return fmt.Errorf("invalid GIF LZW data: %w", err)
		}
		if n == 0 {
			return errors.New("truncated GIF pixel data")
		}
	}
	n, err := decoder.Read(buf[:1])
	if n != 0 || err != io.EOF {
		return errors.New("GIF LZW data does not match the frame dimensions")
	}
	if _, err := src.ReadByte(); err != io.EOF {
		return errors.New("unexpected bytes after GIF LZW end code")
	}
	return nil
}

// validateICO supports PNG entries and uncompressed 40-byte Windows DIB
// headers with 1, 4, 8, 24 or 32 bits per pixel. Every entry is checked.
func validateICO(data []byte) error {
	if err := rasterInput(data); err != nil {
		return err
	}
	if len(data) < 6 || binary.LittleEndian.Uint16(data[:2]) != 0 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
		return errors.New("invalid ICO header")
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if count == 0 || count > MaxFrames || count > (len(data)-6)/16 {
		return errors.New("invalid ICO entry count")
	}
	type extent struct{ start, end uint64 }
	ranges := make([]extent, 0, count)
	dirEnd := uint64(6 + count*16)
	var total int64
	for i := 0; i < count; i++ {
		entry := data[6+i*16 : 6+(i+1)*16]
		w, h := uint64(entry[0]), uint64(entry[1])
		if w == 0 {
			w = 256
		}
		if h == 0 {
			h = 256
		}
		planes := binary.LittleEndian.Uint16(entry[4:6])
		if entry[3] != 0 || planes > 1 {
			return errors.New("invalid ICO directory entry")
		}
		size := uint64(binary.LittleEndian.Uint32(entry[8:12]))
		offset := uint64(binary.LittleEndian.Uint32(entry[12:16]))
		if size == 0 || offset < dirEnd || offset > uint64(len(data)) || size > uint64(len(data))-offset {
			return errors.New("ICO entry lies outside its file")
		}
		end := offset + size
		for _, prior := range ranges {
			if offset < prior.end && prior.start < end {
				return errors.New("overlapping ICO entries")
			}
		}
		ranges = append(ranges, extent{offset, end})
		payload := data[int(offset):int(end)]
		var actualW, actualH uint64
		var err error
		if len(payload) >= 8 && bytes.Equal(payload[:8], []byte("\x89PNG\r\n\x1a\n")) {
			actualW, actualH, err = icoPNGDimensions(payload)
		} else {
			actualW, actualH, err = icoDIBDimensions(payload, binary.LittleEndian.Uint16(entry[6:8]))
		}
		if err != nil {
			return fmt.Errorf("invalid ICO entry %d: %w", i+1, err)
		}
		pixels, err := rasterPixels(actualW, actualH)
		if err != nil {
			return err
		}
		if actualW != w || actualH != h || total > MaxPixels-pixels {
			return errors.New("ICO dimensions disagree with the directory or exceed the pixel limit")
		}
		total += pixels
	}
	return nil
}

func icoPNGDimensions(data []byte) (uint64, uint64, error) {
	var width, height uint64
	pos, chunks := 8, 0
	seenHeader, seenData, endedData, seenPalette := false, false, false, false
	nonemptyData := false
	paletteEntries := 0
	metadata := make(map[string]bool, 5)
	var colorType, colorDepth byte
	for pos < len(data) {
		chunks++
		if chunks > 4096 || len(data)-pos < 12 {
			return 0, 0, errors.New("invalid PNG chunk layout")
		}
		size := uint64(binary.BigEndian.Uint32(data[pos:]))
		if size > uint64(len(data)-pos-12) {
			return 0, 0, errors.New("truncated PNG chunk")
		}
		n := int(size)
		kind := string(data[pos+4 : pos+8])
		payload := data[pos+8 : pos+8+n]
		checksum := binary.BigEndian.Uint32(data[pos+8+n : pos+12+n])
		if crc32.ChecksumIEEE(data[pos+4:pos+8+n]) != checksum {
			return 0, 0, errors.New("invalid PNG chunk checksum")
		}
		pos += n + 12
		if !seenHeader && kind != "IHDR" {
			return 0, 0, errors.New("PNG does not begin with IHDR")
		}
		if seenData && kind != "IDAT" {
			endedData = true
		}
		switch kind {
		case "IHDR":
			if seenHeader || n != 13 {
				return 0, 0, errors.New("invalid PNG IHDR")
			}
			width = uint64(binary.BigEndian.Uint32(payload[:4]))
			height = uint64(binary.BigEndian.Uint32(payload[4:8]))
			if _, err := rasterPixels(width, height); err != nil {
				return 0, 0, err
			}
			depth := payload[8]
			colorDepth, colorType = depth, payload[9]
			validDepth := false
			switch colorType {
			case 0:
				validDepth = depth == 1 || depth == 2 || depth == 4 || depth == 8 || depth == 16
			case 2, 4, 6:
				validDepth = depth == 8 || depth == 16
			case 3:
				validDepth = depth == 1 || depth == 2 || depth == 4 || depth == 8
			}
			if !validDepth || payload[10] != 0 || payload[11] != 0 || payload[12] > 1 {
				return 0, 0, errors.New("unsupported PNG encoding")
			}
			seenHeader = true
		case "PLTE":
			if seenPalette || seenData || n == 0 || n > 768 || n%3 != 0 || colorType == 0 || colorType == 4 {
				return 0, 0, errors.New("invalid PNG palette")
			}
			paletteEntries = n / 3
			if colorType == 3 && paletteEntries > 1<<colorDepth {
				return 0, 0, errors.New("PNG palette exceeds its bit depth")
			}
			seenPalette = true
		case "IDAT":
			if endedData || (colorType == 3 && !seenPalette) {
				return 0, 0, errors.New("invalid PNG image data ordering")
			}
			seenData = true
			nonemptyData = nonemptyData || n > 0
		case "IEND":
			if n != 0 || !nonemptyData || pos != len(data) {
				return 0, 0, errors.New("invalid PNG end")
			}
			return width, height, nil
		case "tRNS", "sRGB", "gAMA", "cHRM", "pHYs":
			if seenData || metadata[kind] {
				return 0, 0, errors.New("invalid PNG metadata ordering")
			}
			metadata[kind] = true
			valid := false
			switch kind {
			case "tRNS":
				valid = (colorType == 0 && n == 2) || (colorType == 2 && n == 6) ||
					(colorType == 3 && seenPalette && n > 0 && n <= paletteEntries)
			case "sRGB":
				valid = n == 1 && payload[0] <= 3
			case "gAMA":
				valid = n == 4 && binary.BigEndian.Uint32(payload) != 0
			case "cHRM":
				valid = n == 32
			case "pHYs":
				valid = n == 9 && payload[8] <= 1
			}
			if !valid {
				return 0, 0, errors.New("invalid PNG metadata")
			}
		default:
			// This also excludes APNG and compressed metadata.
			return 0, 0, errors.New("unsupported ICO PNG chunk")
		}
	}
	return 0, 0, errors.New("missing PNG IEND")
}

func icoDIBDimensions(data []byte, directoryBPP uint16) (uint64, uint64, error) {
	if len(data) < 40 || binary.LittleEndian.Uint32(data[:4]) != 40 {
		return 0, 0, errors.New("unsupported ICO DIB header")
	}
	w := int64(int32(binary.LittleEndian.Uint32(data[4:8])))
	doubleH := int64(int32(binary.LittleEndian.Uint32(data[8:12])))
	bpp := binary.LittleEndian.Uint16(data[14:16])
	if w <= 0 || doubleH <= 0 || doubleH%2 != 0 || binary.LittleEndian.Uint16(data[12:14]) != 1 ||
		binary.LittleEndian.Uint32(data[16:20]) != 0 ||
		(bpp != 1 && bpp != 4 && bpp != 8 && bpp != 24 && bpp != 32) ||
		(directoryBPP != 0 && directoryBPP != bpp) {
		return 0, 0, errors.New("unsupported ICO DIB encoding")
	}
	width, height := uint64(w), uint64(doubleH/2)
	if _, err := rasterPixels(width, height); err != nil {
		return 0, 0, err
	}
	colors := uint64(binary.LittleEndian.Uint32(data[32:36]))
	if bpp <= 8 {
		maxColors := uint64(1) << bpp
		if colors == 0 {
			colors = maxColors
		}
		if colors > maxColors {
			return 0, 0, errors.New("invalid ICO DIB palette")
		}
	} else if colors != 0 {
		return 0, 0, errors.New("unsupported ICO DIB color table")
	}
	xorBytes := ((width*uint64(bpp) + 31) / 32) * 4 * height
	maskBytes := ((width + 31) / 32) * 4 * height
	needed := uint64(40) + colors*4 + xorBytes + maskBytes
	declared := uint64(binary.LittleEndian.Uint32(data[20:24]))
	if needed != uint64(len(data)) || (declared != 0 && declared != xorBytes && declared != xorBytes+maskBytes) {
		return 0, 0, errors.New("truncated or ambiguous ICO DIB pixel data")
	}
	return width, height, nil
}

// validateAVIF accepts a single, unprotected av01 primary item in one mdat,
// with one associated ispe and av1C. It intentionally rejects tracks, grids,
// auxiliary images, references, layered properties, external data and idat.
// ispe is a container declaration; AV1 payload memory use is not verified.
func validateAVIF(data []byte) error {
	if err := rasterInput(data); err != nil {
		return err
	}
	budget := 256
	boxes, err := avifBoxes(data, 0, &budget)
	if err != nil {
		return err
	}
	if len(boxes) == 0 || boxes[0].kind != "ftyp" {
		return errors.New("AVIF must begin with ftyp")
	}
	var meta, media avifBox
	haveType, haveMeta, haveMedia := false, false, false
	for _, box := range boxes {
		switch box.kind {
		case "ftyp":
			if haveType || len(box.data) < 8 || (len(box.data)-8)%4 != 0 {
				return errors.New("invalid AVIF file type")
			}
			avifBrand := false
			for i := 0; i < len(box.data); i += 4 {
				if i == 4 {
					continue // minor_version is not a brand
				}
				brand := string(box.data[i : i+4])
				if brand == "avis" || brand == "msf1" {
					return errors.New("animated AVIF is unsupported")
				}
				if brand == "avif" {
					avifBrand = true
				}
			}
			if !avifBrand {
				return errors.New("missing AVIF brand")
			}
			haveType = true
		case "meta":
			if haveMeta || len(box.data) > 1<<20 {
				return errors.New("duplicate or oversized AVIF metadata")
			}
			meta, haveMeta = box, true
		case "mdat":
			if haveMedia || len(box.data) == 0 {
				return errors.New("unsupported AVIF media layout")
			}
			media, haveMedia = box, true
		case "free", "skip":
		default:
			return errors.New("unsupported AVIF top-level box")
		}
	}
	if !haveType || !haveMeta || !haveMedia {
		return errors.New("incomplete AVIF container")
	}
	if len(meta.data) < 4 || binary.BigEndian.Uint32(meta.data[:4]) != 0 {
		return errors.New("unsupported AVIF meta version")
	}
	children, err := avifBoxes(meta.data[4:], meta.start+4, &budget)
	if err != nil {
		return err
	}
	fields := make(map[string]avifBox, 5)
	for _, child := range children {
		switch child.kind {
		case "hdlr", "pitm", "iinf", "iloc", "iprp":
		default:
			return errors.New("unsupported AVIF metadata box")
		}
		if _, duplicate := fields[child.kind]; duplicate {
			return errors.New("duplicate AVIF metadata box")
		}
		fields[child.kind] = child
	}
	if len(fields) != 5 {
		return errors.New("missing AVIF metadata")
	}
	handler := fields["hdlr"].data
	if len(handler) < 24 || len(handler) > 1024 || binary.BigEndian.Uint32(handler[:4]) != 0 ||
		binary.BigEndian.Uint32(handler[4:8]) != 0 || string(handler[8:12]) != "pict" ||
		!avifZero(handler[12:24]) || (len(handler) > 24 && handler[len(handler)-1] != 0) {
		return errors.New("unsupported AVIF handler")
	}
	id, err := avifPrimaryID(fields["pitm"].data)
	if err != nil {
		return err
	}
	if err := avifSingleItem(fields["iinf"], id, &budget); err != nil {
		return err
	}
	if err := avifItemLocation(fields["iloc"].data, id, media); err != nil {
		return err
	}
	return avifProperties(fields["iprp"], id, &budget)
}

type avifBox struct {
	kind  string
	data  []byte
	start uint64 // absolute start of payload, excluding the box header
	end   uint64
}

func avifBoxes(data []byte, base uint64, budget *int) ([]avifBox, error) {
	var boxes []avifBox
	for pos := 0; pos < len(data); {
		if *budget <= 0 || len(data)-pos < 8 {
			return nil, errors.New("AVIF box count or framing is invalid")
		}
		*budget = *budget - 1
		size := uint64(binary.BigEndian.Uint32(data[pos:]))
		kind := string(data[pos+4 : pos+8])
		header := 8
		if size == 1 {
			if len(data)-pos < 16 {
				return nil, errors.New("truncated AVIF large box")
			}
			size = binary.BigEndian.Uint64(data[pos+8:])
			header = 16
		} else if size == 0 {
			if kind != "mdat" {
				return nil, errors.New("unsupported unbounded AVIF box")
			}
			size = uint64(len(data) - pos)
		}
		if size < uint64(header) || size > uint64(len(data)-pos) {
			return nil, errors.New("AVIF box exceeds container bounds")
		}
		end := pos + int(size)
		boxes = append(boxes, avifBox{
			kind: kind, data: data[pos+header : end],
			start: base + uint64(pos+header), end: base + uint64(end),
		})
		pos = end
	}
	return boxes, nil
}

func avifZero(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

func avifPrimaryID(data []byte) (uint32, error) {
	if len(data) < 4 || !avifZero(data[1:4]) {
		return 0, errors.New("invalid AVIF primary item")
	}
	var id uint32
	if data[0] == 0 && len(data) == 6 {
		id = uint32(binary.BigEndian.Uint16(data[4:6]))
	} else if data[0] == 1 && len(data) == 8 {
		id = binary.BigEndian.Uint32(data[4:8])
	} else {
		return 0, errors.New("unsupported AVIF primary item version")
	}
	if id == 0 {
		return 0, errors.New("invalid AVIF primary item ID")
	}
	return id, nil
}

func avifSingleItem(box avifBox, primary uint32, budget *int) error {
	data := box.data
	if len(data) < 6 || !avifZero(data[1:4]) {
		return errors.New("invalid AVIF item information")
	}
	pos := 6
	if data[0] == 0 {
		if binary.BigEndian.Uint16(data[4:6]) != 1 {
			return errors.New("AVIF collections and auxiliary images are unsupported")
		}
	} else if data[0] == 1 && len(data) >= 8 {
		pos = 8
		if binary.BigEndian.Uint32(data[4:8]) != 1 {
			return errors.New("AVIF collections and auxiliary images are unsupported")
		}
	} else {
		return errors.New("unsupported AVIF item information version")
	}
	items, err := avifBoxes(data[pos:], box.start+uint64(pos), budget)
	if err != nil {
		return err
	}
	if len(items) != 1 || items[0].kind != "infe" {
		return errors.New("invalid AVIF item entry")
	}
	entry := items[0].data
	if len(entry) < 13 || len(entry) > 1024 || !avifZero(entry[1:4]) {
		return errors.New("invalid AVIF item entry")
	}
	var id uint32
	pos = 6
	if entry[0] == 2 {
		id = uint32(binary.BigEndian.Uint16(entry[4:6]))
	} else if entry[0] == 3 && len(entry) >= 15 {
		id = binary.BigEndian.Uint32(entry[4:8])
		pos = 8
	} else {
		return errors.New("unsupported AVIF item entry version")
	}
	if id != primary || binary.BigEndian.Uint16(entry[pos:pos+2]) != 0 || string(entry[pos+2:pos+6]) != "av01" {
		return errors.New("AVIF primary item is not one unprotected AV1 image")
	}
	name := entry[pos+6:]
	if len(name) == 0 || name[len(name)-1] != 0 || bytes.IndexByte(name[:len(name)-1], 0) >= 0 {
		return errors.New("invalid AVIF item name")
	}
	return nil
}

type avifCursor struct {
	data []byte
	pos  int
}

func (r *avifCursor) uintN(n int) (uint64, error) {
	if n < 0 || n > 8 || n > len(r.data)-r.pos {
		return 0, errors.New("truncated AVIF item location")
	}
	var value uint64
	for i := 0; i < n; i++ {
		value = value<<8 | uint64(r.data[r.pos])
		r.pos++
	}
	return value, nil
}

func avifItemLocation(data []byte, primary uint32, media avifBox) error {
	if len(data) < 8 || data[0] > 2 || !avifZero(data[1:4]) {
		return errors.New("invalid AVIF item location")
	}
	offsetSize, lengthSize := int(data[4]>>4), int(data[4]&15)
	baseSize, indexSize := int(data[5]>>4), int(data[5]&15)
	validSize := func(n int) bool { return n == 0 || n == 4 || n == 8 }
	if !validSize(offsetSize) || !validSize(baseSize) || (lengthSize != 4 && lengthSize != 8) || indexSize != 0 {
		return errors.New("unsupported AVIF item extent sizes")
	}
	cursor := avifCursor{data: data, pos: 6}
	idSize := 2
	if data[0] == 2 {
		idSize = 4
	}
	count, err := cursor.uintN(idSize)
	if err != nil || count != 1 {
		return errors.New("AVIF must have exactly one located item")
	}
	id, err := cursor.uintN(idSize)
	if err != nil || id != uint64(primary) {
		return errors.New("AVIF item location does not match the primary item")
	}
	if data[0] != 0 {
		construction, err := cursor.uintN(2)
		if err != nil || construction != 0 {
			return errors.New("AVIF constructed item data is unsupported")
		}
	}
	reference, err := cursor.uintN(2)
	if err != nil || reference != 0 {
		return errors.New("external AVIF data is unsupported")
	}
	base, err := cursor.uintN(baseSize)
	if err != nil {
		return err
	}
	extentCount, err := cursor.uintN(2)
	if err != nil || extentCount != 1 {
		return errors.New("AVIF split image extents are unsupported")
	}
	offset, err := cursor.uintN(offsetSize)
	if err != nil {
		return err
	}
	length, err := cursor.uintN(lengthSize)
	if err != nil || cursor.pos != len(data) || length == 0 {
		return errors.New("invalid AVIF item extent")
	}
	// Subtractions precede additions to keep attacker-controlled uint64 values
	// from wrapping even when the iloc uses 64-bit offsets.
	if base > media.end || offset > media.end-base {
		return errors.New("AVIF item offset is outside mdat")
	}
	start := base + offset
	if start < media.start || start > media.end || length > media.end-start {
		return errors.New("AVIF item extent is outside mdat")
	}
	return nil
}

func avifProperties(box avifBox, primary uint32, budget *int) error {
	children, err := avifBoxes(box.data, box.start, budget)
	if err != nil {
		return err
	}
	var propertyBox, associationBox avifBox
	haveProperties, haveAssociations := false, false
	for _, child := range children {
		switch child.kind {
		case "ipco":
			if haveProperties {
				return errors.New("duplicate AVIF property container")
			}
			propertyBox, haveProperties = child, true
		case "ipma":
			if haveAssociations {
				return errors.New("duplicate AVIF property associations")
			}
			associationBox, haveAssociations = child, true
		default:
			return errors.New("unsupported AVIF property container")
		}
	}
	if !haveProperties || !haveAssociations {
		return errors.New("missing AVIF image properties")
	}
	properties, err := avifBoxes(propertyBox.data, propertyBox.start, budget)
	if err != nil {
		return err
	}
	if len(properties) < 2 || len(properties) > 32 {
		return errors.New("invalid AVIF property count")
	}
	kinds := make(map[string]bool, len(properties))
	for _, property := range properties {
		if kinds[property.kind] {
			return errors.New("ambiguous duplicate AVIF image property")
		}
		kinds[property.kind] = true
		p := property.data
		valid := false
		switch property.kind {
		case "ispe":
			if len(p) != 12 || binary.BigEndian.Uint32(p[:4]) != 0 {
				return errors.New("invalid AVIF spatial extents")
			}
			if _, err := rasterPixels(uint64(binary.BigEndian.Uint32(p[4:8])), uint64(binary.BigEndian.Uint32(p[8:12]))); err != nil {
				return err
			}
			valid = true
		case "av1C":
			// Configuration OBUs and presentation-delay declarations are
			// excluded from this deliberately narrow still-image subset.
			valid = len(p) == 4 && p[0] == 0x81 && p[1]>>5 <= 2 && p[3] == 0
		case "pixi":
			if len(p) >= 5 && binary.BigEndian.Uint32(p[:4]) == 0 && (p[4] == 1 || p[4] == 3) && len(p) == 5+int(p[4]) {
				valid = true
				for _, depth := range p[5:] {
					if depth != 8 && depth != 10 && depth != 12 {
						valid = false
					}
				}
			}
		case "colr":
			valid = len(p) == 11 && string(p[:4]) == "nclx" && p[10]&0x7f == 0
		case "pasp":
			valid = len(p) == 8 && binary.BigEndian.Uint32(p[:4]) != 0 && binary.BigEndian.Uint32(p[4:8]) != 0
		case "irot":
			valid = len(p) == 1 && p[0]&0xfc == 0
		case "imir":
			valid = len(p) == 1 && p[0]&0xfe == 0
		case "clli":
			valid = len(p) == 4
		case "mdcv":
			valid = len(p) == 24
		default:
			return errors.New("unsupported AVIF image property")
		}
		if !valid {
			return errors.New("malformed or unsupported AVIF image property")
		}
	}
	if !kinds["ispe"] || !kinds["av1C"] {
		return errors.New("missing AVIF dimensions or codec configuration")
	}
	return avifAssociations(associationBox.data, primary, len(properties))
}

func avifAssociations(data []byte, primary uint32, propertyCount int) error {
	if len(data) < 8 || data[0] > 1 || data[1] != 0 || data[2] != 0 || data[3] > 1 || binary.BigEndian.Uint32(data[4:8]) != 1 {
		return errors.New("invalid AVIF property associations")
	}
	cursor := avifCursor{data: data, pos: 8}
	idSize := 2
	if data[0] == 1 {
		idSize = 4
	}
	id, err := cursor.uintN(idSize)
	if err != nil || id != uint64(primary) {
		return errors.New("AVIF properties do not belong to the primary image")
	}
	count, err := cursor.uintN(1)
	if err != nil || count != uint64(propertyCount) {
		return errors.New("unassociated or ambiguous AVIF image properties")
	}
	seen := make([]bool, propertyCount+1)
	size, mask := 1, uint64(0x7f)
	if data[3]&1 != 0 {
		size, mask = 2, 0x7fff
	}
	for i := uint64(0); i < count; i++ {
		association, err := cursor.uintN(size)
		if err != nil {
			return err
		}
		index := association & mask
		if index == 0 || index > uint64(propertyCount) || seen[int(index)] {
			return errors.New("invalid or duplicate AVIF property association")
		}
		seen[int(index)] = true
	}
	if cursor.pos != len(data) {
		return errors.New("trailing AVIF property association data")
	}
	return nil
}
