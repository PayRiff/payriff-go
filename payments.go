package payriff

import (
	"context"
	"crypto/rsa"
)

type PaymentsService struct {
	t   *transport
	key *rsa.PublicKey
}

func (s *PaymentsService) DirectPay(ctx context.Context, p *DirectPayParams) (*DirectPayResponse, error) {
	if p == nil {
		return nil, invalid("params are required")
	}
	if err := positive(p.Amount, "amount"); err != nil {
		return nil, err
	}
	if err := required(p.Description, "description"); err != nil {
		return nil, err
	}
	card, err := normalizeCard(p.Card)
	if err != nil {
		return nil, err
	}
	encrypted, err := encryptCard(s.key, card)
	if err != nil {
		return nil, err
	}
	b := body{
		"amount":      p.Amount,
		"operation":   orDefault(p.Operation, OperationPurchase),
		"currency":    orDefault(p.Currency, CurrencyAZN),
		"description": p.Description,
		"paymentData": map[string]any{
			"paymentWay":       "DIRECT",
			"encryptedMessage": encrypted.message,
			"cardSave":         p.CardSave,
		},
	}.
		set("callbackUrl", p.CallbackURL).
		set("threeDS", p.ThreeDS).
		set("customFields", p.CustomFields)
	var out DirectPayResponse
	err = s.t.execute(ctx, apiRequest{
		method:  "POST",
		path:    "/api/v3/directPay",
		headers: map[string]string{"X-REQUEST-RRN": p.RequestRRN, "x-secret-key": encrypted.secretKey},
		body:    b,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
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
		"currency":    orDefault(p.Currency, CurrencyAZN),
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
