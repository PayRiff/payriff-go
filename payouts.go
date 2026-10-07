package payriff

import (
	"context"
	"regexp"
)

type PayoutsService struct {
	t *transport
}

func (s *PayoutsService) Create(ctx context.Context, p *PayoutParams) (*PayoutResult, error) {
	if p == nil {
		return nil, invalid("params are required")
	}
	if p.TransferAmount < 1 {
		return nil, invalid("transferAmount must be at least 1")
	}
	for name, value := range map[string]string{"description": p.Description, "fullName": p.FullName, "finCode": p.FinCode} {
		if err := required(value, name); err != nil {
			return nil, err
		}
	}
	b := body{
		"transferAmount": p.TransferAmount,
		"description":    p.Description,
		"fullName":       p.FullName,
		"finCode":        p.FinCode,
	}.
		set("cardPan", p.CardPAN).
		set("bankName", p.BankName).
		set("cardType", p.CardType).
		set("requestRrn", p.RequestRRN).
		set("customerCode", p.CustomerCode).
		set("voen", p.VOEN).
		set("birthDate", p.BirthDate).
		set("callbackUrl", p.CallbackURL)
	var out PayoutResult
	err := s.t.execute(ctx, apiRequest{
		method:           "POST",
		path:             "/api/v3/payout",
		headers:          map[string]string{"X-IDEMPOTENCY-KEY": p.IdempotencyKey},
		body:             b,
		merchantEnvelope: true,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *PayoutsService) GetByRequestRRN(ctx context.Context, requestRRN string) (*PayoutStatus, error) {
	seg, err := segment(requestRRN, "requestRRN")
	if err != nil {
		return nil, err
	}
	var out PayoutStatus
	if err := s.t.execute(ctx, apiRequest{method: "GET", path: "/api/v3/payout/info/" + seg}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

var sixteenDigits = regexp.MustCompile(`^[0-9]{16}$`)

func (s *PayoutsService) CheckCardholder(ctx context.Context, cardPAN string) (string, error) {
	pan := separators.ReplaceAllString(cardPAN, "")
	if !sixteenDigits.MatchString(pan) {
		return "", invalid("cardPan must be a 16-digit number")
	}
	var out string
	err := s.t.execute(ctx, apiRequest{method: "POST", path: "/api/v3/payout/check-cardholder", body: map[string]string{"cardPan": pan}}, &out)
	return out, err
}

func (s *PayoutsService) List(ctx context.Context, f *PayoutFilter) (*Page[PayoutSummary], error) {
	if f == nil {
		f = &PayoutFilter{}
	}
	paging, err := pageQuery(f.Page, f.Size)
	if err != nil {
		return nil, err
	}
	query := append([][2]string{
		{"rrn", f.RRN},
		{"status", string(f.Status)},
		{"amount", f.Amount},
		{"description", f.Description},
		{"fullName", f.FullName},
		{"finCode", f.FinCode},
		{"bankSource", f.BankSource},
		{"from", filterDate(f.From)},
		{"to", filterDate(f.To)},
	}, paging...)
	var out Page[PayoutSummary]
	if err := s.t.execute(ctx, apiRequest{method: "GET", path: "/api/v3/payouts", query: query}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *PayoutsService) DownloadReceipt(ctx context.Context, requestRRN string) ([]byte, error) {
	seg, err := segment(requestRRN, "requestRRN")
	if err != nil {
		return nil, err
	}
	return s.t.download(ctx, apiRequest{method: "GET", path: "/api/v3/payout/receipt/" + seg})
}
