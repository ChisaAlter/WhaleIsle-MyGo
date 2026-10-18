package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Cache entries describe verified complete installers, including installers
// reconstructed by a delta. The installed executable is never used as a base.
func nativeCachedBase(release NativeRelease, currentVersion, cacheDir string, verify bool) (string, error) {
	d := release.Delta
	if !release.IsDelta() || d == nil {
		return "", fmt.Errorf("selected package has no native delta recipe")
	}
	if currentVersion != d.BaseVersion && currentVersion != "v"+d.BaseVersion && currentVersion != "mygo-v"+d.BaseVersion {
		return "", fmt.Errorf("delta requires installed version %s", d.BaseVersion)
	}
	if cacheDir == "" {
		return "", fmt.Errorf("no verified native installer cache")
	}
	digest := strings.ToLower(d.BaseSHA256)
	raw, err := os.ReadFile(filepath.Join(cacheDir, digest+".json"))
	if err != nil {
		return "", fmt.Errorf("matching base installer is not cached")
	}
	var base NativeRelease
	if json.Unmarshal(raw, &base) != nil || base.IsDelta() || base.Validate(release.Platform, release.Arch, false) != nil || base.PackageKey() != d.BasePackageID || base.Version != d.BaseVersion || !strings.EqualFold(base.SHA256, d.BaseSHA256) || base.Size != d.BaseSize || base.RuntimeIncluded != release.RuntimeIncluded {
		return "", fmt.Errorf("cached installer identity does not match the delta base")
	}
	file := filepath.Join(cacheDir, digest+".exe")
	st, err := os.Stat(file)
	if err != nil || !st.Mode().IsRegular() || st.Size() != d.BaseSize {
		return "", fmt.Errorf("matching base installer bytes are unavailable")
	}
	if verify {
		sum, size, err := hashFileBounded(file, NativeMaxInstallerBytes)
		if err != nil {
			return "", err
		}
		if sum != digest || size != d.BaseSize {
			return "", fmt.Errorf("cached base installer SHA-256/size changed")
		}
	}
	return file, nil
}

type nativeCacheWriter struct {
	ctx    context.Context
	writer io.Writer
}

func (w nativeCacheWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	return w.writer.Write(p)
}

func cacheNativeInstaller(ctx context.Context, release NativeRelease, result ApplyResult, cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}
	base := release
	base.PackageKind = "full"
	base.Delta = nil
	base.PackageID = base.FullPackageKey()
	base.BlockMap, base.BlockMapURL = nil, ""
	digest := strings.ToLower(release.SHA256)
	suffix := ".exe"
	if release.Format == "dmg" {
		suffix = ".dmg"
	}
	dest := filepath.Join(cacheDir, digest+suffix)
	if sum, size, err := hashFileBounded(dest, NativeMaxInstallerBytes); err != nil || sum != digest || size != release.Size {
		in, err := os.Open(result.Path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.CreateTemp(cacheDir, "installer-*.part")
		if err != nil {
			return err
		}
		defer os.Remove(out.Name())
		h := sha256.New()
		n, copyErr := io.Copy(nativeCacheWriter{ctx, io.MultiWriter(out, h)}, io.LimitReader(in, release.Size+1))
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if n != release.Size || hex.EncodeToString(h.Sum(nil)) != digest {
			return fmt.Errorf("installer changed while saving the native base cache")
		}
		if err := os.Rename(out.Name(), dest); err != nil {
			return err
		}
	}
	raw, err := json.Marshal(base)
	if err != nil {
		return err
	}
	meta, err := os.CreateTemp(cacheDir, "identity-*.part")
	if err != nil {
		return err
	}
	defer os.Remove(meta.Name())
	_, writeErr := meta.Write(raw)
	closeErr := meta.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(meta.Name(), filepath.Join(cacheDir, digest+".json"))
}
