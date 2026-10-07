package payriff

import "context"

type InvoicesService struct {
	t *transport
}

func (s *InvoicesService) Create(ctx context.Context, p *InvoiceCreateParams) (*Invoice, error) {
	if p == nil {
		return nil, invalid("params are required")
	}
	if !p.AmountDynamic && p.Amount <= 0 {
		return nil, invalid("amount is required")
	}
	b := body{"currencyType": orDefault(p.Currency, CurrencyAZN)}.
		set("amount", p.Amount).
		set("amountDynamic", p.AmountDynamic).
		set("languageType", p.Language).
		set("fullName", p.FullName).
		set("email", p.Email).
		set("phoneNumber", p.PhoneNumber).
		set("description", p.Description).
		set("customMessage", p.CustomMessage).
		set("approveURL", p.ApproveURL).
		set("cancelURL", p.CancelURL).
		set("declineURL", p.DeclineURL).
		set("redirectURL", p.RedirectURL).
		set("installmentProductType", p.InstallmentProductType).
		set("installmentPeriod", p.InstallmentPeriod).
		set("directPay", p.DirectPay).
		set("sendSms", p.SendSMS).
		set("sendWhatsapp", p.SendWhatsapp).
		set("sendEmail", p.SendEmail).
		set("metadata", p.Metadata).
		set("externalTransactionId", p.ExternalTransactionID)
	if !p.ExpireDate.IsZero() {
		b["expireDate"] = p.ExpireDate.Format("2006-01-02T15:04:05")
	}
	var out Invoice
	if err := s.t.execute(ctx, apiRequest{method: "POST", path: "/api/v2/invoices", body: b, merchantEnvelope: true}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *InvoicesService) Get(ctx context.Context, invoiceUUID string) (*InvoiceDetails, error) {
	if _, err := segment(invoiceUUID, "invoiceUUID"); err != nil {
		return nil, err
	}
	var out InvoiceDetails
	err := s.t.execute(ctx, apiRequest{
		method:           "POST",
		path:             "/api/v2/get-invoice",
		body:             map[string]string{"uuid": invoiceUUID},
		merchantEnvelope: true,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
