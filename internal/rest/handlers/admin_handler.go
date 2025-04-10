package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"sync"

	mjolnirUtils "github.com/dfryer1193/mjolnir/utils"
)

type AdminHandler interface {
	Login(w http.ResponseWriter, r *http.Request) *mjolnirUtils.ApiError
	ValidateToken(token string) (bool, error)
}

type adminHandler struct {
	mu          sync.RWMutex
	activeToken string
}

var (
	adminOnce  sync.Once
	authorizer *adminHandler
)

func GetAdminHandler() AdminHandler {
	adminOnce.Do(func() {
		authorizer = &adminHandler{}
	})
	return authorizer
}

func (h *adminHandler) Login(w http.ResponseWriter, r *http.Request) *mjolnirUtils.ApiError {
	var creds struct {
		Password string `json:"password"`
	}

	if _, err := mjolnirUtils.DecodeJSON(r, &creds); err != nil {
		return mjolnirUtils.BadRequestErr(err)
	}

	adminPassword := os.Getenv("GOMAD_ADMIN_SECRET")
	if adminPassword == "" {
		return mjolnirUtils.InternalServerErr(fmt.Errorf("admin secret not configured"))
	}

	if creds.Password != adminPassword {
		return mjolnirUtils.UnauthorizedErr(fmt.Errorf("invalid credentials"))
	}

	token, err := generateToken()
	if err != nil {
		return mjolnirUtils.InternalServerErr(fmt.Errorf("failed to generate token"))
	}

	h.mu.Lock()
	h.activeToken = token
	h.mu.Unlock()

	mjolnirUtils.RespondJSON(w, r, http.StatusOK, map[string]string{
		"token": token,
	})

	return nil
}

func (h *adminHandler) ValidateToken(tokenString string) (bool, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.activeToken == "" {
		return false, fmt.Errorf("no active token")
	}

	return tokenString == h.activeToken, nil
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
