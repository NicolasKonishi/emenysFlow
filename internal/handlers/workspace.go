package handlers

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

const workspaceCookie = "buffet_workspace"

func workspaceFor(request *http.Request, nav string) string {
	if nav == "workspace" {
		return ""
	}
	if nav == "offline" {
		return "offline"
	}
	if workspaceFromRequest(request) == "offline" && offlineCapableNav(nav) {
		return "offline"
	}
	return "online"
}

func offlineCapableNav(nav string) bool {
	switch nav {
	case "layouts", "offline":
		return true
	default:
		return false
	}
}

func workspaceFromRequest(request *http.Request) string {
	cookie, err := request.Cookie(workspaceCookie)
	if err != nil {
		return ""
	}
	return normalizeWorkspace(cookie.Value)
}

func normalizeWorkspace(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "offline":
		return "offline"
	case "online":
		return "online"
	default:
		return ""
	}
}

func setWorkspaceCookie(writer http.ResponseWriter, workspace string) {
	workspace = normalizeWorkspace(workspace)
	if workspace == "" {
		return
	}
	http.SetCookie(writer, &http.Cookie{
		Name:     workspaceCookie,
		Value:    workspace,
		Path:     "/",
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		MaxAge:   365 * 24 * 60 * 60,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *App) health(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")
	writeJSON(writer, http.StatusOK, map[string]any{"ok": true, "service": "emenysFlow"})
}

func (a *App) workspaceChooser(writer http.ResponseWriter, request *http.Request) {
	data := a.baseData(request, "Escolher área", "workspace")
	data.Workspace = ""
	a.render(writer, request, "workspace", data)
}

func (a *App) onlineDashboard(writer http.ResponseWriter, request *http.Request) {
	setWorkspaceCookie(writer, "online")
	data := a.baseData(request, "Visão geral", "dashboard")
	data.Workspace = "online"
	dashboard, err := a.store.Dashboard(request.Context())
	if err != nil {
		data.Error = databaseErrorMessage(err)
	} else {
		data.Dashboard = dashboard
	}
	a.render(writer, request, "dashboard", data)
}

func (a *App) offlineHub(writer http.ResponseWriter, request *http.Request) {
	setWorkspaceCookie(writer, "offline")
	data := a.baseData(request, "Modo offline", "offline")
	data.Workspace = "offline"
	data.Query = request.URL.Query().Get("q")
	events, err := a.store.ListEvents(request.Context(), data.Query)
	if err != nil {
		data.Error = databaseErrorMessage(err)
	} else {
		data.Events = events
	}
	layouts, err := a.store.ListStandaloneFloorLayouts(request.Context(), "")
	if err != nil && data.Error == "" {
		data.Error = databaseErrorMessage(err)
	} else {
		data.StandaloneLayouts = layouts
	}
	a.render(writer, request, "offline_hub", data)
}

func (a *App) setWorkspace(writer http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		http.Error(writer, "Dados inválidos.", http.StatusBadRequest)
		return
	}
	workspace := normalizeWorkspace(request.FormValue("workspace"))
	if workspace == "" {
		a.redirect(writer, request, "/", http.StatusSeeOther)
		return
	}
	setWorkspaceCookie(writer, workspace)
	if workspace == "offline" {
		a.redirect(writer, request, "/offline", http.StatusSeeOther)
		return
	}
	a.redirect(writer, request, "/online", http.StatusSeeOther)
}

func safeWorkspaceNext(next string, request *http.Request) string {
	if next == "" {
		return "/"
	}
	if strings.HasPrefix(next, "/") && !strings.HasPrefix(next, "//") {
		if strings.HasPrefix(next, "/offline") {
			return "/"
		}
		return next
	}
	parsed, err := url.Parse(next)
	if err != nil || parsed.Host != request.Host {
		return "/"
	}
	path := parsed.Path
	if path == "" || strings.HasPrefix(path, "/offline") {
		return "/"
	}
	if parsed.RawQuery != "" {
		return path + "?" + parsed.RawQuery
	}
	return path
}

func operationNav(request *http.Request) string {
	if workspaceFromRequest(request) == "offline" {
		return "offline"
	}
	return "events"
}
