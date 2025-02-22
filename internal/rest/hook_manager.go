package rest

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/dfryer1193/gomad/internal/utils"
	"github.com/dfryer1193/mjolnir/middleware"
	mjolnirUtils "github.com/dfryer1193/mjolnir/utils"
	"net/http"
	"os"
	"strings"
)

type HookManager struct {
	secret string
}

func NewHookManager() *HookManager {
	// Get webhook secret from environment
	secret := os.Getenv("WEBHOOK_SECRET")
	if secret == "" {
		panic("WEBHOOK_SECRET environment variable is required")
	}

	return &HookManager{
		secret: secret,
	}
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

	// Process changed files
	if err := h.processSQLChanges(event); err != nil {
		middleware.SetInternalError(r, fmt.Errorf("failed to process SQL changes: %w", err))
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

// processSQLChanges handles the changes from the push event
func (h *HookManager) processSQLChanges(event *PushEvent) error {
	changedFiles := make(map[string]struct{})

	// Collect all changed files
	for _, commit := range event.Commits {
		for _, file := range commit.Added {
			if strings.HasSuffix(file, ".sql") {
				changedFiles[file] = struct{}{}
			}
		}
		for _, file := range commit.Modified {
			if strings.HasSuffix(file, ".sql") {
				changedFiles[file] = struct{}{}
			}
		}
	}

	// Process each changed file
	for file := range changedFiles {
		metadata := &utils.FileMetadata{
			RepoName: event.Repository.FullName,
			Path:     file,
			Commit:   event.After,
		}
		if err := h.processFile(metadata); err != nil {
			return fmt.Errorf("failed to process file %s: %w", file, err)
		}
	}

	return nil
}

// processFile handles individual file changes
func (h *HookManager) processFile(metadata *utils.FileMetadata) error {
	content, err := utils.GetGitFileFetcher().FetchRawGitFile(*metadata)
	if err != nil {
		return err
	}

	migrations, err := utils.ParseSQL(content)
	if err != nil {
		return err
	}
	if len(migrations) == 0 {
	}

	// Parse and add DDL to database.
	return nil
}
