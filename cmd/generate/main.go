package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

func main() {
	var (
		inPath     string
		filterPath string
		outPath    string
	)

	flag.StringVar(&inPath, "in", "", "path to the pinned raw OpenAPI input")
	flag.StringVar(&filterPath, "filter", "", "path to the committed filter policy")
	flag.StringVar(&outPath, "out", "", "path to the resolved spec output")
	flag.Parse()

	if err := run(runConfig{
		inPath:     inPath,
		filterPath: filterPath,
		outPath:    outPath,
		stdout:     os.Stdout,
		stderr:     os.Stderr,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "generate: %v\n", err)
		os.Exit(1)
	}
}

type runConfig struct {
	inPath     string
	filterPath string
	outPath    string
	stdout     *os.File
	stderr     *os.File
}

func run(cfg runConfig) error {
	if cfg.inPath == "" || cfg.filterPath == "" || cfg.outPath == "" {
		return errors.New("all of --in, --filter, and --out are required")
	}
	gen := newGenerator(cfg.inPath, cfg.filterPath, cfg.outPath, cfg.stdout, cfg.stderr)
	return gen.Run()
}
