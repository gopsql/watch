package watch

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
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
	goBuildArgs []string      // extra arguments to go build or go test
	isTest      bool          // true to run go test instead of go build
	cleanFirst  bool          // run go clean command before go build or go test
	logger      logger.Logger // no logger by default
	ignoreDirs  []string      // list of directories not to watch
	rebuildKey  byte          // key to enter to run go build or go test again

	workingDir string // working directory for commands
	directory  string // directory to watch
	output     string // path to output file
}

// NewWatch creates new watch instance, watches go files recursively in current
// directory. If any .go and .mod files changed, executes the go build command
// and then the newly built executable.
func NewWatch() *watch {
	return &watch{}
}

// IgnoreDirectory adds directory name to directory ignore list. Ignore
// directories without go files could reduce CPU usage.
func (w *watch) IgnoreDirectory(dirs ...string) *watch {
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		w.ignoreDirs = appendStringIfMissing(w.ignoreDirs, dir)
	}
	return w
}

// Set to true to run "go test" instead of "go build".
func (w *watch) SetTest(isTest bool) *watch {
	w.isTest = isTest
	return w
}

// Set to true to run "go clean" before go build or go test.
func (w *watch) SetClean(clean bool) *watch {
	w.cleanFirst = clean
	return w
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

// ChangeDirectory changes the working directory of commands. Default is
// current process's current directory.
func (w *watch) ChangeDirectory(dir string) *watch {
	w.workingDir = dir
	return w
}

// WithLogger sets the logger.
func (w *watch) WithLogger(logger logger.Logger) *watch {
	w.logger = logger
	return w
}

// WithRebuildKey sets rebuild key.
func (w *watch) WithRebuildKey(key byte) *watch {
	w.rebuildKey = key
	return w
}

// MustDo is like Do, but panics if operation fails.
func (w *watch) MustDo() {
	if w.logger != nil {
		w.logger.Fatal(w.Do())
	} else {
		panic(w.Do())
	}
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
	app.SetDir(w.workingDir)
	app.SetWriter(os.Stdout)

	goPath := w.goPath
	if goPath == "" {
		goPath = "go"
	}

	var cleanArgs []string
	if w.isTest {
		cleanArgs = []string{"clean", "-testcache"}
	} else {
		cleanArgs = []string{"clean", "-cache"}
	}
	clean := newRunner(goPath, cleanArgs...)
	clean.SetDir(w.workingDir)
	clean.SetWriter(os.Stdout)

	var args []string
	if w.isTest {
		args = append([]string{"test"}, w.goBuildArgs...)
	} else {
		args = append([]string{"build", "-o", output}, w.goBuildArgs...)
	}
	build := newRunner(goPath, args...)
	build.SetDir(w.workingDir)
	build.SetWriter(os.Stdout)

	tidy := newRunner(goPath, "mod", "tidy")
	tidy.SetWriter(os.Stdout)

	dirsToIgnore := dirsWithName(directory, w.ignoreDirs...)

	wa := watcher.New()
	wa.SetMaxEvents(1)
	wa.Ignore(output) // prevent endless loop
	wa.AddFilterHook(func(info os.FileInfo, fullPath string) error {
		for _, dir := range dirsToIgnore {
			if fullPath == dir {
				return filepath.SkipDir
			}
		}
		if strings.HasSuffix(fullPath, ".go") || strings.HasSuffix(fullPath, ".mod") {
			return nil
		}
		return watcher.ErrSkip // stop processing
	})
	if err := wa.AddRecursive(directory); err != nil {
		return err
	}

	if w.logger != nil {
		w.logger.Info("Watching", logger.CyanString(strconv.Itoa(len(wa.WatchedFiles()))), "files")
	}

	if w.rebuildKey > 0 {
		if w.logger != nil {
			var action string
			if w.isTest {
				action = "to retest"
			} else {
				action = "to rebuild"
			}
			w.logger.Info("Enter", logger.CyanString([]byte{w.rebuildKey}), action)
		}
		go func() {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				b := scanner.Bytes()
				if len(b) == 1 && b[0] == w.rebuildKey {
					wa.TriggerEvent(watcher.Create, nil)
				}
			}
		}()
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
			if w.cleanFirst {
				w.logger.Info(logger.CyanString("Cleaning..."))
				clean.Run(true)
			}
			if w.logger != nil {
				if w.isTest {
					w.logger.Info(logger.CyanString("Testing..."))
				} else {
					w.logger.Info(logger.CyanString("Building..."))
				}
			}
			begin := time.Now()
			if build.Run(true) == nil {
				spent := time.Now().Sub(begin).Truncate(time.Millisecond)
				if w.logger != nil {
					var action string
					if w.isTest {
						action = "Test"
					} else {
						action = "Build"
					}
					w.logger.Info(logger.GreenBoldString(fmt.Sprintf("%s finished (%s)", action, spent)))
				}
				if w.isTest == false {
					app.Run(false)
				}
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

func appendStringIfMissing(slice []string, element string) []string {
	for _, e := range slice {
		if e == element {
			return slice
		}
	}
	return append(slice, element)
}

func dirsWithName(root string, names ...string) (dirs []string) {
	if len(names) == 0 {
		return
	}
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		for _, name := range names {
			if d.Name() == name {
				dirs = append(dirs, path)
				return filepath.SkipDir
			}
		}
		return nil
	})
	return
}
