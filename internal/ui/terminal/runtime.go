package terminal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/egoist/mygo"
	myterm "github.com/egoist/mygo/plugins/terminal"
	"whaleisle/internal/desktop/terminalruntime"
)

var nativeRuntime struct {
	sync.Mutex
	loaded bool
}

// Verify the exact binding before calling MyGo's loader. A missing packaged DLL
// must not turn opening a terminal into a blocking implicit network download.
// A missing-file error is retryable after staging, without poisoning Load's Once.
func prepareNativeRuntime() error {
	nativeRuntime.Lock()
	defer nativeRuntime.Unlock()
	if nativeRuntime.loaded {
		return nil
	}
	spec, err := terminalruntime.Specification(runtime.GOOS + "-" + runtime.GOARCH)
	if err != nil {
		return err
	}
	var dirs []string
	if dir, err := mygo.App.Path(mygo.PathResources); err == nil {
		dirs = append(dirs, dir)
	}
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	dirs = append(dirs, terminalruntime.CacheDirectory(spec))
	prior, hadOverride := os.LookupEnv("MYGO_GHOSTTY_VT")
	path, err := terminalruntime.Resolve(spec, prior, dirs)
	if err != nil {
		return err
	}
	// Pin this verified path for Load: even a removed file cannot fall through to
	// MyGo's development downloader. Restore the caller's environment afterwards.
	if err := os.Setenv("MYGO_GHOSTTY_VT", path); err != nil {
		return err
	}
	defer func() {
		if hadOverride {
			_ = os.Setenv("MYGO_GHOSTTY_VT", prior)
		} else {
			_ = os.Unsetenv("MYGO_GHOSTTY_VT")
		}
	}()
	if err := myterm.Load(); err != nil {
		return fmt.Errorf("原生终端运行库加载失败，请检查安装包后重启应用：%w", err)
	}
	nativeRuntime.loaded = true
	return nil
}
