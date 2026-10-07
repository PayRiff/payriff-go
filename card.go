package payriff

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
)

const productionCardEncryptionKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxRq5a+44T6Dac60XmVRQ/7cpPyFsBnamXbJlRVJk8CnES5Re5tVMohyD0hZr" +
	"3zcQj+bxodYB4zZpQTPlrXvFBC3zz+rXnGlevxBQ6W2d3QC9q8vWH8p3ZOwTO3qVvDSHH9o+hMMRNbJ7kueq/KZlX/F+bjZ23CZw7iXE" +
	"GQT3HYVYnnHsvpaguYDteWBag2sPPLLsVjeB3zhTfQ7OsWp5XTkDuRwLugHPvs6RHLcwGCnodukWyvwUaEUQR/kMGC+RbMsAIVkcLMP5" +
	"csfR3Xo7Gi98+i44iLN00f7gE8QvEmvv8xDspyTAjDEL1a5gK7TijJ3yLG/Bwa1rr1uskYy7lwIDAQAB"

const (
	aesKeyBytes = 32
	ivBytes     = 12
)

type Card struct {
	PAN         string
	CardHolder  string
	ExpiryMonth string
	ExpiryYear  string
	CVV         string
}

func (c Card) String() string {
	return "Card{" + maskPAN(c.PAN) + "}"
}

func (c Card) GoString() string {
	return c.String()
}

func (c Card) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(c.String())), nil
}

func (c Card) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

func maskPAN(pan string) string {
	digits := separators.ReplaceAllString(pan, "")
	if len(digits) < 10 {
		return "****"
	}
	return digits[:6] + "******" + digits[len(digits)-4:]
}

var (
	separators = regexp.MustCompile(`[\s-]`)
	allDigits  = regexp.MustCompile(`^[0-9]+$`)
)

type cardPayload struct {
	PAN         string `json:"pan"`
	CardHolder  string `json:"cardHolder"`
	ExpiryYear  string `json:"expiryYear"`
	ExpiryMonth string `json:"expiryMonth"`
	CVV         string `json:"cvv"`
}

func digits(value, field string) (string, error) {
	v := separators.ReplaceAllString(value, "")
	if !allDigits.MatchString(v) {
		return "", invalid("%s must contain digits only", field)
	}
	return v, nil
}

func normalizeCard(c Card) (cardPayload, error) {
	pan, err := digits(c.PAN, "pan")
	if err != nil {
		return cardPayload{}, err
	}
	if len(pan) < 12 || len(pan) > 19 {
		return cardPayload{}, invalid("pan must contain 12-19 digits")
	}
	monthDigits, err := digits(c.ExpiryMonth, "expiryMonth")
	if err != nil {
		return cardPayload{}, err
	}
	month, _ := strconv.Atoi(monthDigits)
	if month < 1 || month > 12 {
		return cardPayload{}, invalid("expiryMonth must be 1-12")
	}
	year, err := digits(c.ExpiryYear, "expiryYear")
	if err != nil {
		return cardPayload{}, err
	}
	switch len(year) {
	case 2:
		year = "20" + year
	case 4:
	default:
		return cardPayload{}, invalid("expiryYear must have 2 or 4 digits")
	}
	cvv, err := digits(c.CVV, "cvv")
	if err != nil {
		return cardPayload{}, err
	}
	if len(cvv) < 3 || len(cvv) > 4 {
		return cardPayload{}, invalid("cvv must contain 3-4 digits")
	}
	return cardPayload{
		PAN:         pan,
		CardHolder:  strings.TrimSpace(c.CardHolder),
		ExpiryYear:  year,
		ExpiryMonth: fmt.Sprintf("%02d", month),
		CVV:         cvv,
	}, nil
}

func parsePublicKey(base64OrPEM string) (*rsa.PublicKey, error) {
	body := strings.NewReplacer("-----BEGIN PUBLIC KEY-----", "", "-----END PUBLIC KEY-----", "").Replace(base64OrPEM)
	body = strings.Join(strings.Fields(body), "")
	der, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return nil, invalid("invalid RSA public key")
	}
	key, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, invalid("invalid RSA public key")
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, invalid("invalid RSA public key")
	}
	return rsaKey, nil
}

type encryptedCard struct {
	message   string
	secretKey string
}

func encryptCard(key *rsa.PublicKey, card cardPayload) (encryptedCard, error) {
	plaintext, err := marshal(card)
	if err != nil {
		return encryptedCard{}, err
	}
	keyAndIV := make([]byte, aesKeyBytes+ivBytes)
	defer func() {
		clear(keyAndIV)
		clear(plaintext)
	}()
	if _, err := rand.Read(keyAndIV); err != nil {
		return encryptedCard{}, fmt.Errorf("payriff: card encryption failed: %w", err)
	}
	block, err := aes.NewCipher(keyAndIV[:aesKeyBytes])
	if err != nil {
		return encryptedCard{}, fmt.Errorf("payriff: card encryption failed: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return encryptedCard{}, fmt.Errorf("payriff: card encryption failed: %w", err)
	}
	ciphertext := gcm.Seal(nil, keyAndIV[aesKeyBytes:], plaintext, nil)
	secret, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, key, keyAndIV, nil)
	if err != nil {
		return encryptedCard{}, fmt.Errorf("payriff: card encryption failed: %w", err)
	}
	return encryptedCard{
		message:   base64.StdEncoding.EncodeToString(ciphertext),
		secretKey: base64.StdEncoding.EncodeToString(secret),
	}, nil
}
