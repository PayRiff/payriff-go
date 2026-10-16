package payriff

import "context"

type PaymentsService struct {
	t        *transport
	currency Currency
}

func (s *PaymentsService) AutoPay(ctx context.Context, p *AutoPayParams) (*AutoPayResponse, error) {
	if p == nil {
		return nil, invalid("params are required")
	}
	if err := required(p.CardUUID, "cardUUID"); err != nil {
		return nil, err
	}
	if err := positive(p.Amount, "amount"); err != nil {
		return nil, err
	}
	if err := required(p.Description, "description"); err != nil {
		return nil, err
	}
	b := body{
		"cardUuid":    p.CardUUID,
		"amount":      p.Amount,
		"operation":   orDefault(p.Operation, OperationPurchase),
		"currency":    orDefault(p.Currency, s.currency),
		"description": p.Description,
	}.
		set("callbackUrl", p.CallbackURL).
		set("threeDS", p.ThreeDS).
		set("isOneCLickPayment", p.OneClickPayment)
	var out AutoPayResponse
	err := s.t.execute(ctx, apiRequest{
		method:  "POST",
		path:    "/api/v3/autoPay",
		headers: map[string]string{"X-REQUEST-RRN": p.RequestRRN},
		body:    b,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
