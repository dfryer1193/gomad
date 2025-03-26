package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"

	"github.com/dfryer1193/gomad/internal/data/repository"
	"github.com/dfryer1193/gomad/internal/data/repository/postgres"
)

type SignatureValidator interface {
	ValidateSignature(r *http.Request, repoName string, secret string, body []byte) bool
}

type signatureValidator struct {
	secretsRepo repository.SecretRepository
}

var (
	validator     *signatureValidator
	signatureOnce sync.Once
)

func NewSignatureValidator() *signatureValidator {
	signatureOnce.Do(func() {
		validator = &signatureValidator{
			secretsRepo: postgres.GetSecretsRepository(),
		}
	})

	return validator
}

func (sv *signatureValidator) ValidateSignature(r *http.Request, repoName string, secret string, body []byte) bool {
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		return false
	}

	signature = strings.TrimPrefix(signature, "sha256=")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
