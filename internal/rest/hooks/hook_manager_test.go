package hooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/dfryer1193/gomad/api"
	"github.com/dfryer1193/gomad/internal/utils"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	TEST_SECRET       = "test-secret"
	TEST_SQL_PATH     = "test.sql"
	TEST_NON_SQL_PATH = "test.txt"
)

// Mock implementations
type mockMigrationManager struct {
	migrations []api.MigrationProto
	err        error
}

func (m *mockMigrationManager) ProcessMigrations(_ context.Context, migrations []api.MigrationProto) error {
	m.migrations = migrations
	return m.err
}

type mockGitFileFetcher struct {
	content string
	err     error
}

func (m *mockGitFileFetcher) FetchRawGitFile(_ utils.FileMetadata) (string, error) {
	if m.err != nil {
		return "", m.err
	}

	return m.content, nil
}

type mockSQLFileParser struct {
	migrations []api.MigrationProto
	err        error
}

func (m *mockSQLFileParser) ParseSQL(_ string) ([]api.MigrationProto, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.migrations, nil
}

func TestHandlePush(t *testing.T) {
	testCases := []struct {
		name       string
		event      *PushEvent
		secret     string
		wantStatus int
	}{
		{
			name:       "bad json payload",
			event:      nil,
			secret:     TEST_SECRET,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "bad signature",
			event: &PushEvent{
				Ref: "refs/heads/master",
			},
			secret:     "wrong-secret",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "non-master branch",
			event: &PushEvent{
				Ref: "refs/heads/develop",
				Commits: []Commit{
					{
						Added: []string{TEST_SQL_PATH},
					},
				},
			},
			secret:     TEST_SECRET,
			wantStatus: http.StatusNoContent,
		},
		{
			name: "no sql files",
			event: &PushEvent{
				Ref: "refs/heads/master",
				Commits: []Commit{
					{
						Added: []string{TEST_NON_SQL_PATH},
					},
				},
			},
			secret:     TEST_SECRET,
			wantStatus: http.StatusNoContent,
		},
		{
			name: "error processing sql files",
			event: &PushEvent{
				Ref: "refs/heads/master",
				Commits: []Commit{
					{
						Added: []string{TEST_SQL_PATH},
					},
				},
			},
			secret: TEST_SECRET,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := &HookManager{secret: tc.secret}
			w := httptest.NewRecorder()
			bodyBytes, _ := json.Marshal(tc.event)
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			h.HandlePush(w, req)
			if w.Code != tc.wantStatus {
				t.Errorf("HandlePush() = %v, want %v", w.Code, tc.wantStatus)
			}
		})
	}
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
			h := &HookManager{secret: tc.secret}
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(tc.body))
			req.Header.Set("X-Hub-Signature-256", tc.signature)

			got := h.validateSignature(req, tc.body)
			if got != tc.want {
				t.Errorf("validateSignature() = %v, want %v", got, tc.want)
			}
		})
	}
}
