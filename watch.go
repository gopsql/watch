package watch

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gopsql/logger"
	"github.com/radovskyb/watcher"
)

var (
	opts = map[watcher.Op]string{
		watcher.Create: "created",
		watcher.Write:  "updated",
		watcher.Remove: "deleted",
		watcher.Rename: "renamed",
		watcher.Chmod:  "permission changed",
		watcher.Move:   "moved",
	}
)

type watch struct {
	goPath      string        // defaults to "go"
	goBuildArgs []string      // extra arguments to go build
	logger      logger.Logger // no logger by default

	directory string
	output    string
}

// NewWatch creates new watch instance, watches go files recursively in current
// directory. If any .go and .mod files changed, executes the go build command
// and then the newly built executable.
func NewWatch() *watch {
	return &watch{}
}

// InDirectory sets different directory to watch other than current directory.
func (w *watch) InDirectory(directory string) *watch {
	w.directory = directory
	return w
}

// WithOutput sets different output file name of go build. Defaults to the name
// of current directory.
func (w *watch) WithOutput(output string) *watch {
	w.output = output
	return w
}

// WithGoPath sets path to the "go" executable.
func (w *watch) WithGoPath(path string) *watch {
	w.goPath = path
	return w
}

// WithGoBuildArgs sets extra command line arguments of go build.
func (w *watch) WithGoBuildArgs(args ...string) *watch {
	w.goBuildArgs = args
	return w
}

// WithLogger sets the logger.
func (w *watch) WithLogger(logger logger.Logger) *watch {
	w.logger = logger
	return w
}

// Do starts the watch process.
func (w *watch) Do() error {
	directory, err := filepath.Abs(w.directory)
	if err != nil {
		return err
	}

	output := w.output
	if output == "" {
		output = defaultExecName(directory)
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return err
	}

	app := newRunner(output)
	app.SetDir(filepath.Dir(output))
	app.SetWriter(os.Stdout)

	goPath := w.goPath
	if goPath == "" {
		goPath = "go"
	}
	args := append([]string{"build", "-o", output}, w.goBuildArgs...)
	build := newRunner(goPath, args...)
	build.SetDir(filepath.Dir(output))
	build.SetWriter(os.Stdout)

	tidy := newRunner(goPath, "mod", "tidy")
	tidy.SetWriter(os.Stdout)

	wa := watcher.New()
	wa.SetMaxEvents(1)
	wa.AddFilterHook(func(info os.FileInfo, fullPath string) error {
		if strings.HasSuffix(fullPath, ".go") || strings.HasSuffix(fullPath, ".mod") {
			return nil
		}
		return watcher.ErrSkip
	})
	if err := wa.AddRecursive(directory); err != nil {
		return err
	}

	go wa.TriggerEvent(watcher.Create, nil)

	go func() {
		if err := wa.Start(200 * time.Millisecond); err != nil {
			wa.Error <- err
		}
	}()

	modFileTime := map[string]time.Time{}
	for {
		select {
		case event := <-wa.Event:
			if w.logger != nil && (event.Path != "" && event.Path != "-") {
				base, _ := filepath.Abs(".")
				oldPath, _ := filepath.Rel(base, event.OldPath)
				path, _ := filepath.Rel(base, event.Path)
				if path == "" || strings.HasPrefix(path, "../") {
					oldPath = event.OldPath
					path = event.Path
				}
				if event.Op == watcher.Rename || event.Op == watcher.Move {
					w.logger.Info("File", logger.CyanString(oldPath), opts[event.Op], "to", logger.CyanString(path))
				} else {
					w.logger.Info("File", logger.CyanString(path), opts[event.Op])
				}
			}
			if strings.HasSuffix(event.Path, ".mod") {
				before, err := os.Stat(event.Path)
				if err != nil || before.ModTime() == modFileTime[event.Path] {
					continue
				}
				if w.logger != nil {
					w.logger.Info(logger.CyanString("Running go mod tidy..."))
				}
				tidy.SetDir(filepath.Dir(event.Path))
				tidy.Run(true)
				after, err := os.Stat(event.Path)
				if err != nil {
					continue
				}
				modFileTime[event.Path] = after.ModTime()
			}
			app.Kill()
			if w.logger != nil {
				w.logger.Info(logger.CyanString("Building..."))
			}
			if build.Run(true) == nil {
				if w.logger != nil {
					w.logger.Info(logger.GreenBoldString("Build finished"))
				}
				app.Run(false)
			}
		case err := <-wa.Error:
			return err
		case <-wa.Closed:
			return nil
		}
	}
	return nil
}

func defaultExecName(p string) string {
	_, elem := path.Split(p)
	if isVersionElement(elem) {
		_, elem = path.Split(path.Dir(p))
	}
	return elem
}

// from go/src/cmd/go/internal/load/pkg.go
func isVersionElement(s string) bool {
	if len(s) < 2 || s[0] != 'v' || s[1] == '0' || s[1] == '1' && len(s) == 2 {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] < '0' || '9' < s[i] {
			return false
		}
	}
	return true
}
