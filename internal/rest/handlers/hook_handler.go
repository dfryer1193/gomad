package handlers

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/dfryer1193/gomad/api"
	"github.com/dfryer1193/gomad/internal/rest/managers"
	"github.com/dfryer1193/gomad/internal/utils"
	mjolnirUtils "github.com/dfryer1193/mjolnir/utils"
)

type MigrationFileProcessor interface {
	ProcessFile(repoName string, path string, commit string) ([]api.MigrationProto, error)
}

type SignatureValidator interface {
	ValidateSignature(r *http.Request, repoName string, secret string, body []byte) bool
}

type HookHandler interface {
	HandlePush(w http.ResponseWriter, r *http.Request) *mjolnirUtils.ApiError
	HandleCreateSecret(w http.ResponseWriter, r *http.Request) *mjolnirUtils.ApiError
	Close()
}

type hookHandler struct {
	validator              SignatureValidator
	migrationMgr           managers.MigrationManager
	migrationFileProcessor MigrationFileProcessor
	secretMgr              managers.SecretManager
}

var (
	hookOnce sync.Once
	mgr      *hookHandler
)

func GetHookHandler() *hookHandler {
	hookOnce.Do(func() {
		mgr = &hookHandler{
			validator:              utils.NewSignatureValidator(),
			migrationMgr:           managers.GetMigrationsManager(),
			migrationFileProcessor: utils.GetMigrationFileProcessor(),
			secretMgr:              managers.GetSecretManager(),
		}
	})

	return mgr
}

func (h *hookHandler) HandleCreateSecret(w http.ResponseWriter, r *http.Request) *mjolnirUtils.ApiError {
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}

	// Check if IP is from my local network)
	// TODO: Create a way to specify this from a config
	if !isIPInRange(ip, "192.168.0.0/16") || !isIPInRange(ip, "10.110.173.0/24") {
		return mjolnirUtils.UnauthorizedErr(fmt.Errorf("unauthorized IP address: %s", ip))
	}

	var repoName struct {
		Name string `json:"repoName"`
	}
	_, err := mjolnirUtils.DecodeJSON(r, repoName)
	if err != nil {
		return mjolnirUtils.BadRequestErr(fmt.Errorf("failed to decode JSON: %w", err))
	}

	secret, err := h.secretMgr.SaveSecret(repoName.Name)
	if err != nil {
		return mjolnirUtils.InternalServerErr(fmt.Errorf("failed to save secret: %w", err))
	}

	mjolnirUtils.RespondJSON(w, r, http.StatusOK, map[string]string{"secret": secret})
	return nil
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
func (h *hookHandler) HandlePush(w http.ResponseWriter, r *http.Request) *mjolnirUtils.ApiError {
	// Read the raw body
	event := &PushEvent{}
	bodyBytes, err := mjolnirUtils.DecodeJSON(r, event)
	if err != nil {
		return mjolnirUtils.BadRequestErr(fmt.Errorf("failed to decode JSON: %w", err))
	}

	secret, err := h.secretMgr.GetSecret(event.Repository.FullName)
	if err != nil {
		return mjolnirUtils.InternalServerErr(fmt.Errorf("failed to get secret for repo %s: %w", event.Repository.FullName, err))
	}

	// Validate webhook signature
	if !h.validator.ValidateSignature(r, event.Repository.FullName, secret, bodyBytes) {
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

func (h *hookHandler) Close() {
	h.migrationMgr.Close()
	h.secretMgr.Close()
}

func (h *hookHandler) getSQLFiles(event *PushEvent) []string {
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

func isIPInRange(ipStr, cidrStr string) bool {
	ip := net.ParseIP(strings.Split(ipStr, ":")[0]) // Remove port if present
	if ip == nil {
		return false
	}

	_, ipNet, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return false
	}

	return ipNet.Contains(ip)
}
