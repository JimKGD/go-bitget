// Package futures provides a Go SDK for the Bitget futures trading API.
// It offers comprehensive functionality for futures trading operations including
// order management, account operations, market data retrieval, and real-time WebSocket connections.
//
// The package follows a service-oriented architecture with a fluent API pattern,
// allowing for intuitive method chaining when building requests.
//
// Example usage:
//
//	client := NewClient("api_key", "secret_key", "passphrase")
//	candles, err := client.NewCandlestickService().
//		Symbol("BTCUSDT").
//		ProductType(ProductTypeUSDTFutures).
//		Granularity("1m").
//		Limit("100").
//		Do(context.Background())
package futures

import (
	"errors"
	"fmt"
	"github.com/json-iterator/go"
	"github.com/khanbekov/go-bitget/common"
	"github.com/khanbekov/go-bitget/common/client"
	"github.com/khanbekov/go-bitget/common/types"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
	"net"
	"net/url"

	//jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog"
	"net/http"
	"os"
	"time"

	// NOTE: Subdirectory package imports removed to avoid import cycles
	// Factory methods will be implemented using interface{} returns
)

// BaseApiMainUrl is the main API endpoint for Bitget futures trading.
var (
	BaseApiMainUrl = "https://api.bitget.com"
)

// getApiEndpoint returns the base API endpoint URL.
// Currently returns the main production API URL.
func getApiEndpoint() string {
	return BaseApiMainUrl
}

// Client represents a Bitget futures API client.
// It contains all necessary credentials and configuration for interacting with the Bitget API.
type Client struct {
	// API credentials
	apiKey     string
	secretKey  string
	keyType    string
	passphrase string

	// HTTP client configuration
	BaseURL    string
	UserAgent  string
	fastClient *fasthttp.Client

	// Debugging and logging
	Debug  bool
	Logger zerolog.Logger

	// Request signing
	signer *common.Signer
}

// NewClient initializes a new Bitget futures API client with the provided credentials.
// This function must be called before using any SDK functionality.
//
// Parameters:
//   - apiKey: Your Bitget API key
//   - secretKey: Your Bitget secret key for request signing
//   - passphrase: Your API passphrase
//
// Returns a configured Client instance ready for use.
// Services can be created using the client.NewXXXService() pattern.
//
// Example:
//
//	client := NewClient("your_api_key", "your_secret_key", "your_passphrase")
//	candles, err := client.NewCandlestickService().Symbol("BTCUSDT").Do(ctx)
func NewClient(apiKey, secretKey, passphrase string) *Client {
	return &Client{
		apiKey:     apiKey,
		secretKey:  secretKey,
		passphrase: passphrase,
		signer:     common.NewSigner(secretKey),
		BaseURL:    getApiEndpoint(),
		UserAgent:  "Bitget/golang",
		fastClient: &fasthttp.Client{},
		Logger:     zerolog.New(os.Stderr).With().Timestamp().Logger(),
	}
}

// CallAPI sends an HTTP request to the specified Bitget API endpoint with
// automatic retry logic. It handles request signing, authentication headers,
// and exponential backoff for transient failures (network errors, HTTP 429/503/504).
//
// Returns the API response, response headers, and any error encountered.
func (c *Client) CallAPI(ctx context.Context, method string, endpoint string, queryParams url.Values, body []byte, sign bool) (*client.ApiResponse, *fasthttp.ResponseHeader, error) {
	const (
		maxRetries     = 3
		initialBackoff = 500 * time.Millisecond
		maxBackoff     = 10 * time.Second
		requestTimeout = 5 * time.Second
	)

	backoff := initialBackoff
	for attempt := 0; attempt < maxRetries; attempt++ {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()

		// Build URL
		requestURL := c.GetUrl(endpoint)
		if len(queryParams) > 0 {
			requestURL += "?" + queryParams.Encode()
		}
		req.SetRequestURI(requestURL)

		req.Header.SetMethod(method)
		if method == "POST" {
			req.SetBody(body)
			req.Header.Set("Content-Type", "application/json")
		}

		if sign {
			ts := common.TimestampMs()
			req.Header.Set("ACCESS-TIMESTAMP", ts)
			req.Header.Set("ACCESS-KEY", c.apiKey)
			req.Header.Set("ACCESS-PASSPHRASE", c.passphrase)
			req.Header.Set("locale", "en-US")

			var reqParamStr string
			if method == "GET" {
				if len(queryParams) > 0 {
					reqParamStr = "?" + queryParams.Encode()
				}
			} else {
				reqParamStr = string(body)
			}
			signature := c.signer.Sign(method, endpoint, reqParamStr, ts)
			req.Header.Set("ACCESS-SIGN", signature)
		}

		// Execute with context cancellation support.
		done := make(chan error, 1)
		go func() {
			done <- c.fastClient.DoTimeout(req, resp, requestTimeout)
		}()

		var err error
		select {
		case <-ctx.Done():
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			return nil, nil, ctx.Err()
		case err = <-done:
		}

		if err != nil {
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)

			if isRetryableError(err) && attempt < maxRetries-1 {
				if sleepErr := sleepWithContext(ctx, backoff); sleepErr != nil {
					return nil, nil, sleepErr
				}
				backoff = nextBackoff(backoff, maxBackoff)
				continue
			}
			return nil, nil, err
		}

		statusCode := resp.StatusCode()

		// Retryable HTTP statuses: 429 (rate-limited), 503, 504.
		if statusCode == http.StatusTooManyRequests ||
			statusCode == http.StatusServiceUnavailable ||
			statusCode == http.StatusGatewayTimeout {
			responseBody := string(resp.Body())
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)

			if attempt == maxRetries-1 {
				return nil, nil, fmt.Errorf("API request failed with status %d after %d retries: %s", statusCode, maxRetries, responseBody)
			}

			c.Logger.Warn().
				Int("status_code", statusCode).
				Int("attempt", attempt+1).
				Dur("backoff", backoff).
				Msg("Rate-limited or transient failure, retrying")

			if sleepErr := sleepWithContext(ctx, backoff); sleepErr != nil {
				return nil, nil, sleepErr
			}
			backoff = nextBackoff(backoff, maxBackoff)
			continue
		}

		// Any other client/server error is terminal.
		if statusCode >= http.StatusBadRequest {
			apiErr := &types.APIError{}
			if unmarshalErr := jsoniter.Unmarshal(resp.Body(), apiErr); unmarshalErr != nil {
				fasthttp.ReleaseRequest(req)
				fasthttp.ReleaseResponse(resp)
				return nil, nil, fmt.Errorf("error parsing API response (status %d): %w", statusCode, unmarshalErr)
			}
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			return nil, nil, apiErr
		}

		// Success path.
		var apiResp client.ApiResponse
		if err := jsoniter.Unmarshal(resp.Body(), &apiResp); err != nil {
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			return nil, nil, err
		}

		// Copy header so the caller keeps it after releasing.
		headerCopy := &fasthttp.ResponseHeader{}
		resp.Header.CopyTo(headerCopy)
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
		return &apiResp, headerCopy, nil
	}

	return nil, nil, fmt.Errorf("max retries exceeded")
}

// sleepWithContext sleeps for d, returning early if ctx is cancelled.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// nextBackoff doubles the current backoff, capped at maxBackoff.
func nextBackoff(current, maxBackoff time.Duration) time.Duration {
	next := current * 2
	if next > maxBackoff {
		return maxBackoff
	}
	return next
}

// isRetryableError determines if an error is transient and worth retrying.
// Returns true for network timeouts, connection errors, and other temporary failures.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// FastHTTP-specific errors
	if errors.Is(err, fasthttp.ErrTimeout) ||
		errors.Is(err, fasthttp.ErrNoFreeConns) ||
		errors.Is(err, fasthttp.ErrDialTimeout) {
		return true
	}

	// Network errors
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Temporary() {
		return true
	}

	return false
}

// SetApiEndpoint sets a custom API endpoint URL for the client.
// This can be used to switch between different environments or use a proxy.
func (c *Client) SetApiEndpoint(url string) *Client {
	c.BaseURL = url
	return c
}

// GetUrl constructs the full URL by combining the base URL with the given endpoint.
func (c *Client) GetUrl(endpoint string) string {
	return c.BaseURL + endpoint
}

// Factory methods are not provided in the main client to avoid import cycles.
// Use the package-specific constructors instead:
//
// Account Services Example:
//   import "github.com/khanbekov/go-bitget/futures/account"
//   client := futures.NewClient(apiKey, secretKey, passphrase)
//   accountInfo := account.NewAccountInfoService(client)
//   result, err := accountInfo.Symbol("BTCUSDT").ProductType("USDT-FUTURES").Do(ctx)
//
// Market Services Example:
//   import "github.com/khanbekov/go-bitget/futures/market"
//   candles := market.NewCandlestickService(client)
//   result, err := candles.Symbol("BTCUSDT").ProductType("USDT-FUTURES").Do(ctx)
//
// Trading Services Example:
//   import "github.com/khanbekov/go-bitget/futures/trading"
//   order := trading.NewCreateOrderService(client)
//   result, err := order.Symbol("BTCUSDT").Side("buy").Size("0.01").Do(ctx)
//
// Position Services Example:
//   import "github.com/khanbekov/go-bitget/futures/position"
//   positions := position.NewAllPositionsService(client)
//   result, err := positions.ProductType("USDT-FUTURES").Do(ctx)
//
// This approach provides:
// - Strong type safety 
// - No import cycles
// - Clear package organization
// - Better IDE support with auto-completion
