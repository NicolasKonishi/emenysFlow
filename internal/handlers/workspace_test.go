package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"buffetflow/internal/database"
	"buffetflow/internal/repositories"
	"buffetflow/internal/services"
)

func TestWorkspaceForUsesNavAndCookie(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := workspaceFor(request, "events"); got != "online" {
		t.Fatalf("events workspace %q", got)
	}
	if got := workspaceFor(request, "offline"); got != "offline" {
		t.Fatalf("offline workspace %q", got)
	}
	if got := workspaceFor(request, "layouts"); got != "online" {
		t.Fatalf("layouts should stay online on a live request, got %q", got)
	}
	request.AddCookie(&http.Cookie{Name: workspaceCookie, Value: "offline"})
	if got := workspaceFor(request, "layouts"); got != "online" {
		t.Fatalf("layouts served by the server should be online even with offline cookie, got %q", got)
	}
}

func TestSafeWorkspaceNextStaysOnLayouts(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/workspace", nil)
	request.Host = "example.test"
	if got := safeWorkspaceNext("/layouts", request); got != "/layouts" {
		t.Fatalf("got %q", got)
	}
	if got := safeWorkspaceNext("/offline", request); got != "/" {
		t.Fatalf("offline next should go home, got %q", got)
	}
	if got := safeWorkspaceNext("https://evil.test/layouts", request); got != "/" {
		t.Fatalf("external next %q", got)
	}
}

func TestNormalizeWorkspace(t *testing.T) {
	if got := normalizeWorkspace("OFFLINE"); got != "offline" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeWorkspace("unknown"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestWorkspacePagesRenderAfterLogin(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "workspace-test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	store := repositories.New(db)
	auth := services.NewAuthService(store)
	if err := auth.EnsureDemoAdmin(ctx); err != nil {
		t.Fatal(err)
	}
	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatal(err)
	}
	app := New(store, auth, services.NewChecklistService(store), slog.New(slog.NewTextHandler(io.Discard, nil)), location)
	server := httptest.NewServer(app.Routes())
	defer server.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	response, err := client.Get(server.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	health, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(string(health), `"ok":true`) {
		t.Fatalf("health %d %s", response.StatusCode, health)
	}

	response, err = client.PostForm(server.URL+"/login", url.Values{"email": {"admin@buffet.local"}, "password": {"admin123"}})
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("login status %d", response.StatusCode)
	}

	checks := map[string]string{
		"/":        "Próximos eventos",
		"/online":  "Próximos eventos",
		"/offline": "Checklists e layout das festas",
	}
	for path, expected := range checks {
		response, err := client.Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("GET %s status %d", path, response.StatusCode)
		}
		if !strings.Contains(string(body), expected) {
			t.Fatalf("GET %s missing %q", path, expected)
		}
	}

	response, err = client.Get(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	home, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if !strings.Contains(string(home), "workspace-online") {
		t.Fatal("home should render the full online workspace")
	}
	if !strings.Contains(string(home), "href=\"/layouts\"") {
		t.Fatal("online workspace should include layouts")
	}

	response, err = client.PostForm(server.URL+"/workspace", url.Values{"workspace": {"offline"}})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("POST /workspace status %d", response.StatusCode)
	}
	if !strings.Contains(string(body), "Checklists e layout das festas") {
		t.Fatal("switching to offline did not render the offline hub")
	}
	if !hasWorkspaceCookie(jar, server.URL, "offline") {
		t.Fatal("workspace cookie was not set to offline")
	}

	response, err = client.Get(server.URL + "/layouts")
	if err != nil {
		t.Fatal(err)
	}
	layoutsBody, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET /layouts status %d", response.StatusCode)
	}
	if !strings.Contains(string(layoutsBody), "workspace-online") {
		t.Fatal("live layouts page should render the full online workspace")
	}
	if strings.Contains(string(layoutsBody), "Sem conexão com o emenysFlow. Monte a planta neste aparelho.") {
		t.Fatal("live layouts page should not use the offline-only copy")
	}
	if strings.Contains(string(layoutsBody), "Não foi possível concluir a operação.") {
		t.Fatal("live layouts page should not fail when listing standalone floor layouts")
	}
}

func hasWorkspaceCookie(jar http.CookieJar, rawURL, want string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	for _, cookie := range jar.Cookies(parsed) {
		if cookie.Name == workspaceCookie && cookie.Value == want {
			return true
		}
	}
	return false
}
