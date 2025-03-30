package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/dfryer1193/gomad/api"
	"github.com/dfryer1193/gomad/internal/rest/managers"
	mjolnirUtils "github.com/dfryer1193/mjolnir/utils"
)

type NamespaceManager interface {
	GetNamespaces() ([]string, error)
}

type MigrationHandler struct {
	migrationsMgr managers.MigrationManager
	namespacesMgr NamespaceManager
}

var (
	handler       *MigrationHandler
	migrationOnce sync.Once
)

func GetMigrationHandler() *MigrationHandler {
	migrationOnce.Do(func() {
		handler = &MigrationHandler{
			namespacesMgr: managers.GetNamespaceManager(),
			migrationsMgr: managers.GetMigrationsManager(),
		}
	})

	return handler
}

func (h *MigrationHandler) GetNamespaces(w http.ResponseWriter, r *http.Request) *mjolnirUtils.ApiError {
	namespaces, err := h.namespacesMgr.GetNamespaces()
	if err != nil {
		return mjolnirUtils.InternalServerErr(fmt.Errorf("error fetching namespaces: %w", err))
	}

	mjolnirUtils.RespondJSON(w, r, http.StatusOK, &api.NamespaceList{Namespaces: namespaces})
	return nil
}

func (h *MigrationHandler) GetMigrationsForNamespace(w http.ResponseWriter, r *http.Request) *mjolnirUtils.ApiError {
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		return mjolnirUtils.BadRequestErr(fmt.Errorf("namespace is required"))
	}

	migrations, err := h.migrationsMgr.GetMigrationsForNamespace(namespace)
	if err != nil {
		return mjolnirUtils.InternalServerErr(fmt.Errorf("error fetching migrations: %w", err))
	}

	mjolnirUtils.RespondJSON(w, r, http.StatusOK, &api.MigrationList{Migrations: migrations})
	return nil
}

func (h *MigrationHandler) GetMigrationById(w http.ResponseWriter, r *http.Request) *mjolnirUtils.ApiError {
	namespace := r.URL.Query().Get("namespace")
	idStr := r.URL.Query().Get("migrationId")
	if idStr == "" {
		return mjolnirUtils.BadRequestErr(fmt.Errorf("migrationId is required"))
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return mjolnirUtils.BadRequestErr(fmt.Errorf("invalid migrationId: must be a positive integer"))
	}

	migration, err := h.migrationsMgr.GetMigrationById(id)
	if err != nil {
		return mjolnirUtils.InternalServerErr(fmt.Errorf("error fetching migration id %d for namespace %s: %w", id, namespace, err))
	}

	mjolnirUtils.RespondJSON(w, r, http.StatusOK, migration)
	return nil
}
