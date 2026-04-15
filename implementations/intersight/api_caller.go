package intersight

import (
	"context"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/mimaurer/intersight-mcp/implementations"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
	ciscointersight "github.com/mimaurer/intersight-mcp/intersight"
	"golang.org/x/sync/singleflight"
)

const (
	initialBootstrapBackoff = 1 * time.Second
	maxBootstrapBackoff     = 1 * time.Minute
)

type unavailableClient struct {
	err error
}

func (c unavailableClient) Do(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
	return nil, c.err
}

type RetryingBootstrapClient struct {
	ctx        context.Context
	timeout    time.Duration
	httpClient *http.Client
	baseURL    string
	oauthCfg   ciscointersight.OAuthConfig

	initialBackoff time.Duration
	maxBackoff     time.Duration
	now            func() time.Time

	mu                   sync.RWMutex
	client               *ciscointersight.Client
	lastErr              error
	bootstrapBackoff     time.Duration
	nextBootstrapAttempt time.Time
	bootstrapGroup       singleflight.Group
}

func NewRetryingBootstrapClient(ctx context.Context, timeout time.Duration, httpClient *http.Client, baseURL string, oauthCfg ciscointersight.OAuthConfig) *RetryingBootstrapClient {
	return &RetryingBootstrapClient{
		ctx:            ctx,
		timeout:        timeout,
		httpClient:     httpClient,
		baseURL:        baseURL,
		oauthCfg:       oauthCfg,
		initialBackoff: initialBootstrapBackoff,
		maxBackoff:     maxBootstrapBackoff,
	}
}

func (c *RetryingBootstrapClient) Do(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
	client, err := c.ensureClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.Do(ctx, operation)
}

func (c *RetryingBootstrapClient) EnsureClient(ctx context.Context) (*ciscointersight.Client, error) {
	return c.ensureClient(ctx)
}

func (c *RetryingBootstrapClient) ensureClient(ctx context.Context) (*ciscointersight.Client, error) {
	now := c.timeNow()
	if client, err := c.cachedClient(now); client != nil || err != nil {
		return client, err
	}

	resultCh := c.bootstrapGroup.DoChan("bootstrap-client", func() (any, error) {
		now := c.timeNow()
		if client, err := c.cachedClient(now); client != nil || err != nil {
			return client, err
		}

		manager, err := BootstrapOAuthManager(c.ctx, ctx, c.timeout, c.oauthCfg)
		if err != nil {
			c.recordBootstrapFailure(err, now)
			return nil, err
		}

		bootstrapped := ciscointersight.NewClient(c.httpClient, c.baseURL, manager)

		c.mu.Lock()
		defer c.mu.Unlock()
		if c.client != nil {
			return c.client, nil
		}
		c.client = bootstrapped
		c.lastErr = nil
		c.bootstrapBackoff = 0
		c.nextBootstrapAttempt = time.Time{}
		return c.client, nil
	})

	select {
	case <-ctx.Done():
		return nil, bootstrapWaitError(ctx.Err())
	case result := <-resultCh:
		if result.Err != nil {
			return nil, result.Err
		}
		client, ok := result.Val.(*ciscointersight.Client)
		if !ok || client == nil {
			return nil, contracts.InternalError{Message: "bootstrap client initialization returned an invalid result"}
		}
		return client, nil
	}
}

func (c *RetryingBootstrapClient) cachedClient(now time.Time) (*ciscointersight.Client, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.client != nil {
		return c.client, nil
	}
	if c.lastErr != nil && now.Before(c.nextBootstrapAttempt) {
		return nil, c.lastErr
	}
	return nil, nil
}

func (c *RetryingBootstrapClient) recordBootstrapFailure(err error, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastErr = err
	backoff := c.bootstrapBackoff
	if backoff <= 0 {
		backoff = c.initialBackoff
		if backoff <= 0 {
			backoff = initialBootstrapBackoff
		}
	} else {
		backoff = minDuration(backoff*2, c.maximumBackoff())
	}
	c.bootstrapBackoff = backoff
	c.nextBootstrapAttempt = now.Add(backoff)
}

func (c *RetryingBootstrapClient) maximumBackoff() time.Duration {
	if c.maxBackoff > 0 {
		return c.maxBackoff
	}
	return maxBootstrapBackoff
}

func (c *RetryingBootstrapClient) timeNow() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

func BootstrapOAuthManager(managerCtx, bootstrapBaseCtx context.Context, timeout time.Duration, cfg ciscointersight.OAuthConfig) (*ciscointersight.Manager, error) {
	if managerCtx == nil {
		managerCtx = context.Background()
	}
	if bootstrapBaseCtx == nil {
		bootstrapBaseCtx = managerCtx
	}

	bootstrapCtx, cancel := context.WithTimeout(bootstrapBaseCtx, timeout)
	defer cancel()
	cfg.BootstrapContext = bootstrapCtx
	return ciscointersight.NewOAuthManager(managerCtx, cfg)
}

func minDuration(left, right time.Duration) time.Duration {
	if left <= 0 {
		return right
	}
	if right <= 0 {
		return left
	}
	return time.Duration(math.Min(float64(left), float64(right)))
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

func newAPICaller(ctx context.Context, cfg ConnectionConfig, timeout time.Duration, httpClient *http.Client) implementations.APICaller {
	if !cfg.HasCredentials() {
		return unavailableClient{err: contracts.AuthError{Message: "Intersight credentials are not configured; search is available, but query and mutate require INTERSIGHT_CLIENT_ID and INTERSIGHT_CLIENT_SECRET"}}
	}

	oauthCfg := ciscointersight.OAuthConfig{
		TokenURL:     cfg.OAuthTokenURL,
		ValidateURL:  cfg.APIBaseURL + "/iam/UserPreferences",
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		HTTPClient:   httpClient,
	}
	return NewRetryingBootstrapClient(ctx, timeout, httpClient, cfg.APIBaseURL, oauthCfg)
}
