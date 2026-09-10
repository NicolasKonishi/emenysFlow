package models

import "testing"

func TestUserCanCombinesRoles(t *testing.T) {
	t.Parallel()

	admin := User{Roles: []string{RoleAdmin}}
	corre := User{Roles: []string{RoleCorre}}
	agent := User{Roles: []string{RoleAgent}}
	both := User{Roles: []string{RoleCorre, RoleAgent}}

	if !admin.Can(PermEventEdit) || !admin.Can(PermLayouts) || !admin.Can(PermInventoryEdit) {
		t.Fatal("admin should have full access")
	}
	if corre.Can(PermEventEdit) || corre.Can(PermLayouts) || corre.Can(PermCatalog) {
		t.Fatal("corre should not edit events, layouts or catalogs")
	}
	if !corre.Can(PermEventView) || !corre.Can(PermChecklist) || !corre.Can(PermInventoryEdit) {
		t.Fatal("corre should view events, run checklists and edit inventory")
	}
	if agent.Can(PermChecklist) || agent.Can(PermInventoryView) || agent.Can(PermEventEdit) {
		t.Fatal("agent should not run checklists, inventory or event editing")
	}
	if !agent.Can(PermLayouts) || !agent.Can(PermEventView) {
		t.Fatal("agent should edit layouts and view events")
	}
	if !both.Can(PermLayouts) || !both.Can(PermChecklist) || both.Can(PermEventEdit) {
		t.Fatal("combined corre+agent should keep each role's access without becoming admin")
	}
}

func TestLegacyAccessRole(t *testing.T) {
	t.Parallel()
	if got := LegacyAccessRole([]string{RoleAdmin, RoleCorre}); got != "admin" {
		t.Fatalf("got %q", got)
	}
	if got := LegacyAccessRole([]string{RoleAgent}); got != "organizer" {
		t.Fatalf("got %q", got)
	}
	if got := LegacyAccessRole([]string{RoleCorre}); got != "operational" {
		t.Fatalf("got %q", got)
	}
}
