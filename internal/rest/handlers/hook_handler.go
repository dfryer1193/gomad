package handlers

import (
	"context"
	"fmt"
	"github.com/dfryer1193/gomad/api"
	"github.com/dfryer1193/gomad/internal/rest/migrations"
	"github.com/dfryer1193/gomad/internal/utils"
	mjolnirUtils "github.com/dfryer1193/mjolnir/utils"
	"net/http"
	"os"
	"strings"
	"sync"
)

type MigrationProcessor interface {
	ProcessMigrations(migrations []api.MigrationProto) error
}

type MigrationFileProcessor interface {
	ProcessFile(repoName string, path string, commit string) ([]api.MigrationProto, error)
}

type SignatureValidator interface {
	ValidateSignature(r *http.Request, body []byte) bool
}

type HookHandler struct {
	validator              SignatureValidator
	migrationMgr           MigrationProcessor
	migrationFileProcessor MigrationFileProcessor
}

var (
	hookOnce sync.Once
	mgr      *HookHandler
)

func GetHookHandler() *HookHandler {
	secret := os.Getenv("WEBHOOK_SECRET")
	if secret == "" {
		panic("WEBHOOK_SECRET environment variable is required")
	}

	hookOnce.Do(func() {
		mgr = &HookHandler{
			validator:              utils.NewSignatureValidator(secret),
			migrationMgr:           managers.GetMigrationsManager(),
			migrationFileProcessor: utils.GetMigrationFileProcessor(),
		}
	})

	return mgr
}

type Repository struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

type Commit struct {
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
}

type PushEvent struct {
	Ref        string     `json:"ref"`
	Before     string     `json:"before"`
	After      string     `json:"after"`
	Repository Repository `json:"repository"`
	Commits    []Commit   `json:"commits"`
}

// HandlePush handles Git push webhooks by looking for added or modified sql files and treating them as migrations files
func (h *HookManager) HandlePush(w http.ResponseWriter, r *http.Request) *mjolnirUtils.ApiError {
	// Read the raw body
	event := &PushEvent{}
	bodyBytes, err := mjolnirUtils.DecodeJSON(r, event)
	if err != nil {
		return mjolnirUtils.BadRequestErr(fmt.Errorf("failed to decode JSON: %w", err))
	}

	// Validate webhook signature
	if !h.validator.ValidateSignature(r, bodyBytes) {
		return mjolnirUtils.UnauthorizedErr(fmt.Errorf("Invalid webhook signature"))
	}

	// Only process pushes to master branch
	if event.Ref != "refs/heads/master" {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}

	sqlFiles := h.getSQLFiles(event)
	if len(sqlFiles) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}

	migrationPrototypes := make([]api.MigrationProto, 0)
	for _, file := range sqlFiles {
		proto, err := h.migrationFileProcessor.ProcessFile(event.Repository.FullName, file, event.After)
		if err != nil {
			return mjolnirUtils.InternalServerErr(fmt.Errorf("failed to process SQL file: %s", file))
		}
		migrationPrototypes = append(migrationPrototypes, proto...)
	}

	err = h.migrationMgr.ProcessMigrations(migrationPrototypes)
	if err != nil {
		return mjolnirUtils.InternalServerErr(fmt.Errorf("failed to process SQL changes: %w", err))
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *HookHandler) getSQLFiles(event *PushEvent) []string {
	sqlFiles := make([]string, 0)

	// Collect all changed files
	for _, commit := range event.Commits {
		for _, file := range commit.Added {
			if strings.HasSuffix(file, ".sql") {
				sqlFiles = append(sqlFiles, file)
			}
		}
		for _, file := range commit.Modified {
			if strings.HasSuffix(file, ".sql") {
				sqlFiles = append(sqlFiles, file)
			}
		}
	}

	return sqlFiles
}
