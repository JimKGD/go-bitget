package uta

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/khanbekov/go-bitget/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/valyala/fasthttp"
)

func TestGetInstrumentsService_FluentAPI(t *testing.T) {
	mockClient := &MockClient{}
	service := &GetInstrumentsService{c: mockClient}

	// Test fluent API chaining
	result := service.
		Category(CategoryUSDTFutures).
		Symbol("BTCUSDT")

	assert.Equal(t, service, result)
	assert.Equal(t, CategoryUSDTFutures, *service.category)
	assert.Equal(t, "BTCUSDT", *service.symbol)
}

func TestGetInstrumentsService_Do_Success_AllInstruments(t *testing.T) {
	// Setup mock data for multiple instruments
	mockInstruments := []Instrument{
		{
			Symbol:              "BTCUSDT",
			Category:            CategorySpot,
			BaseCoin:            "BTC",
			QuoteCoin:           "USDT",
			BuyLimitPriceRatio:  "0.02",
			SellLimitPriceRatio: "0.02",
			MinOrderQty:         "0.000001",
			MaxOrderQty:         "0",
			PricePrecision:      "2",
			QuantityPrecision:   "6",
			QuotePrecision:      "8",
			MinOrderAmount:      "1",
			MaxSymbolOrderNum:   "",
			MaxProductOrderNum:  "400",
			MaxPositionNum:      "200",
			Status:              "online",
			MaintainTime:        "",
			SymbolType:          "crypto",
		},
		{
			Symbol:              "BTCUSDT",
			Category:            CategoryUSDTFutures,
			BaseCoin:            "BTC",
			QuoteCoin:           "USDT",
			BuyLimitPriceRatio:  "0.05",
			SellLimitPriceRatio: "0.05",
			MinOrderQty:         "0.0001",
			MaxOrderQty:         "1200",
			PricePrecision:      "1",
			QuantityPrecision:   "4",
			MinOrderAmount:      "5",
			MaxProductOrderNum:  "400",
			MaxPositionNum:      "200",
			Status:              "online",
			SymbolType:          "crypto",
			IsRwa:               "NO",
			FeeRateUpRatio:      "0.005",
			MakerFeeRate:        "0.0002",
			TakerFeeRate:        "0.0006",
			OpenCostUpRatio:     "0.01",
			PriceMultiplier:     "0.1",
			QuantityMultiplier:  "0.0001",
			Type:                "perpetual",
			OffTime:             "-1",
			LimitOpenTime:       "-1",
			DeliveryTime:        "",
			DeliveryStartTime:   "",
			DeliveryPeriod:      "",
			FundInterval:        "8",
			MinLeverage:         "1",
			MaxLeverage:         "150",
			MaxMarketOrderQty:   "220",
		},
		{
			Symbol:                     "ADAUSDT",
			Category:                   CategoryMargin,
			BaseCoin:                   "ADA",
			QuoteCoin:                  "USDT",
			BuyLimitPriceRatio:         "0.02",
			SellLimitPriceRatio:        "0.02",
			MinOrderQty:                "0.001",
			MaxOrderQty:                "0",
			PricePrecision:             "4",
			QuantityPrecision:          "3",
			QuotePrecision:             "7",
			MinOrderAmount:             "1",
			MaxProductOrderNum:         "400",
			MaxPositionNum:             "200",
			Status:                     "online",
			SymbolType:                 "crypto",
			IsIsolatedBaseBorrowable:   "YES",
			IsIsolatedQuotedBorrowable: "YES",
			WarningRiskRatio:           "0.8",
			LiquidationRiskRatio:       "1",
			MaxCrossedLeverage:         "5",
			MaxIsolatedLeverage:        "10",
			UserMinBorrow:              "0.00000001",
			AreaSymbol:                 "no",
			MaxLeverage:                "10",
		},
	}

	mockDataBytes, _ := json.Marshal(mockInstruments)
	mockResponse := &ApiResponse{
		Code:        "00000",
		Msg:         "success",
		RequestTime: 1640995200000,
		Data:        mockDataBytes,
	}

	// Create mock client and service
	mockClient := &MockClient{}
	service := &GetInstrumentsService{c: mockClient}

	// Configure service parameters
	service.Category(CategorySpot)

	// Set up expected parameters
	expectedParams := url.Values{}
	expectedParams.Set("category", CategorySpot)

	// Set up expected API call
	mockClient.On("CallAPI",
		mock.Anything,
		"GET",
		EndpointMarketInstruments,
		expectedParams,
		[]byte(nil),
		false).Return(mockResponse, &fasthttp.ResponseHeader{}, nil)

	// Execute test
	ctx := context.Background()
	result, err := service.Do(ctx)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 3)

	// Check Spot instrument
	spotInstrument := result[0]
	assert.Equal(t, "BTCUSDT", spotInstrument.Symbol)
	assert.Equal(t, CategorySpot, spotInstrument.Category)
	assert.Equal(t, "BTC", spotInstrument.BaseCoin)
	assert.Equal(t, "USDT", spotInstrument.QuoteCoin)
	assert.Equal(t, "0.02", spotInstrument.BuyLimitPriceRatio)
	assert.Equal(t, "0.02", spotInstrument.SellLimitPriceRatio)
	assert.Equal(t, "0.000001", spotInstrument.MinOrderQty)
	assert.Equal(t, "0", spotInstrument.MaxOrderQty)
	assert.Equal(t, "2", spotInstrument.PricePrecision)
	assert.Equal(t, "6", spotInstrument.QuantityPrecision)
	assert.Equal(t, "8", spotInstrument.QuotePrecision)
	assert.Equal(t, "1", spotInstrument.MinOrderAmount)
	assert.Equal(t, "400", spotInstrument.MaxProductOrderNum)
	assert.Equal(t, "200", spotInstrument.MaxPositionNum)
	assert.Equal(t, "online", spotInstrument.Status)
	assert.Equal(t, "crypto", spotInstrument.SymbolType)

	// Check Futures instrument
	futuresInstrument := result[1]
	assert.Equal(t, "NO", futuresInstrument.IsRwa)
	assert.Equal(t, "0.005", futuresInstrument.FeeRateUpRatio)
	assert.Equal(t, "0.0002", futuresInstrument.MakerFeeRate)
	assert.Equal(t, "0.0006", futuresInstrument.TakerFeeRate)
	assert.Equal(t, "0.01", futuresInstrument.OpenCostUpRatio)
	assert.Equal(t, "0.1", futuresInstrument.PriceMultiplier)
	assert.Equal(t, "0.0001", futuresInstrument.QuantityMultiplier)
	assert.Equal(t, "perpetual", futuresInstrument.Type)
	assert.Equal(t, "-1", futuresInstrument.OffTime)
	assert.Equal(t, "-1", futuresInstrument.LimitOpenTime)
	assert.Equal(t, "", futuresInstrument.DeliveryTime)
	assert.Equal(t, "", futuresInstrument.DeliveryStartTime)
	assert.Equal(t, "", futuresInstrument.DeliveryPeriod)
	assert.Equal(t, "8", futuresInstrument.FundInterval)
	assert.Equal(t, "1", futuresInstrument.MinLeverage)
	assert.Equal(t, "150", futuresInstrument.MaxLeverage)
	assert.Equal(t, "220", futuresInstrument.MaxMarketOrderQty)

	// Check Margin instrument
	marginInstrument := result[2]
	assert.Equal(t, "YES", marginInstrument.IsIsolatedBaseBorrowable)
	assert.Equal(t, "YES", marginInstrument.IsIsolatedQuotedBorrowable)
	assert.Equal(t, "0.8", marginInstrument.WarningRiskRatio)
	assert.Equal(t, "1", marginInstrument.LiquidationRiskRatio)
	assert.Equal(t, "5", marginInstrument.MaxCrossedLeverage)
	assert.Equal(t, "10", marginInstrument.MaxIsolatedLeverage)
	assert.Equal(t, "0.00000001", marginInstrument.UserMinBorrow)
	assert.Equal(t, "no", marginInstrument.AreaSymbol)

	mockClient.AssertExpectations(t)
}

func TestGetInstrumentsService_Do_Success_SingleInstrument(t *testing.T) {
	// Setup mock data for single instrument
	mockInstruments := []Instrument{
		{
			Symbol:            "BTCUSDT",
			Category:          CategorySpot,
			BaseCoin:          "BTC",
			QuoteCoin:         "USDT",
			MinOrderQty:       "0.000001",
			PricePrecision:    "2",
			QuantityPrecision: "6",
			MinOrderAmount:    "1",
			Status:            "online",
			SymbolType:        "crypto",
		},
	}

	mockDataBytes, _ := json.Marshal(mockInstruments)
	mockResponse := &ApiResponse{
		Code:        "00000",
		Msg:         "success",
		RequestTime: 1640995200000,
		Data:        mockDataBytes,
	}

	mockClient := &MockClient{}
	service := &GetInstrumentsService{c: mockClient}

	// Configure service for single symbol
	service.Category(CategorySpot).Symbol("BTCUSDT")

	// Set up expected parameters
	expectedParams := url.Values{}
	expectedParams.Set("category", CategorySpot)
	expectedParams.Set("symbol", "BTCUSDT")

	mockClient.On("CallAPI",
		mock.Anything,
		"GET",
		EndpointMarketInstruments,
		expectedParams,
		[]byte(nil),
		false).Return(mockResponse, &fasthttp.ResponseHeader{}, nil)

	ctx := context.Background()
	result, err := service.Do(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, "BTCUSDT", result[0].Symbol)
	assert.Equal(t, CategorySpot, result[0].Category)

	mockClient.AssertExpectations(t)
}

func TestGetInstrumentsService_Do_MissingCategory(t *testing.T) {
	mockClient := &MockClient{}
	service := &GetInstrumentsService{c: mockClient}
	ctx := context.Background()

	// Test missing category
	_, err := service.Do(ctx)
	assert.Error(t, err)
	assert.IsType(t, &common.MissingParameterError{}, err)
}

func TestGetInstrumentsService_Do_DifferentCategories(t *testing.T) {
	categories := []string{CategorySpot, CategoryMargin, CategoryUSDTFutures, CategoryCoinFutures, CategoryUSDCFutures}

	for _, category := range categories {
		t.Run("Category_"+category, func(t *testing.T) {
			mockInstrument := []Instrument{
				{
					Symbol:    "BTCUSDT",
					Category:  category,
					BaseCoin:  "BTC",
					QuoteCoin: "USDT",
					Status:    "online",
				},
			}

			mockDataBytes, _ := json.Marshal(mockInstrument)
			mockResponse := &ApiResponse{
				Code:        "00000",
				Msg:         "success",
				RequestTime: 1640995200000,
				Data:        mockDataBytes,
			}

			mockClient := &MockClient{}
			service := &GetInstrumentsService{c: mockClient}
			service.Category(category)

			expectedParams := url.Values{}
			expectedParams.Set("category", category)

			mockClient.On("CallAPI",
				mock.Anything,
				"GET",
				EndpointMarketInstruments,
				expectedParams,
				[]byte(nil),
				false).Return(mockResponse, &fasthttp.ResponseHeader{}, nil)

			ctx := context.Background()
			result, err := service.Do(ctx)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Len(t, result, 1)
			assert.Equal(t, category, result[0].Category)

			mockClient.AssertExpectations(t)
		})
	}
}

func TestGetInstrumentsService_Do_APIError(t *testing.T) {
	mockClient := &MockClient{}
	service := &GetInstrumentsService{c: mockClient}
	service.Category(CategorySpot)

	expectedParams := url.Values{}
	expectedParams.Set("category", CategorySpot)

	// Set up expected API call to return error
	expectedError := assert.AnError
	mockClient.On("CallAPI",
		mock.Anything,
		"GET",
		EndpointMarketInstruments,
		expectedParams,
		[]byte(nil),
		false).Return(nil, &fasthttp.ResponseHeader{}, expectedError)

	ctx := context.Background()
	result, err := service.Do(ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)

	mockClient.AssertExpectations(t)
}

func TestGetInstrumentsService_Integration(t *testing.T) {
	// Integration-style test using the real client structure
	client := NewClient("test_api_key", "test_secret_key", "test_passphrase")
	service := client.NewGetInstrumentsService()

	assert.NotNil(t, service)
	assert.Equal(t, client, service.c)

	// Test fluent API works with real service
	service.Category(CategorySpot).Symbol("BTCUSDT")
	assert.Equal(t, CategorySpot, *service.category)
	assert.Equal(t, "BTCUSDT", *service.symbol)
}
