package utils

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockSecretRepository struct {
}

func (m *mockSecretRepository) GetSecret(repoName string) (string, error) {
	return "test-secret", nil
}

func (m *mockSecretRepository) InsertSecret(repoName string, secret string) (string, error) {
	return "", nil
}

func (m *mockSecretRepository) Close() {
}

func TestValidateSignature(t *testing.T) {
	testCases := []struct {
		name      string
		body      []byte
		signature string
		secret    string
		want      bool
	}{
		{
			name:   "valid signature",
			body:   []byte("test body"),
			secret: "test-secret",
			signature: func() string {
				mac := hmac.New(sha256.New, []byte("test-secret"))
				mac.Write([]byte("test body"))
				return "sha256=" + hex.EncodeToString(mac.Sum(nil))
			}(),
			want: true,
		},
		{
			name:      "invalid signature",
			body:      []byte("test body"),
			secret:    "test-secret",
			signature: "sha256=invalid",
			want:      false,
		},
		{
			name:      "empty signature",
			body:      []byte("test body"),
			secret:    "test-secret",
			signature: "",
			want:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sv := &signatureValidator{
				secretsRepo: &mockSecretRepository{},
			}
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(tc.body))
			req.Header.Set("X-Hub-Signature-256", tc.signature)

			got := sv.ValidateSignature(req, "test-repo", tc.secret, tc.body)
			if got != tc.want {
				t.Errorf("validateSignature() = %v, want %v", got, tc.want)
			}
		})
	}
}
