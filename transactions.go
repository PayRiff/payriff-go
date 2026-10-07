package payriff

import "context"

type TransactionsService struct {
	t *transport
}

func (s *TransactionsService) List(ctx context.Context, f *TransactionFilter) (*Page[Transaction], error) {
	if f == nil {
		f = &TransactionFilter{}
	}
	paging, err := pageQuery(f.Page, f.Size)
	if err != nil {
		return nil, err
	}
	query := append([][2]string{
		{"orderId", f.OrderID},
		{"rrn", f.RRN},
		{"status", string(f.Status)},
		{"amount", f.Amount},
		{"description", f.Description},
		{"name", f.Name},
		{"fullName", f.FullName},
		{"cardNumber", f.CardNumber},
		{"bookingId", f.BookingID},
		{"invoiceCode", f.InvoiceCode},
		{"from", filterDate(f.From)},
		{"to", filterDate(f.To)},
	}, paging...)
	var out Page[Transaction]
	if err := s.t.execute(ctx, apiRequest{method: "GET", path: "/api/v3/transactions", query: query}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
