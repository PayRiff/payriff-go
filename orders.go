package payriff

import "context"

type OrdersService struct {
	t        *transport
	currency Currency
}

func (s *OrdersService) Create(ctx context.Context, p *CreateOrderParams) (*CreateOrderResponse, error) {
	if p == nil {
		return nil, invalid("params are required")
	}
	if err := positive(p.Amount, "amount"); err != nil {
		return nil, err
	}
	b := body{"amount": p.Amount, "currency": orDefault(p.Currency, s.currency), "operation": orDefault(p.Operation, OperationPurchase)}.
		set("language", p.Language).
		set("description", p.Description).
		set("callbackUrl", p.CallbackURL).
		set("redirectUrl", p.RedirectURL).
		set("cardSave", p.CardSave).
		set("threeDS", p.ThreeDS).
		set("autoPaymentType", p.AutoPaymentType).
		set("installment", p.Installment).
		set("fullName", p.FullName).
		set("phoneNumber", p.PhoneNumber).
		set("metadata", p.Metadata).
		set("fields", p.Fields)
	var out CreateOrderResponse
	err := s.t.execute(ctx, apiRequest{
		method:  "POST",
		path:    "/api/v3/orders",
		headers: map[string]string{"X-REQUEST-RRN": p.RequestRRN},
		body:    b,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *OrdersService) Get(ctx context.Context, orderID string) (*OrderInfo, error) {
	return s.lookup(ctx, orderID, "orderID", "")
}

func (s *OrdersService) GetStatus(ctx context.Context, orderID string) (*OrderInfo, error) {
	return s.lookup(ctx, orderID, "orderID", "/status")
}

func (s *OrdersService) GetByRequestRRN(ctx context.Context, requestRRN string) (*OrderInfo, error) {
	return s.lookup(ctx, requestRRN, "requestRRN", "/rrn")
}

func (s *OrdersService) lookup(ctx context.Context, id, name, suffix string) (*OrderInfo, error) {
	seg, err := segment(id, name)
	if err != nil {
		return nil, err
	}
	var out OrderInfo
	if err := s.t.execute(ctx, apiRequest{method: "GET", path: "/api/v3/orders/" + seg + suffix}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *OrdersService) Expire(ctx context.Context, orderID string) error {
	if _, err := segment(orderID, "orderID"); err != nil {
		return err
	}
	return s.t.execute(ctx, apiRequest{method: "PATCH", path: "/api/v3/expire-status", query: [][2]string{{"orderId", orderID}}}, nil)
}

func (s *OrdersService) Refund(ctx context.Context, p *RefundParams) error {
	if p == nil {
		return invalid("params are required")
	}
	if err := required(p.OrderID, "orderID"); err != nil {
		return err
	}
	b := body{"orderId": p.OrderID}.
		set("amount", p.Amount).
		set("refundReason", p.RefundReason).
		set("callbackUrl", p.CallbackURL)
	return s.t.execute(ctx, apiRequest{method: "POST", path: "/api/v3/refund", body: b}, nil)
}

func (s *OrdersService) Complete(ctx context.Context, p *CompleteParams) error {
	if p == nil {
		return invalid("params are required")
	}
	if err := required(p.OrderID, "orderID"); err != nil {
		return err
	}
	b := body{"orderId": p.OrderID}.
		set("amount", p.Amount).
		set("callbackUrl", p.CallbackURL)
	return s.t.execute(ctx, apiRequest{method: "POST", path: "/api/v3/complete", body: b}, nil)
}

func (s *OrdersService) DownloadReceipt(ctx context.Context, orderIDOrRRN string) ([]byte, error) {
	seg, err := segment(orderIDOrRRN, "orderIDOrRRN")
	if err != nil {
		return nil, err
	}
	return s.t.download(ctx, apiRequest{method: "GET", path: "/api/v3/acquiring/receipt/" + seg})
}
