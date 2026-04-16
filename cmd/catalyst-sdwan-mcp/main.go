package main

import (
	"fmt"
	"os"

	"github.com/mimaurer/intersight-mcp/implementations"
	_ "github.com/mimaurer/intersight-mcp/implementations/catalyst-sdwan"
	"github.com/mimaurer/intersight-mcp/internal/bootstrap"
)

var version = "dev"

var app = bootstrap.App{
	Target:  implementations.MustLookupTarget("catalyst-sdwan"),
	Version: version,
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "serve":
		if err := serve(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "serve: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	app.Usage(os.Stderr)
}

func serve(args []string) error {
	return app.Serve(args)
}
