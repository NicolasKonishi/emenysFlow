package handlers

import (
	"net/http"
)

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
	_, token, expires, err := a.auth.Login(request.Context(), request.FormValue("email"), request.FormValue("password"))
	if err != nil {
		data := PageData{Title: "Entrar", Error: "E-mail ou senha inválidos."}
		a.render(writer, request, "login", data)
		return
	}
	http.SetCookie(writer, &http.Cookie{Name: "buffet_session", Value: token, Path: "/", Expires: expires, HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: request.TLS != nil})
	a.redirect(writer, request, "/", http.StatusSeeOther)
}

func (a *App) logout(writer http.ResponseWriter, request *http.Request) {
	if cookie, err := request.Cookie("buffet_session"); err == nil {
		_ = a.auth.Logout(request.Context(), cookie.Value)
	}
	http.SetCookie(writer, &http.Cookie{Name: "buffet_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	a.redirect(writer, request, "/login?message=Sessão+encerrada.", http.StatusSeeOther)
}
