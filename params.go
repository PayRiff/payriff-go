package payriff

import "time"

type CreateOrderParams struct {
	Amount          float64
	Currency        Currency
	Language        Language
	Operation       Operation
	Description     string
	CallbackURL     string
	RedirectURL     string
	CardSave        *bool
	ThreeDS         *bool
	AutoPaymentType AutoPaymentType
	Installment     *Installment
	FullName        string
	PhoneNumber     string
	Metadata        map[string]string
	Fields          map[string]string
	RequestRRN      string
}

type RefundParams struct {
	OrderID      string
	Amount       float64
	RefundReason string
	CallbackURL  string
}

type CompleteParams struct {
	OrderID     string
	Amount      float64
	CallbackURL string
}

type DirectPayParams struct {
	Amount       float64
	Description  string
	Card         Card
	Operation    Operation
	Currency     Currency
	CallbackURL  string
	ThreeDS      *bool
	CustomFields map[string]string
	CardSave     bool
	RequestRRN   string
}

type AutoPayParams struct {
	CardUUID        string
	Amount          float64
	Description     string
	Operation       Operation
	Currency        Currency
	CallbackURL     string
	ThreeDS         *bool
	OneClickPayment *bool
	RequestRRN      string
}

type CardSaveParams struct {
	CustomerRef    string
	CallbackURL    string
	Description    string
	Language       Language
	Metadata       map[string]string
	IdempotencyKey string
}

type PayoutParams struct {
	TransferAmount float64
	Description    string
	FullName       string
	FinCode        string
	CardPAN        string
	BankName       string
	CardType       string
	RequestRRN     string
	CustomerCode   string
	VOEN           string
	BirthDate      string
	CallbackURL    string
	IdempotencyKey string
}

type InvoiceCreateParams struct {
	Amount                 float64
	AmountDynamic          bool
	Currency               Currency
	Language               Language
	FullName               string
	Email                  string
	PhoneNumber            string
	Description            string
	CustomMessage          string
	ExpireDate             time.Time
	ApproveURL             string
	CancelURL              string
	DeclineURL             string
	RedirectURL            string
	InstallmentProductType InstallmentProductType
	InstallmentPeriod      int
	DirectPay              *bool
	SendSMS                *bool
	SendWhatsapp           *bool
	SendEmail              *bool
	Metadata               map[string]string
	ExternalTransactionID  string
}

type TransactionFilter struct {
	OrderID     string
	RRN         string
	Status      PaymentStatus
	Amount      string
	Description string
	Name        string
	FullName    string
	CardNumber  string
	BookingID   string
	InvoiceCode string
	From        time.Time
	To          time.Time
	Page        int
	Size        int
}

type PayoutFilter struct {
	RRN         string
	Status      TransferState
	Amount      string
	Description string
	FullName    string
	FinCode     string
	BankSource  string
	From        time.Time
	To          time.Time
	Page        int
	Size        int
}
