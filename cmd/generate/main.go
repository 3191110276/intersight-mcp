package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

func main() {
	var (
		inPath      string
		filterPath  string
		metricsPath string
		outPath     string
	)

	flag.StringVar(&inPath, "in", "", "path to the pinned raw OpenAPI input")
	flag.StringVar(&filterPath, "filter", "", "path to the committed filter policy")
	flag.StringVar(&metricsPath, "metrics", "", "path to the committed metrics catalog input")
	flag.StringVar(&outPath, "out", "", "path to the resolved spec output")
	flag.Parse()

	if err := run(runConfig{
		inPath:      inPath,
		filterPath:  filterPath,
		metricsPath: metricsPath,
		outPath:     outPath,
		stdout:      os.Stdout,
		stderr:      os.Stderr,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "generate: %v\n", err)
		os.Exit(1)
	}
}

type runConfig struct {
	inPath      string
	filterPath  string
	metricsPath string
	outPath     string
	stdout      *os.File
	stderr      *os.File
}

func run(cfg runConfig) error {
	if cfg.inPath == "" || cfg.filterPath == "" || cfg.metricsPath == "" || cfg.outPath == "" {
		return errors.New("all of --in, --filter, --metrics, and --out are required")
	}
	gen := newGenerator(cfg.inPath, cfg.filterPath, cfg.metricsPath, cfg.outPath, cfg.stdout, cfg.stderr)
	return gen.Run()
}
