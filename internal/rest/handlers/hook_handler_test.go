package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/dfryer1193/gomad/api"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	TEST_SECRET       = "test-secret"
	TEST_SQL_PATH     = "test.sql"
	TEST_NON_SQL_PATH = "test.txt"
)

type ValidSignatureValidator struct{}

func (v *ValidSignatureValidator) ValidateSignature(r *http.Request, body []byte) bool {
	return true
}

type InvalidSignatureValidator struct{}

func (v *InvalidSignatureValidator) ValidateSignature(r *http.Request, body []byte) bool {
	return false
}

type ErrorFileProcessor struct{}

func (f *ErrorFileProcessor) ProcessFile(_, path, _ string) ([]api.MigrationProto, error) {
	return nil, fmt.Errorf("error processing file %s", path)
}

type MockFileProcessor struct{}

func (f *MockFileProcessor) ProcessFile(_, _, _ string) ([]api.MigrationProto, error) {
	return []api.MigrationProto{}, nil
}

type ErrorMigrationManager struct{}

func (m *ErrorMigrationManager) ProcessMigrations(_ context.Context, _ []api.MigrationProto) error {
	return fmt.Errorf("error processing migrations")
}

type MockMigrationManager struct{}

func (m *MockMigrationManager) ProcessMigrations(_ context.Context, _ []api.MigrationProto) error {
	return nil
}

func TestHandlePush(t *testing.T) {
	testCases := []struct {
		name               string
		signatureValidator SignatureValidator
		fileProcessor      MigrationFileProcessor
		migrationManager   MigrationManager
		event              *PushEvent
		mangleBody         bool
		secret             string
		wantStatus         int
	}{
		{
			name:       "bad json payload",
			event:      nil,
			mangleBody: true,
			secret:     TEST_SECRET,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:               "bad signature",
			signatureValidator: &InvalidSignatureValidator{},
			event: &PushEvent{
				Ref: "refs/heads/master",
			},
			secret:     "wrong-secret",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:               "non-master branch",
			signatureValidator: &ValidSignatureValidator{},
			event: &PushEvent{
				Ref: "refs/heads/develop",
				Commits: []Commit{
					{
						Added: []string{TEST_SQL_PATH},
					},
				},
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:               "no sql files",
			signatureValidator: &ValidSignatureValidator{},
			event: &PushEvent{
				Ref: "refs/heads/master",
				Commits: []Commit{
					{
						Added: []string{TEST_NON_SQL_PATH},
					},
				},
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:               "error processing sql files",
			signatureValidator: &ValidSignatureValidator{},
			fileProcessor:      &ErrorFileProcessor{},
			event: &PushEvent{
				Ref: "refs/heads/master",
				Commits: []Commit{
					{
						Added: []string{TEST_SQL_PATH},
					},
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:               "error processing migrations files",
			signatureValidator: &ValidSignatureValidator{},
			fileProcessor:      &ErrorFileProcessor{},
			event: &PushEvent{
				Ref: "refs/heads/master",
				Commits: []Commit{
					{
						Added: []string{TEST_SQL_PATH},
					},
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:               "error processing migration prototypes",
			signatureValidator: &ValidSignatureValidator{},
			fileProcessor:      &MockFileProcessor{},
			migrationManager:   &ErrorMigrationManager{},
			event: &PushEvent{
				Ref: "refs/heads/master",
				Commits: []Commit{
					{
						Added: []string{TEST_SQL_PATH},
					},
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:               "successful processing",
			signatureValidator: &ValidSignatureValidator{},
			fileProcessor:      &MockFileProcessor{},
			migrationManager:   &MockMigrationManager{},
			event: &PushEvent{
				Ref: "refs/heads/master",
				Commits: []Commit{
					{
						Added: []string{TEST_SQL_PATH},
					},
				},
			},
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := &HookHandler{
				validator:              tc.signatureValidator,
				migrationFileProcessor: tc.fileProcessor,
				migrationMgr:           tc.migrationManager,
			}
			w := httptest.NewRecorder()
			bodyBytes := []byte("invalid json")
			if !tc.mangleBody {
				bodyBytes, _ = json.Marshal(tc.event)
			}
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			err := h.HandlePush(w, req)
			if tc.wantStatus > 204 {
				if err == nil {
					t.Errorf("Expected error, got none")
				}

				// TODO: Validate error code
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}
