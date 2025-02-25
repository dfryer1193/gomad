package hooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/dfryer1193/gomad/api"
	"github.com/dfryer1193/gomad/internal/rest/migrations"
	"github.com/dfryer1193/gomad/internal/utils"
	"github.com/dfryer1193/mjolnir/middleware"
	mjolnirUtils "github.com/dfryer1193/mjolnir/utils"
	"net/http"
	"os"
	"strings"
	"sync"
)

type MigrationManager interface {
	ProcessMigrations(ctx context.Context, migrations []api.MigrationProto) error
}

type GitFileFetcher interface {
	FetchRawGitFile(metadata utils.FileMetadata) (string, error)
}

type SQLFileParser interface {
	ParseSQL(content string) ([]api.MigrationProto, error)
}

type HookManager struct {
	secret         string
	migrationMgr   MigrationManager
	gitFileFetcher GitFileFetcher
	sqlFileParser  SQLFileParser
}

var (
	once sync.Once
	mgr  *HookManager
)

func GetHookManager() *HookManager {
	secret := os.Getenv("WEBHOOK_SECRET")
	if secret == "" {
		panic("WEBHOOK_SECRET environment variable is required")
	}

	once.Do(func() {
		mgr = &HookManager{
			secret:         secret,
			migrationMgr:   migrations.GetMigrationsManager(),
			gitFileFetcher: utils.GetGitFileFetcher(),
			sqlFileParser:  utils.NewMigrationFileParser(),
		}
	})

	return mgr
}

type PushEvent struct {
	Ref        string `json:"ref"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Repository struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
	} `json:"repository"`
	Commits []struct {
		ID        string   `json:"id"`
		Message   string   `json:"message"`
		Timestamp string   `json:"timestamp"`
		Added     []string `json:"added"`
		Modified  []string `json:"modified"`
		Removed   []string `json:"removed"`
		Author    struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"commits"`
}

// HandlePush handles Git push webhooks
func (h *HookManager) HandlePush(w http.ResponseWriter, r *http.Request) {
	// Read the raw body
	event := &PushEvent{}
	bodyBytes, err := mjolnirUtils.DecodeJSON(r, event)
	if err != nil {
		middleware.SetBadRequestError(r, fmt.Errorf("failed to decode JSON: %w", err))
		return
	}

	// Validate webhook signature
	if !h.validateSignature(r, bodyBytes) {
		middleware.SetUnauthorizedError(r, fmt.Errorf("Invalid webhook signature"))
		return
	}

	// Only process pushes to master branch
	if event.Ref != "refs/heads/master" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	migrationPrototypes, err := h.processSQLFiles(event)
	if err != nil {
		middleware.SetInternalError(r, fmt.Errorf("failed to process SQL changes: %w", err))
		return
	}

	err = h.migrationMgr.ProcessMigrations(r.Context(), migrationPrototypes)
	if err != nil {
		middleware.SetInternalError(r, fmt.Errorf("failed to process migrations: %w", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// validateSignature validates the webhook signature
func (h *HookManager) validateSignature(r *http.Request, body []byte) bool {
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		return false
	}

	signature = strings.TrimPrefix(signature, "sha256=")

	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// processSQLFiles handles the changes from the push event
func (h *HookManager) processSQLFiles(event *PushEvent) ([]api.MigrationProto, error) {
	migrationPrototypes := make([]api.MigrationProto, 0)

	// Collect all changed files
	for _, commit := range event.Commits {
		for _, file := range commit.Added {
			if strings.HasSuffix(file, ".sql") {
				protos, err := h.processFile(event.Repository.FullName, file, event.After)
				if err != nil {
					return nil, fmt.Errorf("failed to process file %s: %w", file, err)
				}
				migrationPrototypes = append(migrationPrototypes, protos...)
			}
		}
		for _, file := range commit.Modified {
			if strings.HasSuffix(file, ".sql") {
				protos, err := h.processFile(event.Repository.FullName, file, event.After)
				if err != nil {
					return nil, fmt.Errorf("failed to process file %s: %w", file, err)
				}
				migrationPrototypes = append(migrationPrototypes, protos...)
			}
		}
	}

	return migrationPrototypes, nil
}

// processFile handles individual file changes
func (h *HookManager) processFile(repoName string, path string, commit string) ([]api.MigrationProto, error) {
	metadata := &utils.FileMetadata{
		RepoName: repoName,
		Path:     path,
		Commit:   commit,
	}
	content, err := h.gitFileFetcher.FetchRawGitFile(*metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file %s: %w", metadata.Path, err)
	}

	foundMigrations, err := h.sqlFileParser.ParseSQL(content)
	if err != nil {
		return nil, err
	}

	return foundMigrations, nil
}
