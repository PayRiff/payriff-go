package payriff

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

var (
	testKeyOnce sync.Once
	testKey     *rsa.PrivateKey
)

func privateKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	testKeyOnce.Do(func() {
		k, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}
		testKey = k
	})
	return testKey
}

func publicKeyBase64(t *testing.T) string {
	der, err := x509.MarshalPKIXPublicKey(&privateKey(t).PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(der)
}

func decrypt(t *testing.T, secretKey, encryptedMessage string) []byte {
	t.Helper()
	secret, _ := base64.StdEncoding.DecodeString(secretKey)
	keyAndIV, err := rsa.DecryptOAEP(sha256.New(), nil, privateKey(t), secret, nil)
	if err != nil || len(keyAndIV) != 44 {
		t.Fatalf("RSA decrypt: %v (len %d)", err, len(keyAndIV))
	}
	data, _ := base64.StdEncoding.DecodeString(encryptedMessage)
	block, _ := aes.NewCipher(keyAndIV[:32])
	gcm, _ := cipher.NewGCM(block)
	plain, err := gcm.Open(nil, keyAndIV[32:], data, nil)
	if err != nil {
		t.Fatalf("AES-GCM decrypt: %v", err)
	}
	return plain
}

var testCard = Card{PAN: "4169 7413 3015 1979", CardHolder: " JOHN DOE ", ExpiryMonth: "1", ExpiryYear: "27", CVV: "123"}

func TestEncryptedCardDecryptsWithPayriffScheme(t *testing.T) {
	card, err := normalizeCard(testCard)
	if err != nil {
		t.Fatal(err)
	}

	enc, err := encryptCard(&privateKey(t).PublicKey, card)

	if err != nil {
		t.Fatal(err)
	}
	want := `{"pan":"4169741330151979","cardHolder":"JOHN DOE","expiryYear":"2027","expiryMonth":"01","cvv":"123"}`
	if got := string(decrypt(t, enc.secretKey, enc.message)); got != want {
		t.Fatalf("got %s", got)
	}
}

func TestEveryCallUsesFreshKeyMaterial(t *testing.T) {
	card, _ := normalizeCard(testCard)

	a, _ := encryptCard(&privateKey(t).PublicKey, card)
	b, _ := encryptCard(&privateKey(t).PublicKey, card)

	if a.message == b.message || a.secretKey == b.secretKey {
		t.Fatal("key material reused")
	}
}

func TestParsesBase64AndPEMKeys(t *testing.T) {
	var pem strings.Builder
	pem.WriteString("-----BEGIN PUBLIC KEY-----\n")
	for i := 0; i < len(productionCardEncryptionKey); i += 64 {
		pem.WriteString(productionCardEncryptionKey[i:min(i+64, len(productionCardEncryptionKey))] + "\n")
	}
	pem.WriteString("-----END PUBLIC KEY-----\n")

	for _, key := range []string{productionCardEncryptionKey, pem.String()} {
		k, err := parsePublicKey(key)
		if err != nil || k.N.BitLen() != 2048 {
			t.Fatalf("got %v", err)
		}
	}
}

func TestRejectsNonRSAKey(t *testing.T) {
	ec, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	der, _ := x509.MarshalPKIXPublicKey(&ec.PublicKey)

	_, err := parsePublicKey(base64.StdEncoding.EncodeToString(der))

	mustInvalid(t, err, "invalid RSA public key")
}

func TestNormalizesCardInput(t *testing.T) {
	got, err := normalizeCard(Card{PAN: "4169-7413-3015-1979", CardHolder: "Rəşad", ExpiryMonth: "11", ExpiryYear: "2027", CVV: "1234"})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := marshal(got)
	if string(b) != `{"pan":"4169741330151979","cardHolder":"Rəşad","expiryYear":"2027","expiryMonth":"11","cvv":"1234"}` {
		t.Fatalf("got %s", b)
	}
}

func TestRejectsInvalidCardInput(t *testing.T) {
	cases := []struct {
		mutate  func(*Card)
		message string
	}{
		{func(c *Card) { c.PAN = "4169" }, "pan must contain 12-19 digits"},
		{func(c *Card) { c.PAN = "4169741330151979000000" }, "pan must contain 12-19 digits"},
		{func(c *Card) { c.PAN = "4169a41330151979" }, "pan must contain digits only"},
		{func(c *Card) { c.ExpiryMonth = "13" }, "expiryMonth must be 1-12"},
		{func(c *Card) { c.ExpiryMonth = "0" }, "expiryMonth must be 1-12"},
		{func(c *Card) { c.ExpiryYear = "202" }, "expiryYear must have 2 or 4 digits"},
		{func(c *Card) { c.CVV = "12" }, "cvv must contain 3-4 digits"},
		{func(c *Card) { c.CVV = "" }, "cvv must contain digits only"},
	}
	for _, tc := range cases {
		t.Run(tc.message, func(t *testing.T) {
			c := testCard
			tc.mutate(&c)
			_, err := normalizeCard(c)
			mustInvalid(t, err, tc.message)
		})
	}
}

func TestCardNeverPrintsSensitiveData(t *testing.T) {
	var out bytes.Buffer
	fmt.Fprintf(&out, "%v %+v %#v %s", testCard, testCard, testCard, testCard)
	fmt.Fprintf(&out, "%v %+v", &testCard, struct{ C Card }{testCard})
	j, _ := json.Marshal(map[string]any{"card": testCard})
	out.Write(j)
	slog.New(slog.NewJSONHandler(&out, nil)).Info("pay", "card", testCard)

	s := out.String()
	if !strings.Contains(s, "416974******1979") || strings.Contains(s, "4169741330151979") || strings.Contains(s, "7413 3015") || strings.Contains(s, "123") {
		t.Fatalf("sensitive data leaked: %s", s)
	}
}
