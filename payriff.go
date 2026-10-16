package payriff

import (
	"net"
	"net/http"
	"strings"
	"time"
)

const Version = "0.2.0"

type Region string

const (
	RegionAzerbaijan Region = "azerbaijan"
	RegionSaudi      Region = "saudi"
)

func (r Region) BaseURL() string {
	switch r {
	case RegionAzerbaijan:
		return "https://api.payriff.com"
	case RegionSaudi:
		return "https://api.gccpay.sa"
	}
	return ""
}

func (r Region) Currency() Currency {
	switch r {
	case RegionAzerbaijan:
		return CurrencyAZN
	case RegionSaudi:
		return CurrencySAR
	}
	return ""
}

type Client struct {
	Orders       *OrdersService
	Payments     *PaymentsService
	Cards        *CardsService
	Transactions *TransactionsService
	Payouts      *PayoutsService
	Invoices     *InvoicesService
}

type config struct {
	merchantID     string
	timeout        time.Duration
	connectTimeout time.Duration
	region         Region
	baseURL        string
	httpClient     *http.Client
}

type Option func(*config)

func WithMerchantID(merchantID string) Option {
	return func(c *config) { c.merchantID = merchantID }
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *config) { c.timeout = timeout }
}

func WithConnectTimeout(timeout time.Duration) Option {
	return func(c *config) { c.connectTimeout = timeout }
}

func WithRegion(region Region) Option {
	return func(c *config) { c.region = region }
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *config) { c.httpClient = httpClient }
}

func WithBaseURL(baseURL string) Option {
	return func(c *config) { c.baseURL = baseURL }
}

func NewClient(appKey string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(appKey) == "" {
		return nil, invalid("app key is required")
	}
	cfg := config{
		timeout:        60 * time.Second,
		connectTimeout: 10 * time.Second,
		region:         RegionAzerbaijan,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.timeout <= 0 || cfg.connectTimeout <= 0 {
		return nil, invalid("timeouts must be positive")
	}
	if cfg.region.BaseURL() == "" {
		return nil, invalid("unknown region: " + string(cfg.region))
	}
	if cfg.baseURL == "" {
		cfg.baseURL = cfg.region.BaseURL()
	}
	t := &transport{
		baseURL:    strings.TrimRight(cfg.baseURL, "/"),
		appKey:     appKey,
		merchantID: cfg.merchantID,
		httpClient: httpClient(cfg),
	}
	return newClient(t, cfg.region.Currency()), nil
}

func httpClient(cfg config) *http.Client {
	if cfg.httpClient != nil {
		c := *cfg.httpClient
		c.CheckRedirect = noRedirect
		return &c
	}
	base := http.DefaultTransport.(*http.Transport).Clone()
	base.DialContext = (&net.Dialer{Timeout: cfg.connectTimeout, KeepAlive: 30 * time.Second}).DialContext
	return &http.Client{Transport: base, Timeout: cfg.timeout, CheckRedirect: noRedirect}
}

func newClient(t *transport, currency Currency) *Client {
	return &Client{
		Orders:       &OrdersService{t: t, currency: currency},
		Payments:     &PaymentsService{t: t, currency: currency},
		Cards:        &CardsService{t: t},
		Transactions: &TransactionsService{t: t},
		Payouts:      &PayoutsService{t: t},
		Invoices:     &InvoicesService{t: t, currency: currency},
	}
}
