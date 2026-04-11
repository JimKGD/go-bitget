package uta

import (
	"context"
	"net/url"

	"github.com/khanbekov/go-bitget/common"
)

// GetCurrentPositionsService retrieves real-time open positions.
// If Symbol is not set, positions across all symbols in the category are returned.
type GetCurrentPositionsService struct {
	c        ClientInterface
	category *string
	symbol   *string
	side     *string
}

// Category sets the product category (required): e.g. "USDT-FUTURES", "COIN-FUTURES".
func (s *GetCurrentPositionsService) Category(category string) *GetCurrentPositionsService {
	s.category = &category
	return s
}

// Symbol filters by trading symbol (optional).
func (s *GetCurrentPositionsService) Symbol(symbol string) *GetCurrentPositionsService {
	s.symbol = &symbol
	return s
}

// Side filters by position side (optional): "long" or "short".
func (s *GetCurrentPositionsService) Side(side string) *GetCurrentPositionsService {
	s.side = &side
	return s
}

// Do executes the get current positions request.
func (s *GetCurrentPositionsService) Do(ctx context.Context) ([]CurrentPosition, error) {
	if s.category == nil {
		return nil, common.NewMissingParameterError("category")
	}

	params := url.Values{}
	params.Set("category", *s.category)

	if s.symbol != nil {
		params.Set("symbol", *s.symbol)
	}
	if s.side != nil {
		params.Set("posSide", *s.side)
	}

	res, _, err := s.c.CallAPI(ctx, "GET", EndpointPositionCurrentPosition, params, nil, true)
	if err != nil {
		return nil, err
	}

	var positions []CurrentPosition
	if err := common.UnmarshalJSON(res.Data, &positions); err != nil {
		return nil, err
	}

	return positions, nil
}

// CurrentPosition represents an open position as returned by
// GET /api/v3/position/current-position. JSON tags follow the UTA v3
// field names.
type CurrentPosition struct {
	Symbol           string `json:"symbol"`
	Category         string `json:"category"`
	MarginCoin       string `json:"marginCoin,omitempty"`
	PosSide          string `json:"posSide"`
	HoldMode         string `json:"holdMode,omitempty"`
	MarginMode       string `json:"marginMode"`
	Leverage         string `json:"leverage"`
	Qty              string `json:"qty"`
	Available        string `json:"available,omitempty"`
	Frozen           string `json:"frozen,omitempty"`
	AvgPrice         string `json:"avgPrice"`
	MarkPrice        string `json:"markPrice"`
	LiquidationPrice string `json:"liquidationPrice,omitempty"`
	UnrealizedPNL    string `json:"unrealizedPNL"`
	AchievedPNL      string `json:"achievedPNL,omitempty"`
	MarginSize       string `json:"marginSize,omitempty"`
	MarginRatio      string `json:"marginRatio,omitempty"`
	CreatedTime      string `json:"createdTime"`
	UpdatedTime      string `json:"updatedTime"`
}
