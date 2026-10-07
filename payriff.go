package payriff

import (
	"crypto/rsa"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	Version       = "0.1.0"
	ProductionURL = "https://api.payriff.com"
)

type Client struct {
	Orders       *OrdersService
	Payments     *PaymentsService
	Cards        *CardsService
	Transactions *TransactionsService
	Payouts      *PayoutsService
	Invoices     *InvoicesService
}

type config struct {
	merchantID        string
	timeout           time.Duration
	connectTimeout    time.Duration
	cardEncryptionKey string
	baseURL           string
	httpClient        *http.Client
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

func WithCardEncryptionKey(base64OrPEM string) Option {
	return func(c *config) { c.cardEncryptionKey = base64OrPEM }
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
		timeout:           60 * time.Second,
		connectTimeout:    10 * time.Second,
		cardEncryptionKey: productionCardEncryptionKey,
		baseURL:           ProductionURL,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.timeout <= 0 || cfg.connectTimeout <= 0 {
		return nil, invalid("timeouts must be positive")
	}
	key, err := parsePublicKey(cfg.cardEncryptionKey)
	if err != nil {
		return nil, err
	}
	t := &transport{
		baseURL:    strings.TrimRight(cfg.baseURL, "/"),
		appKey:     appKey,
		merchantID: cfg.merchantID,
		httpClient: httpClient(cfg),
	}
	return newClient(t, key), nil
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

func newClient(t *transport, key *rsa.PublicKey) *Client {
	return &Client{
		Orders:       &OrdersService{t: t},
		Payments:     &PaymentsService{t: t, key: key},
		Cards:        &CardsService{t: t},
		Transactions: &TransactionsService{t: t},
		Payouts:      &PayoutsService{t: t},
		Invoices:     &InvoicesService{t: t},
	}
}
