package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gopsql/logger"
	"github.com/gopsql/watch"
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
	var isTest bool
	var goClean bool
	var changeDir string
	var rebuildKeyStr string
	ignore := list{"node_modules", ".git", "dist"}

	flag.StringVar(&goPath, "go", "", "path to the go executable")
	flag.BoolVar(&isTest, "test", false, "run go test instead of go build")
	flag.BoolVar(&goClean, "clean", false, "run go clean -cache or -testcache (if -test) first")
	flag.StringVar(&changeDir, "cd", "", "set working directory of commands")
	flag.StringVar(&rebuildKeyStr, "rebuild-key", "r", "key to rebuild")
	flag.Var(&ignore, "ignore", "add extra directory name to ignore")
	flag.Usage = func() {
		o := flag.CommandLine.Output()
		fmt.Fprintln(o, "Usage:", os.Args[0], "[options] -- [go build/test args]")
		fmt.Fprintln(o)
		fmt.Fprintln(o, "Options:")
		flag.PrintDefaults()
	}
	flag.Parse()

	var rebuildKey byte
	if rebuildKeyStr != "" {
		rebuildKey = rebuildKeyStr[0]
	}

	goBuildArgs := flag.Args()

	var output string
	for i := len(goBuildArgs) - 2; i > -1; i-- {
		if goBuildArgs[i] == "-o" || goBuildArgs[i] == "--o" {
			output = goBuildArgs[i+1]
			break
		}
	}

	watch.NewWatch().
		IgnoreDirectory(ignore...).
		SetTest(isTest).
		SetClean(goClean).
		ChangeDirectory(changeDir).
		WithOutput(output).
		WithGoPath(goPath).
		WithGoBuildArgs(goBuildArgs...).
		WithLogger(logger.StandardLogger).
		WithRebuildKey(rebuildKey).
		MustDo()
}
