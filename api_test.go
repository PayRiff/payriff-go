package payriff

import (
	"errors"
	"testing"
	"time"
)

const orderInfoJSON = `{"orderId":"ORD-1","amount":10.50,"currencyType":"AZN","merchantName":"Shop","commission":0.20,
"commissionRate":2.00,"operationType":"PURCHASE","paymentStatus":"APPROVED","auto":false,
"createdDate":"2026-10-01T14:05:09.123456","description":"Order #1","metadata":"{\"k\":\"v\"}",
"transactions":[{"uuid":"6f1c2a4e-0b7d-4c3e-9a51-2d8e7f6b1c90","createdDate":"2026-10-01T14:05:10.000001",
"status":"APPROVED","channel":"KAPITAL_BANK","cardDetails":{"maskedPan":"416974******1979","brand":"VISA","bcryptedCardPan":"$2a$x"},
"installment":{"type":"BIRKART","period":"PERIOD_3"}}]}`

func okBody(payload string) string {
	return `{"code":"00000","message":"Operation performed successfully","payload":` + payload + `}`
}

func TestOrdersCreateSendsBodyAndRRNHeader(t *testing.T) {
	m := newMockServer(t)
	m.stub("POST", "/api/v3/orders", 200, nil, okBody(`{"orderId":"ORD-1","paymentUrl":"https://pay.payriff.com/ORD-1","transactionId":77,
		"comissionRate":2.5,"amount":10.00,"fee":0.25,"totalAmount":10.25}`))

	resp, err := m.client(t).Orders.Create(ctx, &CreateOrderParams{
		Amount:      10,
		Language:    LanguageAZ,
		Description: "Order #1",
		CallbackURL: "https://shop.az/cb",
		Installment: &Installment{Type: InstallmentBirKart, Period: InstallmentMonths(3)},
		Metadata:    map[string]string{"cartId": "c-9"},
		RequestRRN:  "rrn-1",
	})

	if err != nil {
		t.Fatal(err)
	}
	sent := m.last(t)
	if sent.Headers.Get("X-REQUEST-RRN") != "rrn-1" || sent.Headers.Get("Content-Type") != "application/json" {
		t.Fatalf("headers %v", sent.Headers)
	}
	mustJSONEqual(t, `{"amount":10,"currency":"AZN","language":"AZ","operation":"PURCHASE","description":"Order #1",
		"callbackUrl":"https://shop.az/cb","installment":{"type":"BIRKART","period":"PERIOD_3"},"metadata":{"cartId":"c-9"}}`, sent.Body)
	if resp.OrderID != "ORD-1" || resp.TransactionID != 77 || resp.CommissionRate != 2.5 || resp.TotalAmount != 10.25 {
		t.Fatalf("got %+v", resp)
	}
}

func TestOrdersCreateOmitsRRNHeaderAndEmptyMaps(t *testing.T) {
	m := newMockServer(t)
	m.ok("POST", "/api/v3/orders", map[string]any{"orderId": "ORD-1"})

	if _, err := m.client(t).Orders.Create(ctx, &CreateOrderParams{Amount: 1, Metadata: map[string]string{}}); err != nil {
		t.Fatal(err)
	}

	if _, ok := m.last(t).Headers["X-Request-Rrn"]; ok {
		t.Fatal("RRN header sent")
	}
	mustJSONEqual(t, `{"amount":1,"currency":"AZN","operation":"PURCHASE"}`, m.last(t).Body)
}

func TestOrdersCreateRejectsMissingAmount(t *testing.T) {
	m := newMockServer(t)

	_, err := m.client(t).Orders.Create(ctx, &CreateOrderParams{})

	mustInvalid(t, err, "amount is required")
	if _, err := m.client(t).Orders.Create(ctx, nil); !isInvalid(err) {
		t.Fatalf("nil params: %v", err)
	}
}

func TestOrdersLookupsParseOrderInfo(t *testing.T) {
	m := newMockServer(t)
	m.stub("GET", "/api/v3/orders/ORD-1", 200, nil, okBody(orderInfoJSON))
	m.stub("GET", "/api/v3/orders/ORD-1/status", 200, nil, okBody(orderInfoJSON))
	c := m.client(t)

	for _, call := range []func() (*OrderInfo, error){
		func() (*OrderInfo, error) { return c.Orders.Get(ctx, "ORD-1") },
		func() (*OrderInfo, error) { return c.Orders.GetStatus(ctx, "ORD-1") },
	} {
		o, err := call()
		if err != nil {
			t.Fatal(err)
		}
		if o.OrderID != "ORD-1" || o.Amount != 10.5 || o.Currency != CurrencyAZN || o.PaymentStatus != PaymentApproved ||
			o.OperationType != OperationPurchase || o.Metadata != `{"k":"v"}` {
			t.Fatalf("got %+v", o)
		}
		if !o.CreatedDate.Equal(time.Date(2026, 10, 1, 14, 5, 9, 123456000, time.UTC)) {
			t.Fatalf("createdDate %v", o.CreatedDate)
		}
		tx := o.Transactions[0]
		if tx.Channel != "KAPITAL_BANK" || tx.CardDetails.MaskedPAN != "416974******1979" || tx.CardDetails.Brand != "VISA" ||
			tx.Installment.Period != "PERIOD_3" {
			t.Fatalf("got %+v", tx)
		}
	}
}

func TestUnknownValuesDoNotBreakParsing(t *testing.T) {
	m := newMockServer(t)
	m.stub("GET", "/api/v3/orders/ORD-1", 200, nil, okBody(`{"orderId":"ORD-1","paymentStatus":"SOMETHING_NEW","currencyType":"GBP",
		"createdDate":"not a date","transactions":null,"brandNewField":{"x":1}}`))

	o, err := m.client(t).Orders.Get(ctx, "ORD-1")

	if err != nil || o.PaymentStatus != "SOMETHING_NEW" || o.Currency != "GBP" || !o.CreatedDate.IsZero() || o.Transactions != nil {
		t.Fatalf("got %+v, %v", o, err)
	}
}

func TestOrdersRefundCompleteExpire(t *testing.T) {
	m := newMockServer(t)
	m.ok("POST", "/api/v3/refund", nil)
	m.ok("POST", "/api/v3/complete", nil)
	c := m.client(t)

	if err := c.Orders.Refund(ctx, &RefundParams{OrderID: "ORD-1", Amount: 5, RefundReason: "damaged"}); err != nil {
		t.Fatal(err)
	}
	mustJSONEqual(t, `{"orderId":"ORD-1","amount":5,"refundReason":"damaged"}`, m.last(t).Body)

	if err := c.Orders.Complete(ctx, &CompleteParams{OrderID: "ORD-1", Amount: 7.5}); err != nil {
		t.Fatal(err)
	}
	mustJSONEqual(t, `{"orderId":"ORD-1","amount":7.5}`, m.last(t).Body)

	mustInvalid(t, c.Orders.Refund(ctx, &RefundParams{}), "orderID is required")
}

func TestAutoPaySendsExplicitCurrencyAndOneClickFlag(t *testing.T) {
	m := newMockServer(t)
	m.stub("POST", "/api/v3/autoPay", 200, nil, okBody(`{"orderId":"ORD-2","amount":3.00,"paymentStatus":"APPROVED","auto":true,
		"createdDate":"2026-10-01T10:00:00.000000","transactionResponseDto":{"threeDS":false}}`))

	resp, err := m.client(t).Payments.AutoPay(ctx, &AutoPayParams{
		CardUUID: "card-uuid-1", Amount: 3, Description: "Subscription", OneClickPayment: Bool(true), RequestRRN: "rrn-2",
	})

	if err != nil {
		t.Fatal(err)
	}
	mustJSONEqual(t, `{"cardUuid":"card-uuid-1","amount":3,"operation":"PURCHASE","currency":"AZN","description":"Subscription","isOneCLickPayment":true}`, m.last(t).Body)
	if m.last(t).Headers.Get("X-REQUEST-RRN") != "rrn-2" || resp.PaymentStatus != PaymentApproved || !resp.Auto || resp.TransactionResponse.ThreeDS {
		t.Fatalf("got %+v", resp)
	}
}

func TestCardsSaveGetListDelete(t *testing.T) {
	m := newMockServer(t)
	m.ok("POST", "/api/v3/cards/save", map[string]any{"cardSaveId": "cs-1", "status": "CREATED", "amount": 0.10})
	m.stub("GET", "/api/v3/cards/save/cs-1", 200, nil, okBody(`{"cardSaveId":"cs-1","status":"VERIFIED","cardUuid":"card-1","verifiedDate":"2026-10-01T10:01:30.5"}`))
	m.ok("GET", "/api/v3/cards/save?customerRef=cust%201", []map[string]any{{"cardUuid": "card-1"}, {"cardUuid": "card-2"}})
	m.ok("DELETE", "/api/v3/cards/card-1", true)
	c := m.client(t)

	saved, err := c.Cards.Save(ctx, &CardSaveParams{CustomerRef: "cust-1", CallbackURL: "https://shop.az/cards/cb", Language: LanguageAZ, IdempotencyKey: "idem-1"})
	if err != nil || saved.CardSaveID != "cs-1" || saved.Status != CardSaveCreated || saved.Amount != 0.1 {
		t.Fatalf("save %+v, %v", saved, err)
	}
	if m.last(t).Headers.Get("X-Idempotency-Key") != "idem-1" {
		t.Fatal("idempotency key missing")
	}
	mustJSONEqual(t, `{"customerRef":"cust-1","callbackUrl":"https://shop.az/cards/cb","language":"AZ"}`, m.last(t).Body)

	details, err := c.Cards.GetSave(ctx, "cs-1")
	if err != nil || details.Status != CardSaveVerified || details.CardUUID != "card-1" || details.VerifiedDate.Nanosecond() != 500000000 {
		t.Fatalf("getSave %+v, %v", details, err)
	}

	cards, err := c.Cards.List(ctx, "cust 1")
	if err != nil || len(cards) != 2 || cards[1].CardUUID != "card-2" {
		t.Fatalf("list %+v, %v", cards, err)
	}

	if err := c.Cards.Delete(ctx, "card-1"); err != nil || m.last(t).Method != "DELETE" {
		t.Fatalf("delete %v", err)
	}

	_, err = c.Cards.Save(ctx, &CardSaveParams{CustomerRef: "c"})
	mustInvalid(t, err, "callbackURL is required")
}

func TestCardsListReturnsEmptySliceForNullPayload(t *testing.T) {
	m := newMockServer(t)
	m.ok("GET", "/api/v3/cards/save", nil)

	cards, err := m.client(t).Cards.List(ctx, "cust-1")

	if err != nil || cards == nil || len(cards) != 0 {
		t.Fatalf("got %#v, %v", cards, err)
	}
}

func TestTransactionsListSendsFilterAndParsesPage(t *testing.T) {
	m := newMockServer(t)
	m.stub("GET", "/api/v3/transactions?status=APPROVED&from=01.09.2026&to=30.09.2026&page=1&offset=20", 200, nil, okBody(`{"content":[{
		"id":5,"orderId":"ORD-1","amount":10.00,"currencyType":"AZN","paymentStatus":"APPROVED",
		"card_brand":"VISA","payment_way":"DIRECT","extra_payment":0.50,"createdDate":"2026-09-15 10:00:00"}],
		"totalElements":41,"totalPages":3,"number":1,"size":20,"first":false,"last":false}`))

	page, err := m.client(t).Transactions.List(ctx, &TransactionFilter{
		Status: PaymentApproved,
		From:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		To:     time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
		Page:   1,
		Size:   20,
	})

	if err != nil {
		t.Fatal(err)
	}
	tx := page.Content[0]
	if page.TotalElements != 41 || page.Last || tx.CardBrand != "VISA" || tx.PaymentWay != "DIRECT" || tx.ExtraPayment != 0.5 || tx.CreatedDate != "2026-09-15 10:00:00" {
		t.Fatalf("got %+v", page)
	}
}

func TestTransactionsDefaultsAndPageSizeCap(t *testing.T) {
	m := newMockServer(t)
	m.ok("GET", "/api/v3/transactions", map[string]any{"content": []any{}})
	c := m.client(t)

	if _, err := c.Transactions.List(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if m.last(t).URL != "/api/v3/transactions?page=0&offset=10" {
		t.Fatalf("url %s", m.last(t).URL)
	}
	for _, size := range []int{21, -1} {
		_, err := c.Transactions.List(ctx, &TransactionFilter{Size: size})
		mustInvalid(t, err, "size must be 1-20")
	}
	_, err := c.Transactions.List(ctx, &TransactionFilter{Page: -1})
	mustInvalid(t, err, "page must be >= 0")
}

func TestPayoutsCreateWrapsBodyInMerchantEnvelope(t *testing.T) {
	m := newMockServer(t)
	m.stub("POST", "/api/v3/payout", 200, nil, okBody(`{"_final":"true","state":"SUCCESS","currentDepositBalance":975.00,"walletHistoryId":321,"bankName":"KAPITAL"}`))
	params := &PayoutParams{TransferAmount: 25, Description: "Refund to customer", FullName: "JOHN DOE", FinCode: "1AB2C3D",
		CardPAN: "4169741330151979", RequestRRN: "po-1", IdempotencyKey: "idem-po-1"}

	res, err := m.client(t).Payouts.Create(ctx, params)

	if err != nil {
		t.Fatal(err)
	}
	if m.last(t).Headers.Get("X-IDEMPOTENCY-KEY") != "idem-po-1" {
		t.Fatal("idempotency key missing")
	}
	mustJSONEqual(t, `{"merchant":"ES1000000","body":{"transferAmount":25,"description":"Refund to customer","fullName":"JOHN DOE",
		"finCode":"1AB2C3D","cardPan":"4169741330151979","requestRrn":"po-1"}}`, m.last(t).Body)
	if res.FinalState != "true" || res.State != "SUCCESS" || res.WalletHistoryID != 321 || res.CurrentDepositBalance != 975 {
		t.Fatalf("got %+v", res)
	}
}

func TestPayoutsErrorsAndValidation(t *testing.T) {
	m := newMockServer(t)
	m.stub("POST", "/api/v3/payout", 402, nil, map[string]any{"code": "01200", "message": "Insufficient wallet balance"})
	c := m.client(t)
	params := PayoutParams{TransferAmount: 25, Description: "d", FullName: "f", FinCode: "c"}

	_, err := c.Payouts.Create(ctx, &params)
	if !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("got %v", err)
	}

	params.TransferAmount = 0.99
	_, err = c.Payouts.Create(ctx, &params)
	mustInvalid(t, err, "transferAmount must be at least 1")

	_, err = c.Payouts.CheckCardholder(ctx, "4169")
	mustInvalid(t, err, "cardPan must be a 16-digit number")
}

func TestPayoutsLookupsAndReceipt(t *testing.T) {
	m := newMockServer(t)
	m.stub("GET", "/api/v3/payout/info/po-1", 200, nil, okBody(`{"state":"IN_PROGRESS","transferAmount":25.00,"createdDate":"2026-10-01T10:00:00.000+00:00","formattedDate":"01.10.2026 10:00:00"}`))
	m.ok("POST", "/api/v3/payout/check-cardholder", "J*** D**")
	m.stub("GET", "/api/v3/payouts?status=SUCCESS&page=0&offset=10", 200, nil, okBody(`{"content":[{"id":1,"requestRrn":"po-1","state":"SUCCESS","createdDate":"2026-10-01T10:00:00"}],"totalElements":1}`))
	m.stub("GET", "/api/v3/payout/receipt/po-1", 200, map[string]string{"Content-Type": "application/pdf"}, []byte("%PDF-"))
	c := m.client(t)

	status, err := c.Payouts.GetByRequestRRN(ctx, "po-1")
	if err != nil || status.State != TransferInProgress || status.FormattedDate != "01.10.2026 10:00:00" {
		t.Fatalf("status %+v, %v", status, err)
	}

	name, err := c.Payouts.CheckCardholder(ctx, "4169 7413 3015 1979")
	if err != nil || name != "J*** D**" {
		t.Fatalf("cardholder %q, %v", name, err)
	}
	mustJSONEqual(t, `{"cardPan":"4169741330151979"}`, m.last(t).Body)

	page, err := c.Payouts.List(ctx, &PayoutFilter{Status: TransferSuccess})
	if err != nil || page.Content[0].RequestRRN != "po-1" || page.Content[0].State != TransferSuccess {
		t.Fatalf("list %+v, %v", page, err)
	}

	pdf, err := c.Payouts.DownloadReceipt(ctx, "po-1")
	if err != nil || string(pdf) != "%PDF-" {
		t.Fatalf("receipt %q, %v", pdf, err)
	}
}

func TestInvoicesCreateAndGet(t *testing.T) {
	m := newMockServer(t)
	m.stub("POST", "/api/v2/invoices", 200, nil, okBody(`{"id":9,"invoiceUuid":"inv-uuid-1","invoiceStatus":"PENDING",
		"paymentUrl":"https://pay.payriff.com/i/inv?type=preview","amount":15.00,"currencyType":"AZN","languageType":"AZ","expireDate":"2026-10-08T23:59:00"}`))
	m.stub("POST", "/api/v2/get-invoice", 200, nil, okBody(`{"invoiceUuid":"inv-uuid-1","invoiceStatus":"COMPLETE","paymentDay":"2026-10-02","createdDate":"2026-10-01T09:00:00.123"}`))
	c := m.client(t)

	inv, err := c.Invoices.Create(ctx, &InvoiceCreateParams{
		Amount:      15,
		Language:    LanguageAZ,
		FullName:    "JOHN DOE",
		PhoneNumber: "+994501234567",
		Description: "Consultation",
		ExpireDate:  time.Date(2026, 10, 8, 23, 59, 0, 0, time.Local),
		ApproveURL:  "https://shop.az/ok",
		SendSMS:     Bool(false),
		Metadata:    map[string]string{"bookingRef": "B-77"},
	})
	if err != nil {
		t.Fatal(err)
	}
	mustJSONEqual(t, `{"merchant":"ES1000000","body":{"amount":15,"currencyType":"AZN","languageType":"AZ","fullName":"JOHN DOE",
		"phoneNumber":"+994501234567","description":"Consultation","expireDate":"2026-10-08T23:59:00","approveURL":"https://shop.az/ok",
		"sendSms":false,"metadata":{"bookingRef":"B-77"}}}`, m.last(t).Body)
	if inv.InvoiceUUID != "inv-uuid-1" || inv.Status != InvoicePending || inv.Currency != CurrencyAZN || inv.Language != LanguageAZ ||
		inv.ExpireDate.Hour() != 23 {
		t.Fatalf("got %+v", inv)
	}

	details, err := c.Invoices.Get(ctx, "inv-uuid-1")
	if err != nil || details.Status != InvoiceComplete || details.PaymentDay.Day() != 2 {
		t.Fatalf("got %+v, %v", details, err)
	}
	mustJSONEqual(t, `{"merchant":"ES1000000","body":{"uuid":"inv-uuid-1"}}`, m.last(t).Body)

	_, err = c.Invoices.Create(ctx, &InvoiceCreateParams{})
	mustInvalid(t, err, "amount is required")
	m.ok("POST", "/api/v2/invoices", map[string]any{"invoiceUuid": "inv-2"})
	if _, err := c.Invoices.Create(ctx, &InvoiceCreateParams{AmountDynamic: true}); err != nil {
		t.Fatal(err)
	}
	mustJSONEqual(t, `{"merchant":"ES1000000","body":{"amountDynamic":true,"currencyType":"AZN"}}`, m.last(t).Body)
}

func TestParseOrderCallback(t *testing.T) {
	body := `{"payload":{"orderId":"ORD-1","invoiceUuid":"inv-1","amount":10.5,"currencyType":"AZN","paymentStatus":"APPROVED",
		"operationType":"PURCHASE","auto":false,"createdDate":"2026-10-01T14:05:09.123","customFields":{"x":"y"},
		"transactions":[{"uuid":"6f1c2a4e-0b7d-4c3e-9a51-2d8e7f6b1c90","status":"APPROVED"}]},
		"code":"00000","message":"Operation performed successfully","route":"/dashboard","responseId":"http-nio-1"}`

	o, err := ParseOrderCallback([]byte(body))

	if err != nil || o.OrderID != "ORD-1" || o.PaymentStatus != PaymentApproved || o.InvoiceUUID != "inv-1" ||
		o.CreatedDate.Nanosecond() != 123000000 || len(o.Transactions) != 1 {
		t.Fatalf("got %+v, %v", o, err)
	}
	for _, bad := range []string{"", "not json", "{}", `{"payload":null}`, `{"payload":{"amount":1}}`, "[]", `{"payload":[]}`} {
		if _, err := ParseOrderCallback([]byte(bad)); !isInvalid(err) {
			t.Fatalf("%q: got %v", bad, err)
		}
	}
}

func TestRegionsMapToBaseURLAndCurrency(t *testing.T) {
	if RegionAzerbaijan.BaseURL() != "https://api.payriff.com" || RegionAzerbaijan.Currency() != CurrencyAZN {
		t.Fatal("azerbaijan")
	}
	if RegionSaudi.BaseURL() != "https://api.gccpay.sa" || RegionSaudi.Currency() != CurrencySAR {
		t.Fatal("saudi")
	}
	c, err := NewClient("k", WithRegion(RegionSaudi))
	if err != nil || c.Orders.t.baseURL != "https://api.gccpay.sa" {
		t.Fatalf("got %v", err)
	}
	c, err = NewClient("k")
	if err != nil || c.Orders.t.baseURL != "https://api.payriff.com" {
		t.Fatalf("got %v", err)
	}
	_, err = NewClient("k", WithRegion("ksa"))
	mustInvalid(t, err, "unknown region: ksa")
}

func TestSaudiDefaultsToSAR(t *testing.T) {
	m := newMockServer(t)
	m.stub("POST", "/api/v3/orders", 200, nil, okBody(`{"orderId":"ORD-1"}`))
	m.stub("POST", "/api/v3/autoPay", 200, nil, okBody(`{"orderId":"ORD-2"}`))
	m.stub("POST", "/api/v2/invoices", 200, nil, okBody(`{"invoiceUuid":"inv-1"}`))
	c := m.client(t, WithRegion(RegionSaudi))

	if _, err := c.Orders.Create(ctx, &CreateOrderParams{Amount: 1}); err != nil {
		t.Fatal(err)
	}
	if got := m.last(t).JSON(t)["currency"]; got != "SAR" {
		t.Fatalf("orders currency %v", got)
	}
	if _, err := c.Payments.AutoPay(ctx, &AutoPayParams{CardUUID: "card-1", Amount: 1, Description: "x"}); err != nil {
		t.Fatal(err)
	}
	if got := m.last(t).JSON(t)["currency"]; got != "SAR" {
		t.Fatalf("autoPay currency %v", got)
	}
	if _, err := c.Invoices.Create(ctx, &InvoiceCreateParams{Amount: 1}); err != nil {
		t.Fatal(err)
	}
	if got := m.last(t).JSON(t)["body"].(map[string]any)["currencyType"]; got != "SAR" {
		t.Fatalf("invoice currency %v", got)
	}
}

func TestExplicitCurrencyWinsOverRegion(t *testing.T) {
	m := newMockServer(t)
	m.stub("POST", "/api/v3/orders", 200, nil, okBody(`{"orderId":"ORD-1"}`))

	if _, err := m.client(t, WithRegion(RegionSaudi)).Orders.Create(ctx, &CreateOrderParams{Amount: 1, Currency: CurrencyUSD}); err != nil {
		t.Fatal(err)
	}

	if got := m.last(t).JSON(t)["currency"]; got != "USD" {
		t.Fatalf("currency %v", got)
	}
}
