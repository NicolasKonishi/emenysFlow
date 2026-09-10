package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"buffetflow/internal/models"
)

func permissionFor(method, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "/"
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	switch {
	case path == "/", path == "/online", path == "/offline", path == "/workspace", path == "/logout":
		return ""
	case strings.HasPrefix(path, "/api/"):
		return ""
	case strings.HasPrefix(path, "/settings"):
		return models.PermAdmin
	case strings.HasPrefix(path, "/rules"), strings.HasPrefix(path, "/catalog"), strings.HasPrefix(path, "/models"):
		return models.PermAdmin
	case strings.HasPrefix(path, "/decorations"), strings.HasPrefix(path, "/photos/"):
		return models.PermAdmin
	case strings.HasPrefix(path, "/layouts"):
		return models.PermLayouts
	case strings.HasPrefix(path, "/inventory"):
		if method != http.MethodGet || strings.Contains(path, "/new") || strings.Contains(path, "/edit") {
			return models.PermInventoryEdit
		}
		return models.PermInventoryView
	case strings.HasPrefix(path, "/checklist"):
		return models.PermChecklist
	case strings.HasPrefix(path, "/events"):
		return eventPermission(method, path)
	default:
		return models.PermAdmin
	}
}

func eventPermission(method, path string) string {
	if strings.Contains(path, "/layout") {
		return models.PermLayouts
	}
	if strings.Contains(path, "/operation") || strings.Contains(path, "/return") || strings.HasSuffix(path, "/finalize") || strings.Contains(path, "/checklist") {
		return models.PermChecklist
	}
	if method == http.MethodGet && (path == "/events" || isEventShowPath(path) || isEventExportPath(path)) {
		return models.PermEventView
	}
	return models.PermEventEdit
}

func isEventShowPath(path string) bool {
	rest := strings.TrimPrefix(path, "/events/")
	if rest == path || rest == "" {
		return false
	}
	if strings.Contains(rest, "/") {
		return false
	}
	_, err := strconv.ParseInt(rest, 10, 64)
	return err == nil
}

func isEventExportPath(path string) bool {
	return strings.HasSuffix(path, "/pdf") || strings.HasSuffix(path, "/export.csv") || strings.HasSuffix(path, "/export.pdf")
}

func (a *App) requireAdmin(request *http.Request) error {
	return a.requirePermission(request, models.PermAdmin)
}

func (a *App) requirePermission(request *http.Request, permission string) error {
	user, ok := request.Context().Value(userContextKey).(models.User)
	if !ok || !user.Can(permission) {
		return fmt.Errorf("acesso restrito")
	}
	return nil
}
