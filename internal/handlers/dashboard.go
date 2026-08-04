package handlers

import "net/http"

func (a *App) dashboard(writer http.ResponseWriter, request *http.Request) {
	data := a.baseData(request, "Visão geral", "dashboard")
	dashboard, err := a.store.Dashboard(request.Context())
	if err != nil {
		data.Error = databaseErrorMessage(err)
	} else {
		data.Dashboard = dashboard
	}
	a.render(writer, request, "dashboard", data)
}
