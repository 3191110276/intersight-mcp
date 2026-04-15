package bootstrap

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mimaurer/intersight-mcp/implementations"
	internalpkg "github.com/mimaurer/intersight-mcp/internal"
	"github.com/mimaurer/intersight-mcp/internal/config"
	"github.com/mimaurer/intersight-mcp/sandbox"
	"github.com/mimaurer/intersight-mcp/server"
	"github.com/mimaurer/intersight-mcp/tools"
)

type App struct {
	Target  implementations.Target
	Version string
}

func (a App) Usage(w io.Writer) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "usage: %s serve [flags]\n", a.Target.RuntimeMetadata().ServerName)
}

func (a App) Serve(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return a.ServeWithIO(ctx, args, os.Stdin, os.Stdout, os.Stderr, os.Environ())
}

func (a App) ServeWithIO(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer, environ []string) error {
	return a.ServeWithIOAndHTTPClient(ctx, args, stdin, stdout, stderr, environ, nil)
}

func (a App) ServeWithIOAndHTTPClient(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer, environ []string, httpClient *http.Client) error {
	runtimeMeta := a.Target.RuntimeMetadata()

	runtimeCfg, err := config.LoadRuntime(args, environ, runtimeMeta.ConfigPrefix)
	if err != nil {
		return err
	}
	connectionCfg, err := a.Target.LoadConnectionConfig(args, environ)
	if err != nil {
		return err
	}
	artifacts := a.Target.Artifacts()
	artifactBundle, err := sandbox.LoadArtifactBundleWithExtensions(artifacts.ResolvedSpec, artifacts.SDKCatalog, artifacts.Rules, artifacts.SearchCatalog, a.Target.SandboxExtensions())
	if err != nil {
		return err
	}

	logger := internalpkg.NewLogger(stderr, runtimeCfg.LogLevel, runtimeCfg.UnsafeLogFullCode, internalpkg.LoggerOptions{
		Redactions: runtimeMeta.Logging.Redactions,
	})
	if runtimeCfg.LogLevel == config.LogLevelDebug && runtimeCfg.UnsafeLogFullCode {
		logger.LogServerMessage(context.Background(), "config", "unsafe full-code debug logging is enabled; submitted tool code may be written to logs with best-effort redaction. Use only for short-lived incident debugging on trusted machines.")
	}
	if httpClient == nil {
		httpClient, err = NewHTTPClient(runtimeCfg.PerCallTimeout, connectionCfg.ProxyURL())
		if err != nil {
			return err
		}
	}
	sandboxCfg := sandbox.Config{
		SearchTimeout:   runtimeCfg.SearchTimeout,
		GlobalTimeout:   runtimeCfg.Execution.GlobalTimeout,
		PerCallTimeout:  runtimeCfg.PerCallTimeout,
		MaxCodeSize:     runtimeCfg.MaxCodeSize,
		MaxAPICalls:     runtimeCfg.Execution.MaxAPICalls,
		MaxOutputBytes:  runtimeCfg.Execution.MaxOutputBytes,
		WASMMemoryBytes: int(runtimeCfg.WASMMemory),
	}

	searchExec, err := sandbox.NewSearchExecutorFromBundle(sandboxCfg, artifactBundle)
	if err != nil {
		return err
	}

	apiCaller := connectionCfg.NewAPICaller(ctx, runtimeCfg.PerCallTimeout, httpClient)

	queryExec, err := sandbox.NewQJSExecutorFromBundle(sandboxCfg, apiCaller, artifactBundle)
	if err != nil {
		_ = searchExec.Close()
		return err
	}

	var mutateExec sandbox.Executor
	if !runtimeCfg.ReadOnly {
		mutateExec, err = sandbox.NewQJSExecutorFromBundle(sandboxCfg, apiCaller, artifactBundle)
		if err != nil {
			_ = searchExec.Close()
			_ = queryExec.Close()
			return err
		}
	}

	runtime, err := server.NewRuntime(server.RuntimeConfig{
		ServerName:     runtimeMeta.ServerName,
		ServerVersion:  a.Version,
		MaxCodeSize:    runtimeCfg.MaxCodeSize,
		MaxConcurrent:  runtimeCfg.Execution.MaxConcurrent,
		MaxOutputBytes: runtimeCfg.Execution.MaxOutputBytes,
		ReadOnly:       runtimeCfg.ReadOnly,
		ToolMetadata: tools.ToolMetadata{
			SearchTitle:       runtimeMeta.ToolDescriptions.SearchTitle,
			SearchDescription: runtimeMeta.ToolDescriptions.SearchDescription,
			QueryTitle:        runtimeMeta.ToolDescriptions.QueryTitle,
			QueryDescription:  runtimeMeta.ToolDescriptions.QueryDescription,
			MutateTitle:       runtimeMeta.ToolDescriptions.MutateTitle,
			MutateDescription: runtimeMeta.ToolDescriptions.MutateDescription,
			AuthErrorHint:     runtimeMeta.AuthErrorHint,
		},
		ContentMode: tools.ContentMode{
			MirrorStructuredContent: runtimeCfg.LegacyContentMirror,
		},
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

func NewHTTPClient(timeout time.Duration, proxyURL string) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	if strings.TrimSpace(proxyURL) != "" {
		parsedProxy, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("configure HTTP client proxy: invalid proxy URL %q: %w", proxyURL, err)
		}
		transport.Proxy = http.ProxyURL(parsedProxy)
	}
	transport.DialContext = (&net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
	}).DialContext
	transport.TLSHandshakeTimeout = timeout
	transport.ResponseHeaderTimeout = timeout
	transport.ExpectContinueTimeout = time.Second

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}, nil
}
