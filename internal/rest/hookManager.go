package rest

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
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
func (h *HookManager) HandlePush(c *gin.Context) {
	// Read the raw body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Err(err).Msg("Failed to read request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Validate webhook signature
	if !h.validateSignature(c.Request, body) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid webhook signature"})
		return
	}

	// Parse the webhook payload
	var event PushEvent
	if err := json.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse webhook payload"})
		return
	}

	// Only process pushes to master branch
	if event.Ref != "refs/heads/master" {
		c.Status(http.StatusOK)
		return
	}

	// Process changed files
	if err := h.processChanges(&event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to process changes: %v", err)})
		return
	}

	c.Status(http.StatusOK)
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

// processChanges handles the changes from the push event
func (h *HookManager) processChanges(event *PushEvent) error {
	changedFiles := make(map[string]struct{})

	// Collect all changed files
	for _, commit := range event.Commits {
		for _, file := range commit.Added {
			changedFiles[file] = struct{}{}
		}
		for _, file := range commit.Modified {
			changedFiles[file] = struct{}{}
		}
		// Optionally handle removed files if needed
		for _, file := range commit.Removed {
			changedFiles[file] = struct{}{}
		}
	}

	// Process each changed file
	for file := range changedFiles {
		if err := h.processFile(file); err != nil {
			return fmt.Errorf("failed to process file %s: %w", file, err)
		}
	}

	return nil
}

// processFile handles individual file changes
func (h *HookManager) processFile(filename string) error {
	// Add your file processing logic here
	// For example, if it's a SQL file, you might want to parse and execute it
	return nil
}
