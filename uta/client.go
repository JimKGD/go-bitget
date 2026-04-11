package uta

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/khanbekov/go-bitget/common"
	"github.com/rs/zerolog"
	"github.com/valyala/fasthttp"
)

// Client represents the UTA API client
type Client struct {
	APIKey      string
	SecretKey   string
	Passphrase  string
	BaseURL     string
	HTTPClient  *fasthttp.Client
	Logger      zerolog.Logger
	json        jsoniter.API
	DemoTrading bool // Enable demo trading mode
}

// NewClient creates a new UTA API client
func NewClient(apiKey, secretKey, passphrase string) *Client {
	return &Client{
		APIKey:     apiKey,
		SecretKey:  secretKey,
		Passphrase: passphrase,
		BaseURL:    BaseURL,
		HTTPClient: &fasthttp.Client{},
		Logger:     zerolog.Nop(),
		json:       jsoniter.ConfigCompatibleWithStandardLibrary,
	}
}

// NewClientWithLogger creates a new UTA API client with custom logger
func NewClientWithLogger(apiKey, secretKey, passphrase string, logger zerolog.Logger) *Client {
	return &Client{
		APIKey:     apiKey,
		SecretKey:  secretKey,
		Passphrase: passphrase,
		BaseURL:    BaseURL,
		HTTPClient: &fasthttp.Client{},
		Logger:     logger,
		json:       jsoniter.ConfigCompatibleWithStandardLibrary,
	}
}

// SetBaseURL sets a custom base URL for the client
func (c *Client) SetBaseURL(baseURL string) *Client {
	c.BaseURL = baseURL
	return c
}

// SetHTTPClient sets a custom HTTP client
func (c *Client) SetHTTPClient(client *fasthttp.Client) *Client {
	c.HTTPClient = client
	return c
}

// SetDemoTrading enables or disables demo trading mode
func (c *Client) SetDemoTrading(demoTrading bool) *Client {
	c.DemoTrading = demoTrading
	return c
}

// Retry configuration for rate-limit / transient failures.
const (
	utaMaxRetries       = 3
	utaInitialBackoff   = 500 * time.Millisecond
	utaMaxBackoff       = 10 * time.Second
	utaRequestTimeout   = 30 * time.Second
)

// CallAPI makes an API call to the UTA API.
// It automatically retries on HTTP 429 (rate limited) with exponential backoff
// and respects context cancellation.
func (c *Client) CallAPI(ctx context.Context, method string, endpoint string, queryParams url.Values, body []byte, sign bool) (*ApiResponse, *fasthttp.ResponseHeader, error) {
	// Build URL once — does not change between retries.
	fullURL := c.BaseURL + endpoint
	if len(queryParams) > 0 {
		fullURL += "?" + queryParams.Encode()
	}

	c.Logger.Debug().
		Str("method", method).
		Str("url", fullURL).
		Int("body_bytes", len(body)).
		Bool("signed", sign).
		Msg("Making UTA API request")

	backoff := utaInitialBackoff
	for attempt := 0; attempt < utaMaxRetries; attempt++ {
		resp, err := c.doRequest(ctx, method, fullURL, endpoint, queryParams, body, sign)
		if err != nil {
			return nil, nil, err
		}

		statusCode := resp.StatusCode()
		// Rate-limit / server-side retryable failures → backoff and retry.
		if statusCode == fasthttp.StatusTooManyRequests ||
			statusCode == fasthttp.StatusServiceUnavailable ||
			statusCode == fasthttp.StatusGatewayTimeout {
			responseBody := string(resp.Body())
			fasthttp.ReleaseResponse(resp)

			if attempt == utaMaxRetries-1 {
				return nil, nil, fmt.Errorf("API request failed with status %d after %d retries: %s", statusCode, utaMaxRetries, responseBody)
			}

			c.Logger.Warn().
				Int("status_code", statusCode).
				Int("attempt", attempt+1).
				Dur("backoff", backoff).
				Msg("Rate-limited or transient failure, retrying")

			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > utaMaxBackoff {
				backoff = utaMaxBackoff
			}
			continue
		}

		// Any other non-200 → terminal error.
		if statusCode != fasthttp.StatusOK {
			responseBody := string(resp.Body())
			fasthttp.ReleaseResponse(resp)
			c.Logger.Error().
				Int("status_code", statusCode).
				Msg("API request failed with non-200 status")
			return nil, nil, fmt.Errorf("API request failed with status %d: %s", statusCode, responseBody)
		}

		// Parse response body.
		var apiResp ApiResponse
		if err := c.json.Unmarshal(resp.Body(), &apiResp); err != nil {
			fasthttp.ReleaseResponse(resp)
			c.Logger.Error().Err(err).Msg("Failed to unmarshal API response")
			return nil, nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		// Copy header so the caller keeps access after we release the response.
		headerCopy := &fasthttp.ResponseHeader{}
		resp.Header.CopyTo(headerCopy)
		fasthttp.ReleaseResponse(resp)

		c.Logger.Debug().
			Str("code", apiResp.Code).
			Int64("request_time", apiResp.RequestTime).
			Msg("Received UTA API response")

		if apiResp.Code != "00000" {
			apiError := &common.APIError{
				Code:    apiResp.Code,
				Message: apiResp.Msg,
			}
			c.Logger.Error().
				Str("error_code", apiError.Code).
				Msg("API returned error")
			return &apiResp, headerCopy, apiError
		}

		return &apiResp, headerCopy, nil
	}

	return nil, nil, fmt.Errorf("max retries exceeded")
}

// doRequest performs a single HTTP request attempt.
// Returns a *fasthttp.Response that the caller must ReleaseResponse after use.
func (c *Client) doRequest(ctx context.Context, method, fullURL, endpoint string, queryParams url.Values, body []byte, sign bool) (*fasthttp.Response, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()

	req.SetRequestURI(fullURL)
	req.Header.SetMethod(method)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "go-bitget-uta/1.0")

	if body != nil {
		req.SetBody(body)
	}

	if sign {
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

		var signString strings.Builder
		signString.WriteString(timestamp)
		signString.WriteString(method)
		signString.WriteString(endpoint)
		if len(queryParams) > 0 {
			signString.WriteString("?")
			signString.WriteString(queryParams.Encode())
		}
		if body != nil {
			signString.Write(body)
		}

		signature := c.createSignature(signString.String())

		req.Header.Set("ACCESS-KEY", c.APIKey)
		req.Header.Set("ACCESS-SIGN", signature)
		req.Header.Set("ACCESS-TIMESTAMP", timestamp)
		req.Header.Set("ACCESS-PASSPHRASE", c.Passphrase)
	}

	if c.DemoTrading {
		req.Header.Set("paptrading", "1")
	}

	// Respect context deadline where possible.
	timeout := utaRequestTimeout
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining < timeout && remaining > 0 {
			timeout = remaining
		}
	}

	if err := c.HTTPClient.DoTimeout(req, resp, timeout); err != nil {
		fasthttp.ReleaseResponse(resp)
		c.Logger.Error().Err(err).Msg("HTTP request failed")
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	return resp, nil
}

// createSignature creates HMAC SHA256 signature for request authentication
func (c *Client) createSignature(message string) string {
	mac := hmac.New(sha256.New, []byte(c.SecretKey))
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// Service factory methods

// Account management services
func (c *Client) NewAccountInfoService() *AccountInfoService {
	return &AccountInfoService{c: c}
}

func (c *Client) NewAccountAssetsService() *AccountAssetsService {
	return &AccountAssetsService{c: c}
}

func (c *Client) NewAccountFundingAssetsService() *AccountFundingAssetsService {
	return &AccountFundingAssetsService{c: c}
}

func (c *Client) NewAccountFeeRateService() *AccountFeeRateService {
	return &AccountFeeRateService{c: c}
}

func (c *Client) NewSetHoldingModeService() *SetHoldingModeService {
	return &SetHoldingModeService{c: c}
}

func (c *Client) NewSetLeverageService() *SetLeverageService {
	return &SetLeverageService{c: c}
}

func (c *Client) NewSwitchAccountService() *SwitchAccountService {
	return &SwitchAccountService{c: c}
}

func (c *Client) NewGetSwitchStatusService() *GetSwitchStatusService {
	return &GetSwitchStatusService{c: c}
}

// Transfer services
func (c *Client) NewTransferService() *TransferService {
	return &TransferService{c: c}
}

func (c *Client) NewSubTransferService() *SubTransferService {
	return &SubTransferService{c: c}
}

func (c *Client) NewGetTransferRecordsService() *GetTransferRecordsService {
	return &GetTransferRecordsService{c: c}
}

func (c *Client) NewGetTransferableCoinsService() *GetTransferableCoinsService {
	return &GetTransferableCoinsService{c: c}
}

// Deposit and withdrawal services
func (c *Client) NewGetDepositAddressService() *GetDepositAddressService {
	return &GetDepositAddressService{c: c}
}

func (c *Client) NewGetDepositRecordsService() *GetDepositRecordsService {
	return &GetDepositRecordsService{c: c}
}

func (c *Client) NewGetSubDepositAddressService() *GetSubDepositAddressService {
	return &GetSubDepositAddressService{c: c}
}

func (c *Client) NewGetSubDepositRecordsService() *GetSubDepositRecordsService {
	return &GetSubDepositRecordsService{c: c}
}

func (c *Client) NewWithdrawalService() *WithdrawalService {
	return &WithdrawalService{c: c}
}

func (c *Client) NewGetWithdrawalRecordsService() *GetWithdrawalRecordsService {
	return &GetWithdrawalRecordsService{c: c}
}

func (c *Client) NewSetDepositAccountService() *SetDepositAccountService {
	return &SetDepositAccountService{c: c}
}

// Financial records services
func (c *Client) NewGetFinancialRecordsService() *GetFinancialRecordsService {
	return &GetFinancialRecordsService{c: c}
}

func (c *Client) NewGetConvertRecordsService() *GetConvertRecordsService {
	return &GetConvertRecordsService{c: c}
}

func (c *Client) NewGetDeductInfoService() *GetDeductInfoService {
	return &GetDeductInfoService{c: c}
}

func (c *Client) NewSwitchDeductService() *SwitchDeductService {
	return &SwitchDeductService{c: c}
}

func (c *Client) NewGetPaymentCoinsService() *GetPaymentCoinsService {
	return &GetPaymentCoinsService{c: c}
}

func (c *Client) NewGetRepayableCoinsService() *GetRepayableCoinsService {
	return &GetRepayableCoinsService{c: c}
}

func (c *Client) NewRepayService() *RepayService {
	return &RepayService{c: c}
}

// Sub-account management services
func (c *Client) NewCreateSubAccountService() *CreateSubAccountService {
	return &CreateSubAccountService{c: c}
}

func (c *Client) NewGetSubAccountListService() *GetSubAccountListService {
	return &GetSubAccountListService{c: c}
}

func (c *Client) NewFreezeSubAccountService() *FreezeSubAccountService {
	return &FreezeSubAccountService{c: c}
}

func (c *Client) NewCreateSubAccountAPIKeyService() *CreateSubAccountAPIKeyService {
	return &CreateSubAccountAPIKeyService{c: c}
}

func (c *Client) NewGetSubAccountAPIKeysService() *GetSubAccountAPIKeysService {
	return &GetSubAccountAPIKeysService{c: c}
}

func (c *Client) NewModifySubAccountAPIKeyService() *ModifySubAccountAPIKeyService {
	return &ModifySubAccountAPIKeyService{c: c}
}

func (c *Client) NewDeleteSubAccountAPIKeyService() *DeleteSubAccountAPIKeyService {
	return &DeleteSubAccountAPIKeyService{c: c}
}

func (c *Client) NewGetSubAccountAssetsService() *GetSubAccountAssetsService {
	return &GetSubAccountAssetsService{c: c}
}

// Trading services
func (c *Client) NewPlaceOrderService() *PlaceOrderService {
	return &PlaceOrderService{c: c}
}

func (c *Client) NewCancelOrderService() *CancelOrderService {
	return &CancelOrderService{c: c}
}

func (c *Client) NewModifyOrderService() *ModifyOrderService {
	return &ModifyOrderService{c: c}
}

func (c *Client) NewBatchPlaceOrdersService() *BatchPlaceOrdersService {
	return &BatchPlaceOrdersService{c: c}
}

func (c *Client) NewBatchCancelOrdersService() *BatchCancelOrdersService {
	return &BatchCancelOrdersService{c: c}
}

func (c *Client) NewBatchModifyOrdersService() *BatchModifyOrdersService {
	return &BatchModifyOrdersService{c: c}
}

func (c *Client) NewCancelAllOrdersService() *CancelAllOrdersService {
	return &CancelAllOrdersService{c: c}
}

func (c *Client) NewCloseAllPositionsService() *CloseAllPositionsService {
	return &CloseAllPositionsService{c: c}
}

func (c *Client) NewCountdownCancelAllService() *CountdownCancelAllService {
	return &CountdownCancelAllService{c: c}
}

// Strategy order services
func (c *Client) NewPlaceStrategyOrderService() *PlaceStrategyOrderService {
	return &PlaceStrategyOrderService{c: c}
}

func (c *Client) NewCancelStrategyOrderService() *CancelStrategyOrderService {
	return &CancelStrategyOrderService{c: c}
}

func (c *Client) NewModifyStrategyOrderService() *ModifyStrategyOrderService {
	return &ModifyStrategyOrderService{c: c}
}

func (c *Client) NewGetUnfilledStrategyOrdersService() *GetUnfilledStrategyOrdersService {
	return &GetUnfilledStrategyOrdersService{c: c}
}

func (c *Client) NewGetStrategyOrderHistoryService() *GetStrategyOrderHistoryService {
	return &GetStrategyOrderHistoryService{c: c}
}

// Order and position query services
func (c *Client) NewGetOpenOrdersService() *GetOpenOrdersService {
	return &GetOpenOrdersService{c: c}
}

func (c *Client) NewGetOrderDetailsService() *GetOrderDetailsService {
	return &GetOrderDetailsService{c: c}
}

func (c *Client) NewGetOrderHistoryService() *GetOrderHistoryService {
	return &GetOrderHistoryService{c: c}
}

func (c *Client) NewGetFillHistoryService() *GetFillHistoryService {
	return &GetFillHistoryService{c: c}
}

func (c *Client) NewGetCurrentPositionsService() *GetCurrentPositionsService {
	return &GetCurrentPositionsService{c: c}
}

func (c *Client) NewGetPositionHistoryService() *GetPositionHistoryService {
	return &GetPositionHistoryService{c: c}
}

func (c *Client) NewGetMaxOpenAvailableService() *GetMaxOpenAvailableService {
	return &GetMaxOpenAvailableService{c: c}
}

func (c *Client) NewGetLoanOrdersService() *GetLoanOrdersService {
	return &GetLoanOrdersService{c: c}
}

// Market data services
func (c *Client) NewGetTickersService() *GetTickersService {
	return &GetTickersService{c: c}
}

func (c *Client) NewGetCandlesticksService() *GetCandlesticksService {
	return &GetCandlesticksService{c: c}
}

func (c *Client) NewGetHistoryCandlesticksService() *GetHistoryCandlesticksService {
	return &GetHistoryCandlesticksService{c: c}
}

func (c *Client) NewGetOrderBookService() *GetOrderBookService {
	return &GetOrderBookService{c: c}
}

// General market data services
func (c *Client) NewGetCurrentFundingRateService() *GetCurrentFundingRateService {
	return &GetCurrentFundingRateService{c: c}
}

func (c *Client) NewGetFundingRateHistoryService() *GetFundingRateHistoryService {
	return &GetFundingRateHistoryService{c: c}
}

func (c *Client) NewGetInstrumentsService() *GetInstrumentsService {
	return &GetInstrumentsService{c: c}
}

func (c *Client) NewGetDiscountRateService() *GetDiscountRateService {
	return &GetDiscountRateService{c: c}
}

func (c *Client) NewGetMarginLoansService() *GetMarginLoansService {
	return &GetMarginLoansService{c: c}
}

func (c *Client) NewGetOpenInterestService() *GetOpenInterestService {
	return &GetOpenInterestService{c: c}
}

func (c *Client) NewGetOILimitService() *GetOILimitService {
	return &GetOILimitService{c: c}
}

func (c *Client) NewGetProofOfReservesService() *GetProofOfReservesService {
	return &GetProofOfReservesService{c: c}
}

func (c *Client) NewGetRiskReserveService() *GetRiskReserveService {
	return &GetRiskReserveService{c: c}
}

func (c *Client) NewGetPositionTierService() *GetPositionTierService {
	return &GetPositionTierService{c: c}
}

func (c *Client) NewGetRecentPublicFillsService() *GetRecentPublicFillsService {
	return &GetRecentPublicFillsService{c: c}
}

// Ensure Client implements ClientInterface
var _ ClientInterface = (*Client)(nil)
