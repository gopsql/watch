package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gopsql/logger"
	"github.com/gopsql/watch"
	"github.com/mattn/go-shellwords"
)

type list []string

func (s list) String() string {
	return strings.Join(s, ", ")
}

func (s *list) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	var goPath string
	var noRun bool
	var isTest bool
	var goClean bool
	var prebuildStr string
	var changeDir string
	var rebuildKeyStr string
	ignore := list{"node_modules", ".git", "dist"}
	exts := list{".go", ".mod"}

	flag.StringVar(&goPath, "go", "", "path to the go executable")
	flag.BoolVar(&noRun, "no-run", false, "do not run the executable after go build")
	flag.BoolVar(&isTest, "test", false, "run go test instead of go build")
	flag.BoolVar(&goClean, "clean", false, "run go clean -cache or -testcache (if -test) first")
	flag.StringVar(&prebuildStr, "prebuild", "", "run command before go build or go test")
	flag.StringVar(&changeDir, "cd", "", "set working directory of commands")
	flag.StringVar(&rebuildKeyStr, "rebuild-key", "r", "key to rebuild")
	flag.Var(&ignore, "ignore", "add extra directory name to ignore")
	flag.Var(&exts, "ext", "add extra file extensions to watch")
	flag.Usage = func() {
		o := flag.CommandLine.Output()
		fmt.Fprintln(o, "Usage:", os.Args[0], "[options] -- [go build/test args] -- [app run args]")
		fmt.Fprintln(o)
		fmt.Fprintln(o, "Options:")
		flag.PrintDefaults()
	}
	flag.Parse()

	var rebuildKey byte
	if rebuildKeyStr != "" {
		rebuildKey = rebuildKeyStr[0]
	}

	var goBuildArgs []string
	var appRunArgs []string

	restArgs := flag.Args()
	argParts := 0
	for _, arg := range restArgs {
		if arg == "--" {
			argParts += 1
			continue
		}
		switch argParts {
		case 0:
			goBuildArgs = append(goBuildArgs, arg)
		case 1:
			appRunArgs = append(appRunArgs, arg)
		}
	}

	var prebuild []string
	if prebuildStr != "" {
		var err error
		prebuild, err = shellwords.Parse(prebuildStr)
		if err != nil {
			log.Fatalln(err)
		}
	}

	var output string
	for i := len(goBuildArgs) - 2; i > -1; i-- {
		if goBuildArgs[i] == "-o" || goBuildArgs[i] == "--o" {
			output = goBuildArgs[i+1]
			break
		}
	}

	watch.NewWatch().
		IgnoreDirectory(ignore...).
		SetNoRun(noRun).
		SetTest(isTest).
		SetClean(goClean).
		SetPrebuild(prebuild...).
		ChangeDirectory(changeDir).
		WithOutput(output).
		WithGoPath(goPath).
		WithGoBuildArgs(goBuildArgs...).
		WithAppRunArgs(appRunArgs...).
		WithLogger(logger.StandardLogger).
		WithRebuildKey(rebuildKey).
		WithFileExts(exts...).
		MustDo()
}
