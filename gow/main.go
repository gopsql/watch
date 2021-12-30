package main

import (
	"github.com/gopsql/logger"
	"github.com/gopsql/watch"
)

func main() {
	logger := logger.StandardLogger
	logger.Fatal(watch.NewWatch().WithLogger(logger).Do())
}
