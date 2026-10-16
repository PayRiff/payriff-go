# Payriff Go SDK

Go client for the Payriff merchant API: orders, saved-card payments, saved cards,
transactions, payouts and invoices.

- Go 1.22 or newer
- No dependencies outside the standard library
- Talks to `https://api.payriff.com` (Azerbaijan) or `https://api.gccpay.sa` (Saudi Arabia)

## Installation

```bash
go get github.com/payriff/payriff-go
```

## Quick start

```go
import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/payriff/payriff-go"
)

client, err := payriff.NewClient(os.Getenv("PAYRIFF_APP_KEY"))
if err != nil {
	log.Fatal(err)
}

order, err := client.Orders.Create(ctx, &payriff.CreateOrderParams{
	Amount:      10,
	Description: "Order #1001",
	CallbackURL: "https://shop.example/payriff/callback",
	RequestRRN:  "order-1001",
})
if err != nil {
	return err
}

// Send the customer to the hosted payment page
http.Redirect(w, r, order.PaymentURL, http.StatusSeeOther)
```

Create one `Client` and reuse it. It is safe for concurrent use. Every call takes a `context.Context`,
so you can cancel it or set a deadline.

## Configuration

| Option | Default | Notes |
|---|---|---|
| `WithMerchantID(id)` | — | Required for payouts and invoices (e.g. `ES1000000`). |
| `WithTimeout(d)` | 60 s | Total request timeout. Bank operations can take tens of seconds. |
| `WithConnectTimeout(d)` | 10 s | TCP connect timeout. |
| `WithHTTPClient(c)` | — | Use your own `*http.Client` (proxy, TLS, tracing). The SDK copies it and disables redirects on the copy. |
| `WithRegion(r)` | `payriff.RegionAzerbaijan` | API region. Sets the base URL and the default currency. |
| `WithBaseURL(u)` | from the region | Override the API URL, e.g. for testing. The currency still comes from the region. |

Keep the app key in an environment variable or secret store. Never commit it.

### Saudi Arabia

```go
client, err := payriff.NewClient(os.Getenv("PAYRIFF_APP_KEY"), payriff.WithRegion(payriff.RegionSaudi))
```

| Region | API | Default currency |
|---|---|---|
| `payriff.RegionAzerbaijan` | `https://api.payriff.com` | `AZN` |
| `payriff.RegionSaudi` | `https://api.gccpay.sa` | `SAR` |

Use the app key from the dashboard of the same region (`dashboard.gccpay.sa` for Saudi Arabia).
A `Currency` set on a request always wins over the region default.

## Orders

```go
created, err := client.Orders.Create(ctx, &payriff.CreateOrderParams{
	Amount:      25,
	Currency:    payriff.CurrencyAZN,        // default: the region currency
	Operation:   payriff.OperationPreAuth,   // default PURCHASE
	Language:    payriff.LanguageAZ,
	Description: "Booking #77",
	CallbackURL: "https://shop.example/payriff/callback",
	Metadata:    map[string]string{"bookingId": "77"},
})

info, err := client.Orders.Get(ctx, created.OrderID)
byRef, err := client.Orders.GetByRequestRRN(ctx, "order-1001")   // the RequestRRN you sent on create

err = client.Orders.Complete(ctx, &payriff.CompleteParams{OrderID: created.OrderID, Amount: 25})
err = client.Orders.Refund(ctx, &payriff.RefundParams{OrderID: created.OrderID, Amount: 5, RefundReason: "Partial return"})
err = client.Orders.Expire(ctx, created.OrderID)                  // cancel an unpaid order

receiptPDF, err := client.Orders.DownloadReceipt(ctx, created.OrderID)
```

`OrderInfo.PaymentStatus` is a `payriff.PaymentStatus` such as `PaymentApproved`, `PaymentDeclined`,
`PaymentPreAuthApproved` or `PaymentRefunded`. Values this SDK version does not know yet are passed
through unchanged.

Amounts are `float64` in major units (`10.5` = 10.50 AZN or SAR). Optional booleans are pointers; use
`payriff.Bool(true)`.

## Charging a saved card

```go
charge, err := client.Payments.AutoPay(ctx, &payriff.AutoPayParams{
	CardUUID:    savedCardUUID,
	Amount:      9.99,
	Description: "Monthly subscription",
	RequestRRN:  "sub-2026-10",
})
```

## Saved cards

```go
session, err := client.Cards.Save(ctx, &payriff.CardSaveParams{
	CustomerRef:    "customer-42",
	CallbackURL:    "https://shop.example/payriff/card-saved",
	IdempotencyKey: "customer-42-card-1",
})
// Redirect the customer to session.PaymentURL to verify the card.

details, err := client.Cards.GetSave(ctx, session.CardSaveID)
if details.Status == payriff.CardSaveVerified {
	cardUUID := details.CardUUID   // store it; use with AutoPay
}

cards, err := client.Cards.List(ctx, "customer-42")
err = client.Cards.Delete(ctx, cards[0].CardUUID)
```

## Transactions

```go
page, err := client.Transactions.List(ctx, &payriff.TransactionFilter{
	Status: payriff.PaymentApproved,
	From:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local),
	To:     time.Date(2026, 9, 30, 0, 0, 0, 0, time.Local),
	Page:   0,
	Size:   20,                    // server maximum is 20
})

for _, tx := range page.Content {
	fmt.Println(tx.OrderID, tx.Amount)
}
```

## Payouts

Payouts require a merchant ID on the client.

```go
client, err := payriff.NewClient(os.Getenv("PAYRIFF_APP_KEY"), payriff.WithMerchantID(os.Getenv("PAYRIFF_MERCHANT_ID")))

maskedName, err := client.Payouts.CheckCardholder(ctx, "4169741330151979")   // e.g. "J*** D**"

payout, err := client.Payouts.Create(ctx, &payriff.PayoutParams{
	TransferAmount: 50,            // minimum 1
	Description:    "Refund for order #1001",
	FullName:       "JOHN DOE",
	FinCode:        "1AB2C3D",
	CardPAN:        "4169741330151979",
	RequestRRN:     "payout-1001",
	IdempotencyKey: "payout-1001",
})

status, err := client.Payouts.GetByRequestRRN(ctx, "payout-1001")
history, err := client.Payouts.List(ctx, &payriff.PayoutFilter{Status: payriff.TransferSuccess})
receipt, err := client.Payouts.DownloadReceipt(ctx, "payout-1001")
```

## Invoices

Invoices require a merchant ID on the client.

```go
invoice, err := client.Invoices.Create(ctx, &payriff.InvoiceCreateParams{
	Amount:      15,
	FullName:    "JOHN DOE",
	PhoneNumber: "+994501234567",
	Description: "Consultation",
	ExpireDate:  time.Now().AddDate(0, 0, 7),
	SendSMS:     payriff.Bool(true),
})

link := invoice.PaymentURL   // share with the customer
details, err := client.Invoices.Get(ctx, invoice.InvoiceUUID)
```

## Callbacks

When an order changes state, Payriff POSTs JSON to the `CallbackURL` you set on the order.

```go
http.HandleFunc("POST /payriff/callback", func(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	notified, err := payriff.ParseOrderCallback(body)
	if err != nil {
		http.Error(w, "bad callback", http.StatusBadRequest)
		return
	}

	// Callbacks are not signed: confirm the state with Payriff before fulfilling.
	confirmed, err := client.Orders.Get(r.Context(), notified.OrderID)
	if err != nil {
		http.Error(w, "retry later", http.StatusServiceUnavailable)
		return
	}
	if confirmed.PaymentStatus == payriff.PaymentApproved {
		// fulfil the order (make this idempotent: the same callback can arrive more than once)
	}
	w.WriteHeader(http.StatusOK)
})
```

## Errors

Payriff failures are returned as `*payriff.Error`, which carries `Kind`, `HTTPStatus`, `Code`
(Payriff result code) and `ResponseID` (quote it when contacting support). Match them with
`errors.Is` or inspect them with `errors.As`:

| Sentinel | When |
|---|---|
| `ErrAuthentication` | App key rejected (`14010`, `14013`, `14014`, `14015`) |
| `ErrValidation` | Invalid request (`15400` or HTTP 400) |
| `ErrRequestRejected` | Business refusal (`01000`), e.g. application under review |
| `ErrInsufficientBalance` | Not enough wallet balance for a payout (`01200`) |
| `ErrPayoutLimit` | Payout limit reached (`01300`, `01400`, `01500`) |
| `ErrConnection` | No response: network error or timeout |
| `ErrInvalidArgument` | Invalid parameters, rejected before any request is sent |

Any other Payriff failure is a `*payriff.Error` with `Kind == payriff.KindAPI`. Payriff can report a
failure with HTTP 200; the SDK checks the result code in the body, so checking `err` is enough.

```go
err := client.Orders.Refund(ctx, &payriff.RefundParams{OrderID: orderID})
var perr *payriff.Error
switch {
case err == nil:
case errors.Is(err, payriff.ErrValidation):
	log.Printf("refund rejected: %v", err)
case errors.Is(err, payriff.ErrConnection):
	// Outcome unknown: check client.Orders.Get(ctx, orderID) before retrying
case errors.As(err, &perr):
	log.Printf("payriff error %s responseId=%s", perr.Code, perr.ResponseID)
default:
	return err
}
```

### Retries

The SDK never retries on its own, because payment calls are not safe to repeat blindly. After an
`ErrConnection`, look the operation up first (`Orders.GetByRequestRRN`, `Payouts.GetByRequestRRN`)
and retry only if it does not exist. Set `RequestRRN` / `IdempotencyKey` on every request so that
this lookup is possible.

## Development

```bash
go test -race ./...
```

## License

MIT