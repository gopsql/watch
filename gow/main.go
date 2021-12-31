package main

import (
	"flag"
	"fmt"
	"io"
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
	var rebuildKeyStr string
	ignore := list{"node_modules", ".git", "dist"}

	flag.StringVar(&goPath, "go", "", "path to the go executable")
	flag.StringVar(&rebuildKeyStr, "rebuild-key", "r", "key to rebuild")
	flag.Var(&ignore, "ignore", "add extra directory name to ignore")
	flag.Usage = func() {
		o := flag.CommandLine.Output()
		fmt.Fprintln(o, "Usage:", os.Args[0], "[options] -- [go build args]")
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
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&output, "o", "", "")
	fs.Parse(goBuildArgs)

	watch.NewWatch().
		IgnoreDirectory(ignore...).
		WithOutput(output).
		WithGoPath(goPath).
		WithGoBuildArgs(goBuildArgs...).
		WithLogger(logger.StandardLogger).
		WithRebuildKey(rebuildKey).
		MustDo()
}
