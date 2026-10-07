package payriff

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type Currency string

const (
	CurrencyAZN Currency = "AZN"
	CurrencyUSD Currency = "USD"
	CurrencyEUR Currency = "EUR"
	CurrencyPKR Currency = "PKR"
	CurrencyAED Currency = "AED"
	CurrencySAR Currency = "SAR"
)

type Language string

const (
	LanguageAZ Language = "AZ"
	LanguageEN Language = "EN"
	LanguageRU Language = "RU"
	LanguageAR Language = "AR"
)

type Operation string

const (
	OperationPurchase Operation = "PURCHASE"
	OperationPreAuth  Operation = "PRE_AUTH"
	OperationComplete Operation = "COMPLETE"
	OperationRefund   Operation = "REFUND"
	OperationReverse  Operation = "REVERSE"
)

type AutoPaymentType string

const (
	AutoPaymentNone      AutoPaymentType = "NONE"
	AutoPaymentDefault   AutoPaymentType = "DEFAULT"
	AutoPaymentRecurring AutoPaymentType = "RECURRING"
)

type InstallmentProductType string

const (
	InstallmentBirKart InstallmentProductType = "BIRKART"
	InstallmentAllBali InstallmentProductType = "ALLBALI"
	InstallmentBolKart InstallmentProductType = "BOLKART"
	InstallmentTamKart InstallmentProductType = "TAMKART"
)

type InstallmentPeriod string

func InstallmentMonths(months int) InstallmentPeriod {
	if months <= 0 {
		return "PERIOD"
	}
	return InstallmentPeriod("PERIOD_" + strconv.Itoa(months))
}

type PaymentStatus string

const (
	PaymentCreated          PaymentStatus = "CREATED"
	PaymentApproved         PaymentStatus = "APPROVED"
	PaymentCanceled         PaymentStatus = "CANCELED"
	PaymentDeclined         PaymentStatus = "DECLINED"
	PaymentRefunded         PaymentStatus = "REFUNDED"
	PaymentPreAuthApproved  PaymentStatus = "PREAUTH_APPROVED"
	PaymentExpired          PaymentStatus = "EXPIRED"
	PaymentReverse          PaymentStatus = "REVERSE"
	PaymentPartialRefund    PaymentStatus = "PARTIAL_REFUND"
	PaymentPartial          PaymentStatus = "PARTIAL"
	PaymentAccepted         PaymentStatus = "ACCEPTED"
	PaymentRefundInProgress PaymentStatus = "REFUND_IN_PROGRESS"
	PaymentCash             PaymentStatus = "CASH"
	PaymentPending          PaymentStatus = "PENDING"
	PaymentPreAuthExpired   PaymentStatus = "PREAUTH_EXPIRED"
	PaymentInReview         PaymentStatus = "IN_REVIEW"
)

type CardSaveStatus string

const (
	CardSaveCreated       CardSaveStatus = "CREATED"
	CardSaveVerified      CardSaveStatus = "VERIFIED"
	CardSaveReversed      CardSaveStatus = "REVERSED"
	CardSaveReverseFailed CardSaveStatus = "REVERSE_FAILED"
	CardSaveDeclined      CardSaveStatus = "DECLINED"
	CardSaveExpired       CardSaveStatus = "EXPIRED"
)

type TransferState string

const (
	TransferCreated                  TransferState = "CREATED"
	TransferInProgress               TransferState = "IN_PROGRESS"
	TransferNotFound                 TransferState = "NOT_FOUND"
	TransferFail                     TransferState = "FAIL"
	TransferSuccess                  TransferState = "SUCCESS"
	TransferDailyPayoutLimitExceeded TransferState = "DAILY_PAYOUT_LIMIT_EXCEEDED"
)

type InvoiceStatus string

const (
	InvoicePending  InvoiceStatus = "PENDING"
	InvoiceError    InvoiceStatus = "ERROR"
	InvoiceExpired  InvoiceStatus = "EXPIRED"
	InvoicePartial  InvoiceStatus = "PARTIAL"
	InvoiceComplete InvoiceStatus = "COMPLETE"
	InvoiceCash     InvoiceStatus = "CASH"
	InvoiceDeclined InvoiceStatus = "DECLINED"
	InvoiceCanceled InvoiceStatus = "CANCELED"
)

func Bool(v bool) *bool {
	return &v
}

type DateTime struct {
	time.Time
}

var dateTimeLayouts = []string{
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02",
}

func (d *DateTime) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	var s string
	if json.Unmarshal(data, &s) != nil {
		return nil
	}
	s = strings.TrimSpace(s)
	for _, layout := range dateTimeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			d.Time = t
			return nil
		}
	}
	return nil
}

func (d DateTime) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.Format("2006-01-02T15:04:05.999999999"))
}

type Installment struct {
	Type   InstallmentProductType `json:"type"`
	Period InstallmentPeriod      `json:"period"`
}

type Page[T any] struct {
	Content       []T   `json:"content"`
	TotalElements int64 `json:"totalElements"`
	TotalPages    int   `json:"totalPages"`
	Number        int   `json:"number"`
	Size          int   `json:"size"`
	First         bool  `json:"first"`
	Last          bool  `json:"last"`
}

type CardDetails struct {
	MaskedPAN      string `json:"maskedPan"`
	Brand          string `json:"brand"`
	UUID           string `json:"uuid"`
	CardHolderName string `json:"cardHolderName"`
	PhoneNumber    string `json:"phoneNumber"`
}

type OrderTransaction struct {
	UUID                string       `json:"uuid"`
	CreatedDate         DateTime     `json:"createdDate"`
	Status              string       `json:"status"`
	Channel             string       `json:"channel"`
	ChannelType         string       `json:"channelType"`
	RequestRRN          string       `json:"requestRrn"`
	ResponseRRN         string       `json:"responseRrn"`
	ExternalRRN         string       `json:"externalRrn"`
	PAN                 string       `json:"pan"`
	PaymentWay          string       `json:"paymentWay"`
	CardDetails         *CardDetails `json:"cardDetails"`
	CardUUID            string       `json:"cardUuid"`
	RecurrenceID        int          `json:"recurrenceId"`
	ResponseDescription string       `json:"responseDescription"`
	MerchantCategory    string       `json:"merchantCategory"`
	Installment         *Installment `json:"installment"`
}

type OrderInfo struct {
	OrderID               string             `json:"orderId"`
	ExternalTransactionID string             `json:"externalTransactionId"`
	InvoiceUUID           string             `json:"invoiceUuid"`
	Amount                float64            `json:"amount"`
	Currency              Currency           `json:"currencyType"`
	MerchantName          string             `json:"merchantName"`
	Commission            float64            `json:"commission"`
	CommissionRate        float64            `json:"commissionRate"`
	PaidAmount            float64            `json:"paidAmount"`
	ExtraPayment          float64            `json:"extraPayment"`
	OperationType         Operation          `json:"operationType"`
	PaymentStatus         PaymentStatus      `json:"paymentStatus"`
	Auto                  bool               `json:"auto"`
	CreatedDate           DateTime           `json:"createdDate"`
	Description           string             `json:"description"`
	Metadata              string             `json:"metadata"`
	IdempotencyKey        string             `json:"idempotencyKey"`
	Transactions          []OrderTransaction `json:"transactions"`
}

type CreateOrderResponse struct {
	OrderID        string  `json:"orderId"`
	SessionID      string  `json:"sessionId"`
	PaymentURL     string  `json:"paymentUrl"`
	PreviewURL     string  `json:"previewUrl"`
	TransactionID  int64   `json:"transactionId"`
	CommissionRate float64 `json:"comissionRate"`
	Amount         float64 `json:"amount"`
	Fee            float64 `json:"fee"`
	TotalAmount    float64 `json:"totalAmount"`
}

type DirectPayResponse struct {
	OrderID     string            `json:"orderId"`
	ThreeDS     bool              `json:"threeDS"`
	Redirect    bool              `json:"redirect"`
	RedirectURL string            `json:"redirectUrl"`
	PaymentURL  string            `json:"paymentUrl"`
	Transaction *OrderTransaction `json:"transactionResponse"`
}

type TransactionResponse struct {
	Redirect          bool               `json:"redirect"`
	RedirectURL       string             `json:"redirectUrl"`
	ThreeDS           bool               `json:"threeDS"`
	AccessURL         string             `json:"accessUrl"`
	Channel           string             `json:"channel"`
	TransactionResult *DirectPayResponse `json:"transactionResult"`
}

type AutoPayResponse struct {
	OrderID             string               `json:"orderId"`
	PaymentURL          string               `json:"paymentUrl"`
	Description         string               `json:"description"`
	Amount              float64              `json:"amount"`
	Commission          float64              `json:"commission"`
	CommissionRate      float64              `json:"commissionRate"`
	Currency            Currency             `json:"currencyType"`
	OperationType       Operation            `json:"operationType"`
	PaymentStatus       PaymentStatus        `json:"paymentStatus"`
	Auto                bool                 `json:"auto"`
	CreatedDate         DateTime             `json:"createdDate"`
	Transactions        []OrderTransaction   `json:"transactions"`
	TransactionResponse *TransactionResponse `json:"transactionResponseDto"`
}

type CardSaveResponse struct {
	CardSaveID string         `json:"cardSaveId"`
	OrderID    string         `json:"orderId"`
	SessionID  string         `json:"sessionId"`
	PaymentURL string         `json:"paymentUrl"`
	Amount     float64        `json:"amount"`
	Currency   Currency       `json:"currency"`
	Status     CardSaveStatus `json:"status"`
}

type CardSaveDetails struct {
	CardSaveID   string         `json:"cardSaveId"`
	OrderID      string         `json:"orderId"`
	Status       CardSaveStatus `json:"status"`
	CardUUID     string         `json:"cardUuid"`
	MaskedPAN    string         `json:"maskedPan"`
	CardBrand    string         `json:"cardBrand"`
	Amount       float64        `json:"amount"`
	Currency     Currency       `json:"currency"`
	CustomerRef  string         `json:"customerRef"`
	CreatedDate  DateTime       `json:"createdDate"`
	VerifiedDate DateTime       `json:"verifiedDate"`
}

type SavedCard struct {
	CardUUID    string   `json:"cardUuid"`
	MaskedPAN   string   `json:"maskedPan"`
	CardBrand   string   `json:"cardBrand"`
	CreatedDate DateTime `json:"createdDate"`
}

type Transaction struct {
	ID                  int64         `json:"id"`
	ApplicationID       int64         `json:"applicationId"`
	OrderID             string        `json:"orderId"`
	SessionID           string        `json:"sessionId"`
	UUID                string        `json:"uuid"`
	RRN                 string        `json:"rrn"`
	ExternalRRN         string        `json:"externalRrn"`
	Amount              float64       `json:"amount"`
	PaidAmount          float64       `json:"paidAmount"`
	RefundAmount        float64       `json:"refundAmount"`
	RestOfAmount        float64       `json:"restOfAmount"`
	AmountWithoutFee    float64       `json:"amountWithoutFee"`
	PayriffAmount       float64       `json:"payriffAmount"`
	CommissionRate      float64       `json:"commissionRate"`
	ExtraPayment        float64       `json:"extra_payment"`
	Currency            Currency      `json:"currencyType"`
	PaymentStatus       PaymentStatus `json:"paymentStatus"`
	PaymentSource       string        `json:"paymentSource"`
	Description         string        `json:"description"`
	ResponseDescription string        `json:"responseDescription"`
	OrderLanguage       Language      `json:"orderLanguage"`
	TariffType          string        `json:"tariffType"`
	Source              string        `json:"source"`
	FullName            string        `json:"fullName"`
	PhoneNumber         string        `json:"phoneNumber"`
	BookingID           string        `json:"bookingId"`
	InvoiceCode         string        `json:"invoiceCode"`
	PAN                 string        `json:"pan"`
	CardBrand           string        `json:"card_brand"`
	PaymentRoute        string        `json:"payment_route"`
	PaymentWay          string        `json:"payment_way"`
	TransitID           string        `json:"transitId"`
	CreatedDate         string        `json:"createdDate"`
	LastModifiedDate    string        `json:"lastModifiedDate"`
}

type PayoutResult struct {
	FinalState            string  `json:"_final"`
	State                 string  `json:"state"`
	StateDescription      string  `json:"stateDescription"`
	CurrentDepositBalance float64 `json:"currentDepositBalance"`
	WalletHistoryID       int64   `json:"walletHistoryId"`
	BankName              string  `json:"bankName"`
}

type PayoutStatus struct {
	BankName       string        `json:"bankName"`
	State          TransferState `json:"state"`
	CardPAN        string        `json:"cardPan"`
	TransferAmount float64       `json:"transferAmount"`
	CreatedDate    string        `json:"createdDate"`
	FormattedDate  string        `json:"formattedDate"`
	TransferType   string        `json:"transferType"`
	Merchant       string        `json:"merchant"`
	Description    string        `json:"description"`
	FullName       string        `json:"fullName"`
	FinCode        string        `json:"finCode"`
}

type PayoutSummary struct {
	ID               int64         `json:"id"`
	RequestRRN       string        `json:"requestRrn"`
	TransferAmount   float64       `json:"transferAmount"`
	AmountWithFee    float64       `json:"amountWithFee"`
	Fee              float64       `json:"fee"`
	CreatedDate      DateTime      `json:"createdDate"`
	State            TransferState `json:"state"`
	StateDescription string        `json:"stateDescription"`
	CardPAN          string        `json:"cardPan"`
	FinCode          string        `json:"finCode"`
	FullName         string        `json:"fullName"`
	Description      string        `json:"description"`
}

type Invoice struct {
	ID                    int64         `json:"id"`
	MerchantID            string        `json:"merchantId"`
	UUID                  string        `json:"uuid"`
	InvoiceUUID           string        `json:"invoiceUuid"`
	InvoiceCode           string        `json:"invoiceCode"`
	Status                InvoiceStatus `json:"invoiceStatus"`
	PaymentURL            string        `json:"paymentUrl"`
	Amount                float64       `json:"amount"`
	TotalAmount           float64       `json:"totalAmount"`
	PayriffAmount         float64       `json:"payriffAmount"`
	PayriffFee            float64       `json:"payriffFee"`
	PayriffFixedFeeAmount float64       `json:"payriffFixedFeeAmount"`
	Currency              Currency      `json:"currencyType"`
	Language              Language      `json:"languageType"`
	PaymentType           string        `json:"paymentType"`
	FullName              string        `json:"fullName"`
	Email                 string        `json:"email"`
	PhoneNumber           string        `json:"phoneNumber"`
	Description           string        `json:"description"`
	CustomMessage         string        `json:"customMessage"`
	ExpireDate            DateTime      `json:"expireDate"`
	CreatedDate           DateTime      `json:"createdDate"`
	ApproveURL            string        `json:"approveURL"`
	CancelURL             string        `json:"cancelURL"`
	DeclineURL            string        `json:"declineURL"`
	Active                bool          `json:"active"`
	SendSMS               bool          `json:"sendSms"`
}

type InvoiceDetails struct {
	ID                    int64         `json:"id"`
	MerchantID            string        `json:"merchantId"`
	UUID                  string        `json:"uuid"`
	InvoiceUUID           string        `json:"invoiceUuid"`
	InvoiceCode           string        `json:"invoiceCode"`
	Status                InvoiceStatus `json:"invoiceStatus"`
	BaseURL               string        `json:"baseUrl"`
	Amount                float64       `json:"amount"`
	TotalAmount           float64       `json:"totalAmount"`
	PayriffAmount         float64       `json:"payriffAmount"`
	PayriffFee            float64       `json:"payriffFee"`
	PayriffFixedFeeAmount float64       `json:"payriffFixedFeeAmount"`
	Currency              Currency      `json:"currencyType"`
	Language              Language      `json:"languageType"`
	PaymentType           string        `json:"paymentType"`
	FullName              string        `json:"fullName"`
	Email                 string        `json:"email"`
	PhoneNumber           string        `json:"phoneNumber"`
	Description           string        `json:"description"`
	ExpireDate            DateTime      `json:"expireDate"`
	PaymentDay            DateTime      `json:"paymentDay"`
	ExpireDay             DateTime      `json:"expireDay"`
	CreatedDate           DateTime      `json:"createdDate"`
	ApproveURL            string        `json:"approveURL"`
	CancelURL             string        `json:"cancelURL"`
	DeclineURL            string        `json:"declineURL"`
	Active                bool          `json:"active"`
	InstallmentPeriod     int           `json:"installmentPeriod"`
	Source                string        `json:"source"`
	DirectPay             bool          `json:"directPay"`
	Metadata              string        `json:"metadata"`
}
