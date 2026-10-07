package payriff

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

var httpClientWithRedirects = http.Client{}

func isInvalid(err error) bool {
	return errors.Is(err, ErrInvalidArgument)
}

func TestUnwrapsPayloadOnSuccess(t *testing.T) {
	m := newMockServer(t)
	m.ok("GET", "/api/v3/orders/ORD-1", map[string]any{"orderId": "ORD-1", "paymentStatus": "APPROVED"})

	order, err := m.client(t).Orders.Get(ctx, "ORD-1")

	if err != nil || order.OrderID != "ORD-1" || order.PaymentStatus != PaymentApproved {
		t.Fatalf("got %+v, %v", order, err)
	}
}

func TestSendsAuthAndClientHeaders(t *testing.T) {
	m := newMockServer(t)
	m.ok("GET", "/api/v3/orders/ORD-1", map[string]any{"orderId": "ORD-1"})

	if _, err := m.client(t).Orders.Get(ctx, "ORD-1"); err != nil {
		t.Fatal(err)
	}

	h := m.last(t).Headers
	if h.Get("Authorization") != "app-key" || h.Get("User-Agent") != "payriff-go/"+Version ||
		h.Get("Accept") != "application/json" || h.Get("Content-Type") != "" {
		t.Fatalf("unexpected headers: %v", h)
	}
}

func TestEncodesPathSegmentsAndQuery(t *testing.T) {
	m := newMockServer(t)
	m.ok("PATCH", "/api/v3/expire-status", nil)
	m.ok("GET", "/api/v3/orders/rrn%201%2F2/rrn", map[string]any{"orderId": "ORD-1"})

	if err := m.client(t).Orders.Expire(ctx, "ORD 1/2+3"); err != nil {
		t.Fatal(err)
	}
	if got := m.last(t).URL; got != "/api/v3/expire-status?orderId=ORD%201%2F2%2B3" {
		t.Fatalf("url = %s", got)
	}
	if _, err := m.client(t).Orders.GetByRequestRRN(ctx, "rrn 1/2"); err != nil {
		t.Fatal(err)
	}
	if got := m.last(t).URL; got != "/api/v3/orders/rrn%201%2F2/rrn" {
		t.Fatalf("url = %s", got)
	}
}

func TestRejectsBlankPathSegmentBeforeSending(t *testing.T) {
	m := newMockServer(t)

	_, err := m.client(t).Orders.Get(ctx, " ")

	mustInvalid(t, err, "orderID must not be blank")
	if m.count() != 0 {
		t.Fatal("request was sent")
	}
}

func TestMerchantEnvelopeRequiresMerchantID(t *testing.T) {
	m := newMockServer(t)
	c, _ := NewClient("app-key", WithBaseURL(m.URL))

	_, err := c.Invoices.Get(ctx, "inv-1")

	mustInvalid(t, err, "merchant ID must be configured on the client for this operation")
	if m.count() != 0 {
		t.Fatal("request was sent")
	}
}

func TestMapsFailuresToErrorKinds(t *testing.T) {
	cases := []struct {
		status   int
		code     string
		kind     ErrorKind
		sentinel error
	}{
		{200, "15000", KindAPI, nil},
		{401, "14010", KindAuthentication, ErrAuthentication},
		{401, "14013", KindAuthentication, ErrAuthentication},
		{200, "14014", KindAuthentication, ErrAuthentication},
		{401, "14015", KindAuthentication, ErrAuthentication},
		{400, "15400", KindValidation, ErrValidation},
		{400, "99999", KindValidation, ErrValidation},
		{403, "99999", KindAuthentication, ErrAuthentication},
		{402, "01000", KindRequestRejected, ErrRequestRejected},
		{402, "01200", KindInsufficientBalance, ErrInsufficientBalance},
		{402, "01300", KindPayoutLimit, ErrPayoutLimit},
		{402, "01400", KindPayoutLimit, ErrPayoutLimit},
		{402, "01500", KindPayoutLimit, ErrPayoutLimit},
		{500, "15000", KindAPI, nil},
		{503, "15000", KindAPI, nil},
	}
	for _, tc := range cases {
		t.Run(tc.code+"/"+tc.kind.String(), func(t *testing.T) {
			m := newMockServer(t)
			m.stub("GET", "/api/v3/orders/ORD-1", tc.status, nil, map[string]any{"code": tc.code, "message": "Failure reason", "responseId": "resp-1"})

			_, err := m.client(t).Orders.Get(ctx, "ORD-1")

			var perr *Error
			if !errors.As(err, &perr) {
				t.Fatalf("expected *Error, got %v", err)
			}
			if perr.Kind != tc.kind || perr.HTTPStatus != tc.status || perr.Code != tc.code || perr.ResponseID != "resp-1" || perr.Message != "Failure reason" {
				t.Fatalf("got %+v", perr)
			}
			if tc.sentinel != nil && !errors.Is(err, tc.sentinel) {
				t.Fatalf("errors.Is(%v) = false", tc.sentinel)
			}
			if errors.Is(err, ErrConnection) || errors.Is(err, ErrInvalidArgument) {
				t.Fatal("matched an unrelated sentinel")
			}
		})
	}
}

func TestMissingMessageFallsBackToHTTPStatus(t *testing.T) {
	m := newMockServer(t)
	m.stub("GET", "/api/v3/orders/ORD-1", 500, nil, map[string]any{"code": "15000"})

	_, err := m.client(t).Orders.Get(ctx, "ORD-1")

	var perr *Error
	if !errors.As(err, &perr) || perr.Message != "Payriff request failed (HTTP 500)" {
		t.Fatalf("got %v", err)
	}
}

func TestNonJSONErrorBodyKeepsStatusAndTruncates(t *testing.T) {
	m := newMockServer(t)
	m.stub("GET", "/api/v3/orders/ORD-1", 502, map[string]string{"Content-Type": "text/html"}, strings.Repeat("x", 600))

	_, err := m.client(t).Orders.Get(ctx, "ORD-1")

	var perr *Error
	if !errors.As(err, &perr) || perr.HTTPStatus != 502 || perr.Code != "" || perr.Message != strings.Repeat("x", 500)+"..." {
		t.Fatalf("got %+v", perr)
	}
}

func TestSuccessStatusWithoutEnvelopeIsRejected(t *testing.T) {
	m := newMockServer(t)
	m.stub("GET", "/api/v3/orders/ORD-1", 200, nil, map[string]any{"orderId": "ORD-1"})

	_, err := m.client(t).Orders.Get(ctx, "ORD-1")

	var perr *Error
	if !errors.As(err, &perr) || perr.Kind != KindAPI || perr.Message != "unexpected response from Payriff" {
		t.Fatalf("got %v", err)
	}
}

func TestRedirectsAreNotFollowed(t *testing.T) {
	m := newMockServer(t)
	m.stub("GET", "/api/v3/orders/ORD-1", 302, map[string]string{"Location": "/api/v3/orders/OTHER"}, map[string]any{"code": "15000", "message": "Redirect"})

	_, err := m.client(t).Orders.Get(ctx, "ORD-1")

	var perr *Error
	if !errors.As(err, &perr) || perr.Message != "Redirect" || m.count() != 1 {
		t.Fatalf("got %v after %d requests", err, m.count())
	}
}

func TestDownloadReturnsBytes(t *testing.T) {
	m := newMockServer(t)
	m.stub("GET", "/api/v3/acquiring/receipt/ORD-1", 200, map[string]string{"Content-Type": "application/pdf"}, []byte("%PDF-1.7"))

	pdf, err := m.client(t).Orders.DownloadReceipt(ctx, "ORD-1")

	if err != nil || string(pdf) != "%PDF-1.7" || m.last(t).Headers.Get("Accept") != "application/pdf, application/json" {
		t.Fatalf("got %q, %v", pdf, err)
	}
}

func TestDownloadMapsEnvelopeError(t *testing.T) {
	m := newMockServer(t)
	m.stub("GET", "/api/v3/acquiring/receipt/ORD-1", 400, nil, map[string]any{"code": "15400", "message": "Receipt not found"})

	_, err := m.client(t).Orders.DownloadReceipt(ctx, "ORD-1")

	if !errors.Is(err, ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestConnectionFailure(t *testing.T) {
	c, _ := NewClient("app-key", WithBaseURL("http://127.0.0.1:1"))

	_, err := c.Orders.Get(ctx, "ORD-1")

	var perr *Error
	if !errors.Is(err, ErrConnection) || !errors.As(err, &perr) || perr.HTTPStatus != 0 || perr.Err == nil {
		t.Fatalf("got %v", err)
	}
}

func TestTimeout(t *testing.T) {
	c, _ := NewClient("app-key", WithBaseURL("http://10.255.255.1"), WithConnectTimeout(50*time.Millisecond))

	_, err := c.Orders.Get(ctx, "ORD-1")

	if !errors.Is(err, ErrConnection) {
		t.Fatalf("got %v", err)
	}
}

func TestCustomHTTPClientStillDoesNotFollowRedirects(t *testing.T) {
	m := newMockServer(t)
	m.stub("GET", "/api/v3/orders/ORD-1", 302, map[string]string{"Location": "/elsewhere"}, map[string]any{"code": "15000", "message": "Redirect"})
	custom := &httpClientWithRedirects

	_, err := m.client(t, WithHTTPClient(custom)).Orders.Get(ctx, "ORD-1")

	if err == nil || m.count() != 1 {
		t.Fatalf("got %v after %d requests", err, m.count())
	}
	if custom.CheckRedirect != nil {
		t.Fatal("caller's http.Client was modified")
	}
}

func TestClientValidation(t *testing.T) {
	_, err := NewClient("  ")
	mustInvalid(t, err, "app key is required")

	_, err = NewClient("k", WithCardEncryptionKey("not-a-key"))
	mustInvalid(t, err, "invalid RSA public key")

	c, err := NewClient("k")
	if err != nil || c.Orders == nil || c.Payments == nil || c.Cards == nil || c.Transactions == nil || c.Payouts == nil || c.Invoices == nil {
		t.Fatalf("got %+v, %v", c, err)
	}
}
