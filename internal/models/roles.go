package models

const (
	RoleAdmin = "admin"
	RoleCorre = "corre"
	RoleAgent = "agent"

	PermEventView     = "events.view"
	PermEventEdit     = "events.edit"
	PermChecklist     = "checklist"
	PermInventoryView = "inventory.view"
	PermInventoryEdit = "inventory.edit"
	PermLayouts       = "layouts"
	PermRules         = "rules"
	PermCatalog       = "catalog"
	PermModels        = "models"
	PermAdmin         = "admin"
)

type Role struct {
	ID          int64
	Slug        string
	Name        string
	Description string
}

func (u User) HasRole(slug string) bool {
	for _, role := range u.Roles {
		if role == slug {
			return true
		}
	}
	return false
}

func (u User) Can(permission string) bool {
	if u.HasRole(RoleAdmin) {
		return true
	}
	switch permission {
	case PermEventView:
		return u.HasRole(RoleCorre) || u.HasRole(RoleAgent)
	case PermEventEdit, PermRules, PermCatalog, PermModels, PermAdmin:
		return false
	case PermChecklist, PermInventoryView, PermInventoryEdit:
		return u.HasRole(RoleCorre)
	case PermLayouts:
		return u.HasRole(RoleAgent)
	default:
		return false
	}
}

func (u User) RoleLabels() string {
	labels := make([]string, 0, len(u.Roles))
	for _, slug := range u.Roles {
		labels = append(labels, RoleDisplayName(slug))
	}
	if len(labels) == 0 {
		return RoleDisplayName(u.Role)
	}
	return joinRoleLabels(labels)
}

func RoleDisplayName(slug string) string {
	switch slug {
	case RoleAdmin:
		return "Administrador"
	case RoleCorre:
		return "Corre"
	case RoleAgent:
		return "Agent"
	case "organizer":
		return "Administrador"
	case "operational":
		return "Corre"
	default:
		return slug
	}
}

func KnownRoleSlugs() []string {
	return []string{RoleAdmin, RoleCorre, RoleAgent}
}

func IsKnownRole(slug string) bool {
	switch slug {
	case RoleAdmin, RoleCorre, RoleAgent:
		return true
	default:
		return false
	}
}

func PrimaryRole(roles []string) string {
	for _, candidate := range []string{RoleAdmin, RoleCorre, RoleAgent} {
		for _, role := range roles {
			if role == candidate {
				return candidate
			}
		}
	}
	return RoleCorre
}

func LegacyAccessRole(roles []string) string {
	switch PrimaryRole(roles) {
	case RoleAdmin:
		return "admin"
	case RoleAgent:
		return "organizer"
	default:
		return "operational"
	}
}

func joinRoleLabels(labels []string) string {
	if len(labels) == 1 {
		return labels[0]
	}
	if len(labels) == 2 {
		return labels[0] + " e " + labels[1]
	}
	return labels[0] + ", " + joinRoleLabels(labels[1:])
}
