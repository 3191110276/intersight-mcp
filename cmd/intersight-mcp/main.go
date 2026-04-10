package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mimaurer/intersight-mcp/generated"
	internalpkg "github.com/mimaurer/intersight-mcp/internal"
	"github.com/mimaurer/intersight-mcp/internal/config"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"github.com/mimaurer/intersight-mcp/intersight"
	"github.com/mimaurer/intersight-mcp/sandbox"
	"github.com/mimaurer/intersight-mcp/server"
)

var version = "dev"

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
	fmt.Fprintln(os.Stderr, "usage: intersight-mcp serve [flags]")
}

func serve(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return serveWithIO(ctx, args, os.Stdin, os.Stdout, os.Stderr, os.Environ(), generated.ResolvedSpecBytes(), generated.SDKCatalogBytes(), generated.RulesBytes(), generated.SearchCatalogBytes())
}

func serveWithIO(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer, environ []string, specBytes, sdkCatalogBytes, rulesBytes, searchCatalogBytes []byte) error {
	cfg, err := config.Load(args, environ)
	if err != nil {
		return err
	}
	if err := server.ValidateEmbeddedArtifacts(specBytes, sdkCatalogBytes, rulesBytes, searchCatalogBytes); err != nil {
		return err
	}
	artifacts, err := sandbox.LoadArtifactBundle(specBytes, sdkCatalogBytes, rulesBytes, searchCatalogBytes)
	if err != nil {
		return err
	}

	logger := internalpkg.NewLogger(stderr, cfg.LogLevel)
	httpClient := &http.Client{}

	oauthManager, err := intersight.NewOAuthManager(ctx, intersight.OAuthConfig{
		TokenURL:     cfg.OAuthTokenURL,
		ValidateURL:  cfg.APIBaseURL + "/iam/UserPreferences",
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		HTTPClient:   httpClient,
	})
	if err != nil {
		return err
	}

	apiClient := intersight.NewClient(httpClient, cfg.APIBaseURL, oauthManager)
	sandboxCfg := sandbox.Config{
		SearchTimeout:   cfg.SearchTimeout,
		GlobalTimeout:   cfg.Execution.GlobalTimeout,
		PerCallTimeout:  cfg.PerCallTimeout,
		MaxCodeSize:     cfg.MaxCodeSize,
		MaxAPICalls:     cfg.Execution.MaxAPICalls,
		MaxOutputBytes:  cfg.Execution.MaxOutputBytes,
		WASMMemoryBytes: int(cfg.WASMMemory),
	}

	searchExec, err := sandbox.NewSearchExecutorFromBundle(sandboxCfg, artifacts)
	if err != nil {
		return err
	}

	queryExec, err := sandbox.NewQJSExecutorFromBundle(sandboxCfg, sandboxClient{client: apiClient}, artifacts)
	if err != nil {
		_ = searchExec.Close()
		return err
	}
	mutateExec, err := sandbox.NewQJSExecutorFromBundle(sandboxCfg, sandboxClient{client: apiClient}, artifacts)
	if err != nil {
		_ = searchExec.Close()
		_ = queryExec.Close()
		return err
	}

	runtime, err := server.NewRuntime(server.RuntimeConfig{
		ServerName:     "intersight-mcp",
		ServerVersion:  version,
		MaxConcurrent:  cfg.Execution.MaxConcurrent,
		Logger:         logger,
		SearchExecutor: searchExec,
		QueryExecutor:  queryExec,
		MutateExecutor: mutateExec,
	})
	if err != nil {
		_ = searchExec.Close()
		_ = queryExec.Close()
		_ = mutateExec.Close()
		return err
	}
	defer runtime.Close()

	return runtime.Listen(ctx, stdin, stdout)
}

type sandboxClient struct {
	client *intersight.Client
}

func (c sandboxClient) Do(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
	return c.client.Do(ctx, operation)
}
