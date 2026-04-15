package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/mimaurer/intersight-mcp/implementations"
	_ "github.com/mimaurer/intersight-mcp/implementations/all"
)

func main() {
	var (
		provider string
	)

	flag.StringVar(&provider, "provider", "intersight", "provider name")
	flag.Parse()

	if err := run(runConfig{
		provider: provider,
		stdout:   os.Stdout,
		stderr:   os.Stderr,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "generate: %v\n", err)
		os.Exit(1)
	}
}

type runConfig struct {
	provider string
	stdout   *os.File
	stderr   *os.File
}

func run(cfg runConfig) error {
	target, err := implementations.LookupTarget(cfg.provider)
	if err != nil {
		return err
	}
	generation := target.GenerationConfig()
	if generation.RawSpecPath == "" || generation.ManifestPath == "" || generation.OutputPath == "" {
		return errors.New("provider generation config requires raw spec, manifest, and output paths")
	}
	gen := newGenerator(
		generation.RawSpecPath,
		generation.ManifestPath,
		generation.FilterPath,
		generation.MetricsPath,
		generation.OutputPath,
		generation.FallbackPathPrefixes,
		generation.RuleTemplates,
		generation.SchemaNormalizationHook,
		cfg.stdout,
		cfg.stderr,
	)
	return gen.Run()
}
