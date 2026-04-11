package uta

import (
	"context"
	"net/url"

	"github.com/khanbekov/go-bitget/common"
)

// GetOpenOrdersService retrieves unfilled (open) orders.
// If Symbol is not set, orders across all symbols in the category are returned.
type GetOpenOrdersService struct {
	c         ClientInterface
	category  *string
	symbol    *string
	orderID   *string
	clientOid *string
	orderType *string
	startTime *string
	endTime   *string
	limit     *string
	cursor    *string
}

// Category sets the product category (required): e.g. "USDT-FUTURES", "SPOT".
func (s *GetOpenOrdersService) Category(category string) *GetOpenOrdersService {
	s.category = &category
	return s
}

// Symbol filters by trading symbol (optional).
func (s *GetOpenOrdersService) Symbol(symbol string) *GetOpenOrdersService {
	s.symbol = &symbol
	return s
}

// OrderID filters by exchange order ID (optional).
func (s *GetOpenOrdersService) OrderID(orderID string) *GetOpenOrdersService {
	s.orderID = &orderID
	return s
}

// ClientOid filters by client order ID (optional).
func (s *GetOpenOrdersService) ClientOid(clientOid string) *GetOpenOrdersService {
	s.clientOid = &clientOid
	return s
}

// OrderType filters by order type (optional): "limit" or "market".
func (s *GetOpenOrdersService) OrderType(orderType string) *GetOpenOrdersService {
	s.orderType = &orderType
	return s
}

// StartTime sets the start of the query window in Unix milliseconds (optional).
func (s *GetOpenOrdersService) StartTime(startTime string) *GetOpenOrdersService {
	s.startTime = &startTime
	return s
}

// EndTime sets the end of the query window in Unix milliseconds (optional).
func (s *GetOpenOrdersService) EndTime(endTime string) *GetOpenOrdersService {
	s.endTime = &endTime
	return s
}

// Limit caps the number of records returned (optional, default/max defined by exchange).
func (s *GetOpenOrdersService) Limit(limit string) *GetOpenOrdersService {
	s.limit = &limit
	return s
}

// Cursor sets the pagination cursor from a previous response (optional).
func (s *GetOpenOrdersService) Cursor(cursor string) *GetOpenOrdersService {
	s.cursor = &cursor
	return s
}

// Do executes the get unfilled orders request.
// Returns the slice of open orders and a cursor for pagination (empty when no more pages).
func (s *GetOpenOrdersService) Do(ctx context.Context) ([]UnfilledOrder, string, error) {
	if s.category == nil {
		return nil, "", common.NewMissingParameterError("category")
	}

	params := url.Values{}
	params.Set("category", *s.category)

	if s.symbol != nil {
		params.Set("symbol", *s.symbol)
	}
	if s.orderID != nil {
		params.Set("orderId", *s.orderID)
	}
	if s.clientOid != nil {
		params.Set("clientOid", *s.clientOid)
	}
	if s.orderType != nil {
		params.Set("orderType", *s.orderType)
	}
	if s.startTime != nil {
		params.Set("startTime", *s.startTime)
	}
	if s.endTime != nil {
		params.Set("endTime", *s.endTime)
	}
	if s.limit != nil {
		params.Set("limit", *s.limit)
	}
	if s.cursor != nil {
		params.Set("cursor", *s.cursor)
	}

	res, _, err := s.c.CallAPI(ctx, "GET", EndpointTradeUnfilledOrders, params, nil, true)
	if err != nil {
		return nil, "", err
	}

	var page unfilledOrdersPage
	if err := common.UnmarshalJSON(res.Data, &page); err != nil {
		return nil, "", err
	}

	return page.List, page.Cursor, nil
}

// unfilledOrdersPage wraps the paginated API response payload.
type unfilledOrdersPage struct {
	List   []UnfilledOrder `json:"list"`
	Cursor string          `json:"cursor"`
}

// UnfilledOrder represents an open/unfilled order as returned by
// GET /api/v3/trade/unfilled-orders. JSON tags follow the UTA v3 field names,
// which differ from the minimal POST /place-order response captured by Order.
type UnfilledOrder struct {
	OrderID      string         `json:"orderId"`
	ClientOid    string         `json:"clientOid"`
	Symbol       string         `json:"symbol"`
	Category     string         `json:"category"`
	Side         string         `json:"side"`
	OrderType    string         `json:"orderType"`
	Price        string         `json:"price,omitempty"`
	Qty          string         `json:"qty"`
	Amount       string         `json:"amount,omitempty"`
	CumExecQty   string         `json:"cumExecQty"`
	CumExecValue string         `json:"cumExecValue"`
	AvgPrice     string         `json:"avgPrice"`
	TimeInForce  string         `json:"timeInForce,omitempty"`
	OrderStatus  string         `json:"orderStatus"`
	PosSide      string         `json:"posSide,omitempty"`
	HoldMode     string         `json:"holdMode,omitempty"`
	ReduceOnly   string         `json:"reduceOnly,omitempty"`
	STP          string         `json:"stp,omitempty"`
	FeeDetail    []OrderFeeItem `json:"feeDetail,omitempty"`
	CreatedTime  string         `json:"createdTime"`
	UpdatedTime  string         `json:"updatedTime"`
}

// OrderFeeItem represents a single fee entry from the feeDetail array.
type OrderFeeItem struct {
	FeeCoin string `json:"feeCoin"`
	Fee     string `json:"fee"`
}
