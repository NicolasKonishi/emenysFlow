package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	loginWindow          = 15 * time.Minute
	loginAccountFailures = 5
)

type loginAttempt struct {
	failures int
	first    time.Time
}

type loginLimiter struct {
	mu       sync.Mutex
	attempts map[string]loginAttempt
}

func newLoginLimiter() *loginLimiter { return &loginLimiter{attempts: make(map[string]loginAttempt)} }

func (l *loginLimiter) allowed(keys ...string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	for _, key := range keys {
		attempt, ok := l.attempts[key]
		if !ok || now.Sub(attempt.first) >= loginWindow {
			delete(l.attempts, key)
			continue
		}
		if attempt.failures >= loginAccountFailures {
			return false
		}
	}
	return true
}

func (l *loginLimiter) failed(keys ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	for _, key := range keys {
		attempt := l.attempts[key]
		if attempt.first.IsZero() || now.Sub(attempt.first) >= loginWindow {
			attempt = loginAttempt{first: now}
		}
		attempt.failures++
		l.attempts[key] = attempt
	}
}

func (l *loginLimiter) reset(keys ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, key := range keys {
		delete(l.attempts, key)
	}
}

func (a *App) loginPage(writer http.ResponseWriter, request *http.Request) {
	if cookie, err := request.Cookie("buffet_session"); err == nil {
		if _, err := a.auth.Authenticate(request.Context(), cookie.Value); err == nil {
			a.redirect(writer, request, "/", http.StatusSeeOther)
			return
		}
	}
	data := PageData{Title: "Entrar", Flash: request.URL.Query().Get("message"), FlashType: "danger"}
	a.render(writer, request, "login", data)
}

func (a *App) login(writer http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		http.Error(writer, "Dados inválidos.", http.StatusBadRequest)
		return
	}
	userID, err := strconv.ParseInt(strings.TrimSpace(request.FormValue("id")), 10, 64)
	accountKey := "account:" + strings.TrimSpace(request.FormValue("id"))
	if !a.loginLimiter.allowed(accountKey) {
		data := PageData{Title: "Entrar", Error: "Muitas tentativas. Aguarde alguns minutos e tente novamente."}
		writer.Header().Set("Retry-After", strconv.Itoa(int(loginWindow.Seconds())))
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writer.WriteHeader(http.StatusTooManyRequests)
		a.render(writer, request, "login", data)
		return
	}
	if err != nil || userID <= 0 {
		a.loginLimiter.failed(accountKey)
		data := PageData{Title: "Entrar", Error: "ID ou senha inválidos."}
		a.render(writer, request, "login", data)
		return
	}
	_, token, expires, err := a.auth.Login(request.Context(), userID, request.FormValue("password"))
	if err != nil {
		a.loginLimiter.failed(accountKey)
		data := PageData{Title: "Entrar", Error: "ID ou senha inválidos."}
		a.render(writer, request, "login", data)
		return
	}
	a.loginLimiter.reset(accountKey)
	http.SetCookie(writer, &http.Cookie{Name: "buffet_session", Value: token, Path: "/", Expires: expires, HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: a.secureCookies || request.TLS != nil})
	a.redirect(writer, request, "/", http.StatusSeeOther)
}

func (a *App) logout(writer http.ResponseWriter, request *http.Request) {
	if cookie, err := request.Cookie("buffet_session"); err == nil {
		_ = a.auth.Logout(request.Context(), cookie.Value)
	}
	http.SetCookie(writer, &http.Cookie{Name: "buffet_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: a.secureCookies})
	a.redirect(writer, request, "/login?message=Sessão+encerrada.", http.StatusSeeOther)
}
