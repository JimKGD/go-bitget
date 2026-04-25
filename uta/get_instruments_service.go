package uta

import (
	"context"
	"net/url"

	"github.com/khanbekov/go-bitget/common"
)

// GetInstrumentsService retrieves specifications for trading pair instruments
type GetInstrumentsService struct {
	c        ClientInterface
	category *string
	symbol   *string
}

// Category sets the product category (required)
func (s *GetInstrumentsService) Category(category string) *GetInstrumentsService {
	s.category = &category
	return s
}

// Symbol sets the trading symbol (optional, if not set returns all symbols)
func (s *GetInstrumentsService) Symbol(symbol string) *GetInstrumentsService {
	s.symbol = &symbol
	return s
}

// Do executes the get tickers request
func (s *GetInstrumentsService) Do(ctx context.Context) ([]Instrument, error) {
	if s.category == nil {
		return nil, common.NewMissingParameterError("category")
	}

	params := url.Values{}
	params.Set("category", *s.category)

	if s.symbol != nil {
		params.Set("symbol", *s.symbol)
	}

	res, _, err := s.c.CallAPI(ctx, "GET", EndpointMarketInstruments, params, nil, false)
	if err != nil {
		return nil, err
	}

	var instruments []Instrument
	if err := common.UnmarshalJSON(res.Data, &instruments); err != nil {
		return nil, err
	}

	return instruments, nil
}
