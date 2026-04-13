package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mimaurer/intersight-mcp/generated"
	internalpkg "github.com/mimaurer/intersight-mcp/internal"
	"github.com/mimaurer/intersight-mcp/internal/config"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"github.com/mimaurer/intersight-mcp/intersight"
	"github.com/mimaurer/intersight-mcp/sandbox"
	"github.com/mimaurer/intersight-mcp/server"
	"golang.org/x/sync/singleflight"
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
	artifacts, err := sandbox.LoadArtifactBundle(specBytes, sdkCatalogBytes, rulesBytes, searchCatalogBytes)
	if err != nil {
		return err
	}

	logger := internalpkg.NewLogger(stderr, cfg.LogLevel, cfg.LogFullCode)
	httpClient := newHTTPClient(cfg.PerCallTimeout)
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

	var apiCaller sandbox.APICaller = unavailableClient{err: contracts.AuthError{Message: "Intersight credentials are not configured; search is available, but query and mutate require INTERSIGHT_CLIENT_ID and INTERSIGHT_CLIENT_SECRET"}}
	if cfg.HasCredentials() {
		oauthCfg := intersight.OAuthConfig{
			TokenURL:     cfg.OAuthTokenURL,
			ValidateURL:  cfg.APIBaseURL + "/iam/UserPreferences",
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			HTTPClient:   httpClient,
		}
		oauthManager, authErr := bootstrapOAuthManager(ctx, ctx, cfg.PerCallTimeout, oauthCfg)
		if authErr != nil {
			apiCaller = &retryingBootstrapClient{
				ctx:        ctx,
				timeout:    cfg.PerCallTimeout,
				httpClient: httpClient,
				baseURL:    cfg.APIBaseURL,
				oauthCfg:   oauthCfg,
				lastErr:    authErr,
			}
		} else {
			apiCaller = sandboxClient{client: intersight.NewClient(httpClient, cfg.APIBaseURL, oauthManager)}
		}
	}

	queryExec, err := sandbox.NewQJSExecutorFromBundle(sandboxCfg, apiCaller, artifacts)
	if err != nil {
		_ = searchExec.Close()
		return err
	}
	mutateExec, err := sandbox.NewQJSExecutorFromBundle(sandboxCfg, apiCaller, artifacts)
	if err != nil {
		_ = searchExec.Close()
		_ = queryExec.Close()
		return err
	}

	runtime, err := server.NewRuntime(server.RuntimeConfig{
		ServerName:     "intersight-mcp",
		ServerVersion:  version,
		MaxConcurrent:  cfg.Execution.MaxConcurrent,
		MaxOutputBytes: cfg.Execution.MaxOutputBytes,
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

func newHTTPClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
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
	}
}

func bootstrapOAuthManager(managerCtx, bootstrapBaseCtx context.Context, timeout time.Duration, cfg intersight.OAuthConfig) (*intersight.Manager, error) {
	if managerCtx == nil {
		managerCtx = context.Background()
	}
	if bootstrapBaseCtx == nil {
		bootstrapBaseCtx = managerCtx
	}

	bootstrapCtx, cancel := context.WithTimeout(bootstrapBaseCtx, timeout)
	defer cancel()
	cfg.BootstrapContext = bootstrapCtx
	return intersight.NewOAuthManager(managerCtx, cfg)
}

type sandboxClient struct {
	client *intersight.Client
}

func (c sandboxClient) Do(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
	return c.client.Do(ctx, operation)
}

type unavailableClient struct {
	err error
}

func (c unavailableClient) Do(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
	return nil, c.err
}

type retryingBootstrapClient struct {
	ctx        context.Context
	timeout    time.Duration
	httpClient *http.Client
	baseURL    string
	oauthCfg   intersight.OAuthConfig

	mu             sync.RWMutex
	client         *intersight.Client
	lastErr        error
	bootstrapGroup singleflight.Group
}

func (c *retryingBootstrapClient) Do(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
	client, err := c.ensureClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.Do(ctx, operation)
}

func (c *retryingBootstrapClient) ensureClient(ctx context.Context) (*intersight.Client, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()
	if client != nil {
		return client, nil
	}

	resultCh := c.bootstrapGroup.DoChan("bootstrap-client", func() (any, error) {
		c.mu.RLock()
		client := c.client
		c.mu.RUnlock()
		if client != nil {
			return client, nil
		}

		manager, err := bootstrapOAuthManager(c.ctx, c.ctx, c.timeout, c.oauthCfg)
		if err != nil {
			c.mu.Lock()
			c.lastErr = err
			c.mu.Unlock()
			return nil, err
		}

		bootstrapped := intersight.NewClient(c.httpClient, c.baseURL, manager)

		c.mu.Lock()
		defer c.mu.Unlock()
		if c.client != nil {
			return c.client, nil
		}
		c.client = bootstrapped
		c.lastErr = nil
		return c.client, nil
	})

	select {
	case <-ctx.Done():
		return nil, bootstrapWaitError(ctx.Err())
	case result := <-resultCh:
		if result.Err != nil {
			return nil, result.Err
		}
		client, ok := result.Val.(*intersight.Client)
		if !ok || client == nil {
			return nil, contracts.InternalError{Message: "bootstrap client initialization returned an invalid result"}
		}
		return client, nil
	}
}

func bootstrapWaitError(err error) error {
	switch err {
	case nil:
		return nil
	case context.DeadlineExceeded:
		return contracts.TimeoutError{Message: "OAuth bootstrap timed out while waiting for credentials", Err: err}
	case context.Canceled:
		return contracts.NetworkError{Message: "OAuth bootstrap was canceled while waiting for credentials", Err: err}
	default:
		return contracts.NetworkError{Message: "OAuth bootstrap failed while waiting for credentials", Err: err}
	}
}
