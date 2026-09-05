package handlers

import (
	"net/http"
	"strings"
	"time"
)

const workspaceCookie = "buffet_workspace"

func workspaceFor(request *http.Request, nav string) string {
	switch nav {
	case "workspace":
		return ""
	case "offline", "layouts":
		return "offline"
	case "dashboard", "events", "inventory", "models", "catalog", "rules", "settings", "decorations":
		return "online"
	}
	if cookie := workspaceFromRequest(request); cookie != "" {
		return cookie
	}
	return "online"
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
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *App) workspaceChooser(writer http.ResponseWriter, request *http.Request) {
	data := a.baseData(request, "Escolher área", "workspace")
	data.Workspace = ""
	a.render(writer, request, "workspace", data)
}

func (a *App) onlineDashboard(writer http.ResponseWriter, request *http.Request) {
	setWorkspaceCookie(writer, "online")
	data := a.baseData(request, "Modo online", "dashboard")
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
