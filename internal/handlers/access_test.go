package handlers

import (
	"net/http"
	"testing"

	"buffetflow/internal/models"
)

func TestPermissionFor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		method, path, want string
	}{
		{http.MethodGet, "/", ""},
		{http.MethodGet, "/calendar", models.PermEventView},
		{http.MethodGet, "/events", models.PermEventView},
		{http.MethodGet, "/events/12", models.PermEventView},
		{http.MethodGet, "/events/new", models.PermEventEdit},
		{http.MethodGet, "/events/menu-model-preview", models.PermEventEdit},
		{http.MethodGet, "/events/12/edit", models.PermEventEdit},
		{http.MethodPost, "/events/12", models.PermEventEdit},
		{http.MethodPost, "/events/12/notes", models.PermEventEdit},
		{http.MethodPost, "/events/12/cancel", models.PermAdmin},
		{http.MethodGet, "/events/12/layout", models.PermLayouts},
		{http.MethodPost, "/events/12/layout", models.PermLayouts},
		{http.MethodGet, "/events/12/operation", models.PermChecklist},
		{http.MethodPost, "/checklist/items/4/status", models.PermChecklist},
		{http.MethodGet, "/inventory", models.PermInventoryView},
		{http.MethodGet, "/inventory/new", models.PermInventoryEdit},
		{http.MethodPost, "/inventory/8/toggle", models.PermInventoryEdit},
		{http.MethodGet, "/layouts", models.PermLayouts},
		{http.MethodGet, "/catalog", models.PermAdmin},
		{http.MethodGet, "/models", models.PermAdmin},
		{http.MethodGet, "/rules", models.PermAdmin},
		{http.MethodGet, "/settings", models.PermAdmin},
	}
	for _, tc := range cases {
		if got := permissionFor(tc.method, tc.path); got != tc.want {
			t.Errorf("permissionFor(%s %s)=%q want %q", tc.method, tc.path, got, tc.want)
		}
	}
}
