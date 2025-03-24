package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"sync"
)

type SignatureValidator struct {
	secret string
}

var (
	signatureValidator *SignatureValidator
	signatureOnce      sync.Once
)

func NewSignatureValidator() *SignatureValidator {
	signatureOnce.Do(func() {
		secret := os.Getenv("WEBHOOK_SECRET")
		if secret == "" {
			panic("WEBHOOK_SECRET environment variable not set")
		}

		signatureValidator = &SignatureValidator{
			secret: secret,
		}
	})

	return signatureValidator
}

func (sv *SignatureValidator) ValidateSignature(r *http.Request, body []byte) bool {
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		return false
	}

	signature = strings.TrimPrefix(signature, "sha256=")

	mac := hmac.New(sha256.New, []byte(sv.secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
