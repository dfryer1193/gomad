package hooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
}

func TestProcessFile(t *testing.T) {
	testCases := []struct {
		name        string
		fetchErr    error
		migrations  []api.MigrationProto
		parseErr    error
		expectedErr error
	}{
		{
			name:        "successful file processing",
			fetchErr:    nil,
			migrations:  []api.MigrationProto{},
			parseErr:    nil,
			expectedErr: nil,
		},
		{
			name:        "fetch error",
			fetchErr:    errors.New("network error"),
			migrations:  nil,
			parseErr:    nil,
			expectedErr: errors.New("failed to fetch file testPath: network error"),
		},
		{
			name:        "parse error",
			fetchErr:    nil,
			migrations:  nil,
			parseErr:    errors.New("syntax error"),
			expectedErr: errors.New("syntax error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fetcher := &mockGitFileFetcher{
				content: "test content",
				err:     tc.fetchErr,
			}
			parser := &mockSQLFileParser{
				migrations: tc.migrations,
				err:        tc.parseErr,
			}

			h := &HookManager{
				gitFileFetcher: fetcher,
				sqlFileParser:  parser,
			}

			actualMigrations, actualErr := h.processFile("testRepo", "testPath", "testCommit")

			// Verify error handling
			if tc.expectedErr != nil {
				if actualErr == nil {
					t.Fatal("expected error but got nil")
				}
				if actualErr.Error() != tc.expectedErr.Error() {
					t.Errorf("expected error %v, got %v", tc.expectedErr, actualErr)
				}
				return
			}

			if actualErr != nil {
				t.Fatalf("unexpected error: %v", actualErr)
			}

			// Verify actualMigrations
			if tc.migrations != nil && actualMigrations == nil {
				t.Errorf("expected actualMigrations to be non-nil, got nil")
			}
		})
	}
}

//	testCases := []struct {
//		name           string
//		event          PushEvent
//		migrations     []api.MigrationProto
//		fetchErr       error
//		parseErr       error
//		wantErr        bool
//		wantMigrations []api.MigrationProto
//	}{
//		{
//			name: "successful processing",
//			event: PushEvent{
//				After: "abc123",
//				Repository: struct {
//					Name     string `json:"name"`
//					FullName string `json:"full_name"`
//				}{FullName: "test/repo"},
//				Commits: []struct {
//					ID        string   `json:"id"`
//					Message   string   `json:"message"`
//					Timestamp string   `json:"timestamp"`
//					Added     []string `json:"added"`
//					Modified  []string `json:"modified"`
//					Removed   []string `json:"removed"`
//					Author    struct {
//						Name  string `json:"name"`
//						Email string `json:"email"`
//					} `json:"author"`
//				}{{
//					Added: []string{"test.sql"},
//				}},
//			},
//			migrations:     []api.MigrationProto{{Name: "test"}},
//			wantMigrations: []api.MigrationProto{{Name: "test"}},
//		},
//		{
//			name: "no sql files",
//			event: PushEvent{
//				Commits: []struct {
//					ID        string   `json:"id"`
//					Message   string   `json:"message"`
//					Timestamp string   `json:"timestamp"`
//					Added     []string `json:"added"`
//					Modified  []string `json:"modified"`
//					Removed   []string `json:"removed"`
//					Author    struct {
//						Name  string `json:"name"`
//						Email string `json:"email"`
//					} `json:"author"`
//				}{{
//					Added: []string{"test.txt"},
//				}},
//			},
//			wantMigrations: []api.MigrationProto{},
//		},
//	}
//
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			fetcher := &mockGitFileFetcher{
//				content: "sql content",
//				err:     tc.fetchErr,
//			}
//			parser := &mockSQLFileParser{
//				migrations: tc.migrations,
//				err:        tc.parseErr,
//			}
//
//			h := &HookManager{
//				gitFileFetcher: fetcher,
//				sqlFileParser:  parser,
//			}
//
//			got, err := h.processSQLFiles(&tc.event)
//
//			if (err != nil) != tc.wantErr {
//				t.Errorf("processSQLFiles() error = %v, wantErr %v", err, tc.wantErr)
//				return
//			}
//
//			if !tc.wantErr && !reflect.DeepEqual(got, tc.wantMigrations) {
//				t.Errorf("processSQLFiles() = %v, want %v", got, tc.wantMigrations)
//			}
//		})
//	}
//}
//
//func TestHandlePush(t *testing.T) {
//	testCases := []struct {
//		name       string
//		event      PushEvent
//		secret     string
//		wantStatus int
//		migrations []api.MigrationProto
//		processErr error
//	}{
//		{
//			name: "successful push to master",
//			event: PushEvent{
//				Ref: "refs/heads/master",
//				Commits: []struct {
//					ID        string   `json:"id"`
//					Message   string   `json:"message"`
//					Timestamp string   `json:"timestamp"`
//					Added     []string `json:"added"`
//					Modified  []string `json:"modified"`
//					Removed   []string `json:"removed"`
//					Author    struct {
//						Name  string `json:"name"`
//						Email string `json:"email"`
//					} `json:"author"`
//				}{{Added: []string{"test.sql"}}},
//			},
//			secret:     "test-secret",
//			wantStatus: http.StatusNoContent,
//			migrations: []api.MigrationProto{{Name: "test"}},
//		},
//		{
//			name: "non-master branch",
//			event: PushEvent{
//				Ref: "refs/heads/develop",
//			},
//			secret:     "test-secret",
//			wantStatus: http.StatusNoContent,
//		},
//		{
//			name: "invalid signature",
//			event: PushEvent{
//				Ref: "refs/heads/master",
//			},
//			secret:     "wrong-secret",
//			wantStatus: http.StatusUnauthorized,
//		},
//	}
//
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			migrationMgr := &mockMigrationManager{err: tc.processErr}
//			fetcher := &mockGitFileFetcher{content: "sql content"}
//			parser := &mockSQLFileParser{migrations: tc.migrations}
//
//			h := &HookManager{
//				secret:         "test-secret",
//				migrationMgr:   migrationMgr,
//				gitFileFetcher: fetcher,
//				sqlFileParser:  parser,
//			}
//
//			body, err := json.Marshal(tc.event)
//			if err != nil {
//				t.Fatalf("failed to marshal event: %v", err)
//			}
//
//			req := httptest.NewRequest(http.MethodPost, "/hooks/v1/push", bytes.NewReader(body))
//			req.Header.Set("Content-Type", "application/json")
//
//			// Calculate signature
//			mac := hmac.New(sha256.New, []byte(tc.secret))
//			mac.Write(body)
//			signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
//			req.Header.Set("X-Hub-Signature-256", signature)
//
//			w := httptest.NewRecorder()
//			h.HandlePush(w, req)
//
//			if w.Code != tc.wantStatus {
//				t.Errorf("HandlePush() status = %v, want %v", w.Code, tc.wantStatus)
//			}
//		})
//	}
//}

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
