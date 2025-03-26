package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dfryer1193/gomad/api"
	"github.com/dfryer1193/gomad/internal/rest/managers"
)

const (
	TEST_SECRET       = "test-secret"
	TEST_SQL_PATH     = "test.sql"
	TEST_NON_SQL_PATH = "test.txt"
)

type validSignatureValidator struct{}

func (v *validSignatureValidator) ValidateSignature(r *http.Request, repoName string, secret string, body []byte) bool {
	return true
}

type invalidSignatureValidator struct{}

func (v *invalidSignatureValidator) ValidateSignature(r *http.Request, repoName string, secret string, body []byte) bool {
	return false
}

type errorFileProcessor struct{}

func (f *errorFileProcessor) ProcessFile(_, path, _ string) ([]api.MigrationProto, error) {
	return nil, fmt.Errorf("error processing file %s", path)
}

type mockFileProcessor struct{}

func (f *mockFileProcessor) ProcessFile(_, _, _ string) ([]api.MigrationProto, error) {
	return []api.MigrationProto{}, nil
}

type errorMigrationManager struct{}

func (m *errorMigrationManager) ProcessMigrations(_ []api.MigrationProto) error {
	return fmt.Errorf("error processing migrations")
}

func (m *errorMigrationManager) GetMigrationsForNamespace(_ string) ([]*api.Migration, error) {
	return nil, fmt.Errorf("error getting migrations")
}

func (m *errorMigrationManager) GetMigrationById(_ uint64) (*api.Migration, error) {
	return nil, fmt.Errorf("error getting migration")
}

func (m *errorMigrationManager) Close() {}

type mockMigrationManager struct{}

func (m *mockMigrationManager) ProcessMigrations(_ []api.MigrationProto) error {
	return nil
}

func (m *mockMigrationManager) GetMigrationsForNamespace(_ string) ([]*api.Migration, error) {
	return nil, nil
}

func (m *mockMigrationManager) GetMigrationById(_ uint64) (*api.Migration, error) {
	return nil, nil
}

func (m *mockMigrationManager) Close() {}

type secretManagerMock struct{}

func (s *secretManagerMock) SaveSecret(_ string) (string, error) {
	return "test-secret", nil
}

func (s *secretManagerMock) GetSecret(_ string) (string, error) {
	return "test-secret", nil
}

func (s *secretManagerMock) Close() {}

func TestHandlePush(t *testing.T) {
	testCases := []struct {
		name               string
		signatureValidator SignatureValidator
		fileProcessor      MigrationFileProcessor
		migrationManager   managers.MigrationManager
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
			signatureValidator: &invalidSignatureValidator{},
			event: &PushEvent{
				Ref: "refs/heads/master",
			},
			secret:     "wrong-secret",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:               "non-master branch",
			signatureValidator: &validSignatureValidator{},
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
			signatureValidator: &validSignatureValidator{},
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
			signatureValidator: &validSignatureValidator{},
			fileProcessor:      &errorFileProcessor{},
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
			signatureValidator: &validSignatureValidator{},
			fileProcessor:      &errorFileProcessor{},
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
			signatureValidator: &validSignatureValidator{},
			fileProcessor:      &mockFileProcessor{},
			migrationManager:   &errorMigrationManager{},
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
			signatureValidator: &validSignatureValidator{},
			fileProcessor:      &mockFileProcessor{},
			migrationManager:   &mockMigrationManager{},
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
			h := &hookHandler{
				validator:              tc.signatureValidator,
				migrationFileProcessor: tc.fileProcessor,
				migrationMgr:           tc.migrationManager,
				secretMgr:              &secretManagerMock{},
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
