package uta

import (
	"context"
	"errors"
)

// This file contains stub implementations for services that are not yet wired to
// the Bitget v3 UTA API. Every Do() method returns ErrNotImplemented so callers
// receive a loud, explicit failure instead of a silent empty result.
//
// Each stub should be replaced with a real implementation as the service is
// needed. The ClientInterface factories are already registered in client.go.

// ErrNotImplemented is returned by every stub service. Callers can check
// errors.Is(err, uta.ErrNotImplemented) to distinguish stubbed endpoints.
var ErrNotImplemented = errors.New("uta: endpoint not implemented")

// Switch account service stubs
type SwitchAccountService struct{ c ClientInterface }

func (s *SwitchAccountService) Do(ctx context.Context) error { return ErrNotImplemented }

type GetSwitchStatusService struct{ c ClientInterface }

func (s *GetSwitchStatusService) Do(ctx context.Context) (*SwitchStatus, error) {
	return nil, ErrNotImplemented
}

// Transfer service stubs
type SubTransferService struct{ c ClientInterface }

func (s *SubTransferService) Do(ctx context.Context) (*TransferResult, error) {
	return nil, ErrNotImplemented
}

type GetTransferRecordsService struct{ c ClientInterface }

func (s *GetTransferRecordsService) Do(ctx context.Context) ([]TransferRecord, error) {
	return nil, ErrNotImplemented
}

type GetTransferableCoinsService struct{ c ClientInterface }

func (s *GetTransferableCoinsService) Do(ctx context.Context) ([]TransferableCoin, error) {
	return nil, ErrNotImplemented
}

// Deposit/withdrawal service stubs
type GetDepositAddressService struct{ c ClientInterface }

func (s *GetDepositAddressService) Do(ctx context.Context) (*DepositAddress, error) {
	return nil, ErrNotImplemented
}

type GetDepositRecordsService struct{ c ClientInterface }

func (s *GetDepositRecordsService) Do(ctx context.Context) ([]DepositRecord, error) {
	return nil, ErrNotImplemented
}

type GetSubDepositAddressService struct{ c ClientInterface }

func (s *GetSubDepositAddressService) Do(ctx context.Context) (*DepositAddress, error) {
	return nil, ErrNotImplemented
}

type GetSubDepositRecordsService struct{ c ClientInterface }

func (s *GetSubDepositRecordsService) Do(ctx context.Context) ([]DepositRecord, error) {
	return nil, ErrNotImplemented
}

type WithdrawalService struct{ c ClientInterface }

func (s *WithdrawalService) Do(ctx context.Context) (*WithdrawalResult, error) {
	return nil, ErrNotImplemented
}

type GetWithdrawalRecordsService struct{ c ClientInterface }

func (s *GetWithdrawalRecordsService) Do(ctx context.Context) ([]WithdrawalRecord, error) {
	return nil, ErrNotImplemented
}

type SetDepositAccountService struct{ c ClientInterface }

func (s *SetDepositAccountService) Do(ctx context.Context) error { return ErrNotImplemented }

// Financial records service stubs
type GetFinancialRecordsService struct{ c ClientInterface }

func (s *GetFinancialRecordsService) Do(ctx context.Context) ([]FinancialRecord, error) {
	return nil, ErrNotImplemented
}

type GetConvertRecordsService struct{ c ClientInterface }

func (s *GetConvertRecordsService) Do(ctx context.Context) ([]ConvertRecord, error) {
	return nil, ErrNotImplemented
}

type GetDeductInfoService struct{ c ClientInterface }

func (s *GetDeductInfoService) Do(ctx context.Context) (*DeductInfo, error) {
	return nil, ErrNotImplemented
}

type SwitchDeductService struct{ c ClientInterface }

func (s *SwitchDeductService) Do(ctx context.Context) error { return ErrNotImplemented }

type GetPaymentCoinsService struct{ c ClientInterface }

func (s *GetPaymentCoinsService) Do(ctx context.Context) ([]PaymentCoin, error) {
	return nil, ErrNotImplemented
}

type GetRepayableCoinsService struct{ c ClientInterface }

func (s *GetRepayableCoinsService) Do(ctx context.Context) ([]RepayableCoin, error) {
	return nil, ErrNotImplemented
}

type RepayService struct{ c ClientInterface }

func (s *RepayService) Do(ctx context.Context) (*RepayResult, error) {
	return nil, ErrNotImplemented
}

// Sub-account service stubs
type CreateSubAccountService struct{ c ClientInterface }

func (s *CreateSubAccountService) Do(ctx context.Context) (*SubAccount, error) {
	return nil, ErrNotImplemented
}

type GetSubAccountListService struct{ c ClientInterface }

func (s *GetSubAccountListService) Do(ctx context.Context) ([]SubAccount, error) {
	return nil, ErrNotImplemented
}

type FreezeSubAccountService struct{ c ClientInterface }

func (s *FreezeSubAccountService) Do(ctx context.Context) error { return ErrNotImplemented }

type CreateSubAccountAPIKeyService struct{ c ClientInterface }

func (s *CreateSubAccountAPIKeyService) Do(ctx context.Context) (*SubAccountAPIKey, error) {
	return nil, ErrNotImplemented
}

type GetSubAccountAPIKeysService struct{ c ClientInterface }

func (s *GetSubAccountAPIKeysService) Do(ctx context.Context) ([]SubAccountAPIKey, error) {
	return nil, ErrNotImplemented
}

type ModifySubAccountAPIKeyService struct{ c ClientInterface }

func (s *ModifySubAccountAPIKeyService) Do(ctx context.Context) error { return ErrNotImplemented }

type DeleteSubAccountAPIKeyService struct{ c ClientInterface }

func (s *DeleteSubAccountAPIKeyService) Do(ctx context.Context) error { return ErrNotImplemented }

type GetSubAccountAssetsService struct{ c ClientInterface }

func (s *GetSubAccountAssetsService) Do(ctx context.Context) ([]SubAccountAssets, error) {
	return nil, ErrNotImplemented
}

// Trading service stubs
// Note: ModifyOrderService is now implemented in modify_order_service.go

type BatchPlaceOrdersService struct{ c ClientInterface }

func (s *BatchPlaceOrdersService) Do(ctx context.Context) ([]BatchOrderResult, error) {
	return nil, ErrNotImplemented
}

type BatchCancelOrdersService struct{ c ClientInterface }

func (s *BatchCancelOrdersService) Do(ctx context.Context) ([]BatchOrderResult, error) {
	return nil, ErrNotImplemented
}

type BatchModifyOrdersService struct{ c ClientInterface }

func (s *BatchModifyOrdersService) Do(ctx context.Context) ([]BatchOrderResult, error) {
	return nil, ErrNotImplemented
}

type CancelAllOrdersService struct{ c ClientInterface }

func (s *CancelAllOrdersService) Do(ctx context.Context) error { return ErrNotImplemented }

type CloseAllPositionsService struct{ c ClientInterface }

func (s *CloseAllPositionsService) Do(ctx context.Context) error { return ErrNotImplemented }

type CountdownCancelAllService struct{ c ClientInterface }

func (s *CountdownCancelAllService) Do(ctx context.Context) error { return ErrNotImplemented }

// Strategy order service stubs
type PlaceStrategyOrderService struct{ c ClientInterface }

func (s *PlaceStrategyOrderService) Do(ctx context.Context) (*StrategyOrder, error) {
	return nil, ErrNotImplemented
}

type CancelStrategyOrderService struct{ c ClientInterface }

func (s *CancelStrategyOrderService) Do(ctx context.Context) (*StrategyOrder, error) {
	return nil, ErrNotImplemented
}

type ModifyStrategyOrderService struct{ c ClientInterface }

func (s *ModifyStrategyOrderService) Do(ctx context.Context) (*StrategyOrder, error) {
	return nil, ErrNotImplemented
}

type GetUnfilledStrategyOrdersService struct{ c ClientInterface }

func (s *GetUnfilledStrategyOrdersService) Do(ctx context.Context) ([]StrategyOrder, error) {
	return nil, ErrNotImplemented
}

type GetStrategyOrderHistoryService struct{ c ClientInterface }

func (s *GetStrategyOrderHistoryService) Do(ctx context.Context) ([]StrategyOrder, error) {
	return nil, ErrNotImplemented
}

// Order/position query service stubs
type GetOrderDetailsService struct{ c ClientInterface }

func (s *GetOrderDetailsService) Do(ctx context.Context) (*Order, error) {
	return nil, ErrNotImplemented
}

type GetOrderHistoryService struct{ c ClientInterface }

func (s *GetOrderHistoryService) Do(ctx context.Context) ([]Order, error) {
	return nil, ErrNotImplemented
}

type GetFillHistoryService struct{ c ClientInterface }

func (s *GetFillHistoryService) Do(ctx context.Context) ([]Fill, error) {
	return nil, ErrNotImplemented
}

type GetPositionHistoryService struct{ c ClientInterface }

func (s *GetPositionHistoryService) Do(ctx context.Context) ([]Position, error) {
	return nil, ErrNotImplemented
}

type GetMaxOpenAvailableService struct{ c ClientInterface }

func (s *GetMaxOpenAvailableService) Do(ctx context.Context) (*MaxOpenAvailable, error) {
	return nil, ErrNotImplemented
}

type GetLoanOrdersService struct{ c ClientInterface }

func (s *GetLoanOrdersService) Do(ctx context.Context) ([]LoanOrder, error) {
	return nil, ErrNotImplemented
}

// Market data service stubs
type GetHistoryCandlesticksService struct{ c ClientInterface }

func (s *GetHistoryCandlesticksService) Do(ctx context.Context) ([]Candlestick, error) {
	return nil, ErrNotImplemented
}

// GetOrderBookService implementation moved to get_orderbook_service.go

// General market data service stubs
type GetCurrentFundingRateService struct{ c ClientInterface }

func (s *GetCurrentFundingRateService) Do(ctx context.Context) (interface{}, error) {
	return nil, ErrNotImplemented
}

type GetFundingRateHistoryService struct{ c ClientInterface }

func (s *GetFundingRateHistoryService) Do(ctx context.Context) (interface{}, error) {
	return nil, ErrNotImplemented
}

type GetDiscountRateService struct{ c ClientInterface }

func (s *GetDiscountRateService) Do(ctx context.Context) (interface{}, error) {
	return nil, ErrNotImplemented
}

type GetMarginLoansService struct{ c ClientInterface }

func (s *GetMarginLoansService) Do(ctx context.Context) (interface{}, error) {
	return nil, ErrNotImplemented
}

type GetOpenInterestService struct{ c ClientInterface }

func (s *GetOpenInterestService) Do(ctx context.Context) (interface{}, error) {
	return nil, ErrNotImplemented
}

type GetOILimitService struct{ c ClientInterface }

func (s *GetOILimitService) Do(ctx context.Context) (interface{}, error) {
	return nil, ErrNotImplemented
}

type GetProofOfReservesService struct{ c ClientInterface }

func (s *GetProofOfReservesService) Do(ctx context.Context) (interface{}, error) {
	return nil, ErrNotImplemented
}

type GetRiskReserveService struct{ c ClientInterface }

func (s *GetRiskReserveService) Do(ctx context.Context) (interface{}, error) {
	return nil, ErrNotImplemented
}

type GetPositionTierService struct{ c ClientInterface }

func (s *GetPositionTierService) Do(ctx context.Context) (interface{}, error) {
	return nil, ErrNotImplemented
}

type GetRecentPublicFillsService struct{ c ClientInterface }

func (s *GetRecentPublicFillsService) Do(ctx context.Context) (interface{}, error) {
	return nil, ErrNotImplemented
}
