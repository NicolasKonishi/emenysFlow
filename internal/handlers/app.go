package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"buffetflow/internal/models"
	"buffetflow/internal/repositories"
	"buffetflow/internal/services"
	"buffetflow/internal/templates"
	webassets "buffetflow/web"
)

type contextKey string

const userContextKey contextKey = "user"

type App struct {
	store     *repositories.Store
	auth      *services.AuthService
	checklist *services.ChecklistService
	renderer  *templates.Renderer
	logger    *slog.Logger
	location  *time.Location
}

type PageData struct {
	Title                    string
	User                     models.User
	Flash                    string
	FlashType                string
	Error                    string
	CurrentNav               string
	Dashboard                models.Dashboard
	Events                   []models.Event
	Event                    models.Event
	Checklist                models.Checklist
	Groups                   []models.ChecklistGroup
	Items                    []models.InventoryItem
	Item                     models.InventoryItem
	Categories               []models.Category
	Locations                []models.Location
	Rules                    []models.CalculationRule
	Rule                     models.CalculationRule
	MenuItems                []models.MenuItem
	MenuItem                 models.MenuItem
	MenuTemplates            []models.MenuTemplate
	MenuTemplate             models.MenuTemplate
	MenuModels               []models.MenuModel
	ServiceModels            []models.ServiceModel
	KitchenCooks             []models.KitchenCook
	KitchenBoxes             []models.KitchenCookBox
	KitchenBox               models.KitchenCookBox
	BoxInventoryOptions      []models.InventoryItem
	MenuModel                models.MenuModel
	ServiceModel             models.ServiceModel
	ModelSections            []models.MenuModelSection
	SnapshotSections         []models.EventMenuSnapshotSection
	ServiceComponents        []models.ServiceComponent
	CurrentMenuModelID       int64
	SelectedServiceIDs       map[int64]bool
	ModelCustomItems         string
	MenuModelSnapshotVersion int
	MenuModelCurrentVersion  int
	MenuModelOutdated        bool
	ModelDifferences         []models.ModelDifference
	MenuCategories           []models.MenuCategory
	Containers               []models.ContainerType
	Container                models.ContainerType
	Equipment                []models.EquipmentLink
	MenuIngredients          []models.MenuItemIngredient
	RecipeIngredientOptions  []models.InventoryItem
	EventMenu                []models.EventMenuItem
	Operation                models.EventOperation
	ReturnItems              []models.ReturnInspection
	Movements                []models.InventoryMovement
	ShareURL                 string
	Public                   bool
	Users                    []models.User
	Decorations              []models.EventDecoration
	RentedDecorations        []models.DecorationCompositionItem
	DecorationProfile        models.DecorationProfile
	Shortages                []models.ChecklistShortage
	StaffSummary             models.StaffSummary
	FloorLayout              models.EventFloorLayout
	StandaloneLayout         models.StandaloneFloorLayout
	StandaloneLayouts        []models.StandaloneFloorLayout
	LayoutMode               string
	OperationalSettings      []models.OperationalSetting
	Recalculation            models.RecalculationSummary
	CakeConfiguration        models.EventCakeConfiguration
	Query                    string
	Filter                   string
	ActiveTab                string
	IsEdit                   bool
	MenuCustomized           bool
	FormAction               string
	Workspace                string
}

func New(store *repositories.Store, auth *services.AuthService, checklist *services.ChecklistService, logger *slog.Logger, location *time.Location) *App {
	return &App{store: store, auth: auth, checklist: checklist, renderer: templates.NewRenderer(), logger: logger, location: location}
}

func (a *App) Routes() http.Handler {
	root := http.NewServeMux()
	root.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(webassets.StaticFS()))))
	root.HandleFunc("GET /manifest.webmanifest", a.manifest)
	root.HandleFunc("GET /sw.js", a.serviceWorker)
	root.HandleFunc("GET /login", a.loginPage)
	root.HandleFunc("POST /login", a.login)
	root.HandleFunc("GET /share/{token}", a.sharedEvent)
	root.HandleFunc("GET /api/health", a.health)

	protected := http.NewServeMux()
	protected.HandleFunc("GET /", a.onlineDashboard)
	protected.HandleFunc("GET /online", a.onlineDashboard)
	protected.HandleFunc("GET /offline", a.offlineHub)
	protected.HandleFunc("POST /workspace", a.setWorkspace)
	protected.HandleFunc("POST /logout", a.logout)
	protected.HandleFunc("GET /events", a.events)
	protected.HandleFunc("GET /models", a.modelsPage)
	protected.HandleFunc("POST /models/menus", a.menuModelCreate)
	protected.HandleFunc("GET /models/menus/{id}", a.menuModelPage)
	protected.HandleFunc("POST /models/menus/{id}", a.menuModelUpdate)
	protected.HandleFunc("POST /models/menus/{id}/duplicate", a.menuModelDuplicate)
	protected.HandleFunc("POST /models/menus/{id}/toggle", a.menuModelToggle)
	protected.HandleFunc("POST /models/menus/{id}/items", a.menuModelItemAdd)
	protected.HandleFunc("POST /models/menus/{id}/items/{itemID}", a.menuModelItemUpdate)
	protected.HandleFunc("POST /models/menus/{id}/items/{itemID}/remove", a.menuModelItemRemove)
	protected.HandleFunc("POST /models/menus/{id}/groups/{groupID}", a.menuChoiceGroupUpdate)
	protected.HandleFunc("POST /models/menus/{id}/sections", a.menuModelSectionAdd)
	protected.HandleFunc("POST /models/menus/{id}/sections/{sectionID}", a.menuModelSectionUpdate)
	protected.HandleFunc("POST /models/menus/{id}/sections/{sectionID}/remove", a.menuModelSectionRemove)
	protected.HandleFunc("POST /models/services", a.serviceModelCreate)
	protected.HandleFunc("GET /models/services/{id}", a.serviceModelPage)
	protected.HandleFunc("POST /models/services/{id}", a.serviceModelUpdate)
	protected.HandleFunc("POST /models/services/{id}/duplicate", a.serviceModelDuplicate)
	protected.HandleFunc("POST /models/services/{id}/toggle", a.serviceModelToggle)
	protected.HandleFunc("POST /models/services/{id}/components", a.serviceModelComponentAdd)
	protected.HandleFunc("POST /models/services/{id}/components/{componentID}", a.serviceModelComponentUpdate)
	protected.HandleFunc("POST /models/services/{id}/components/{componentID}/remove", a.serviceModelComponentRemove)
	protected.HandleFunc("GET /events/new", a.eventForm)
	protected.HandleFunc("POST /events", a.eventCreate)
	protected.HandleFunc("GET /events/menu-model-preview", a.menuModelPreview)
	protected.HandleFunc("GET /events/{id}", a.eventShow)
	protected.HandleFunc("GET /events/{id}/edit", a.eventForm)
	protected.HandleFunc("POST /events/{id}", a.eventUpdate)
	protected.HandleFunc("POST /events/{id}/generate", a.eventGenerate)
	protected.HandleFunc("POST /events/{id}/reserve", a.eventReserve)
	protected.HandleFunc("POST /events/{id}/duplicate", a.eventDuplicate)
	protected.HandleFunc("POST /events/{id}/cancel", a.eventCancel)
	protected.HandleFunc("POST /events/{id}/checklist/groups/{group}/status", a.checklistGroupStatus)
	protected.HandleFunc("GET /events/{id}/menu-model/compare", a.eventMenuModelCompare)
	protected.HandleFunc("POST /events/{id}/menu-model/reapply", a.eventMenuModelReapply)
	protected.HandleFunc("POST /checklist/items/{id}/status", a.checklistStatus)
	protected.HandleFunc("POST /checklist/items/{id}/override", a.checklistOverride)
	protected.HandleFunc("GET /inventory", a.inventory)
	protected.HandleFunc("GET /decorations", a.decorationCatalogPage)
	protected.HandleFunc("POST /decorations", a.decorationCatalogCreate)
	protected.HandleFunc("POST /decorations/{id}", a.decorationCatalogUpdate)
	protected.HandleFunc("POST /decorations/{id}/toggle", a.decorationCatalogToggle)
	protected.HandleFunc("GET /inventory/kitchen-boxes", a.kitchenBoxesPage)
	protected.HandleFunc("GET /inventory/kitchen-boxes/box/{boxID}", a.kitchenBoxPage)
	protected.HandleFunc("POST /inventory/kitchen-boxes/box/{boxID}/items", a.kitchenBoxItemAdd)
	protected.HandleFunc("POST /inventory/kitchen-boxes/box/{boxID}/items/{itemID}", a.kitchenBoxItemUpdate)
	protected.HandleFunc("POST /inventory/kitchen-boxes/box/{boxID}/items/{itemID}/remove", a.kitchenBoxItemRemove)
	protected.HandleFunc("GET /inventory/new", a.inventoryForm)
	protected.HandleFunc("POST /inventory", a.inventoryCreate)
	protected.HandleFunc("GET /inventory/{id}/edit", a.inventoryForm)
	protected.HandleFunc("POST /inventory/{id}", a.inventoryUpdate)
	protected.HandleFunc("POST /inventory/{id}/toggle", a.inventoryToggle)
	protected.HandleFunc("GET /inventory/{id}/movements", a.inventoryMovements)
	protected.HandleFunc("POST /inventory/{id}/adjust", a.inventoryAdjust)
	protected.HandleFunc("GET /rules", a.rules)
	protected.HandleFunc("GET /rules/new", a.ruleForm)
	protected.HandleFunc("POST /rules", a.ruleCreate)
	protected.HandleFunc("GET /rules/{id}/edit", a.ruleForm)
	protected.HandleFunc("POST /rules/{id}", a.ruleUpdate)
	protected.HandleFunc("POST /rules/{id}/toggle", a.ruleToggle)
	protected.HandleFunc("GET /catalog", a.catalog)
	protected.HandleFunc("GET /catalog/templates/new", a.menuTemplateForm)
	protected.HandleFunc("POST /catalog/templates", a.menuTemplateCreate)
	protected.HandleFunc("GET /catalog/templates/{id}/edit", a.menuTemplateForm)
	protected.HandleFunc("POST /catalog/templates/{id}", a.menuTemplateUpdate)
	protected.HandleFunc("POST /catalog/templates/{id}/toggle", a.menuTemplateToggle)
	protected.HandleFunc("GET /catalog/templates/{templateID}/items/new", a.templateMenuItemForm)
	protected.HandleFunc("POST /catalog/templates/{templateID}/items", a.templateMenuItemCreate)
	protected.HandleFunc("POST /catalog/templates/{templateID}/items/clone", a.templateMenuItemClone)
	protected.HandleFunc("GET /catalog/templates/{templateID}/items/{id}/edit", a.templateMenuItemForm)
	protected.HandleFunc("POST /catalog/templates/{templateID}/items/{id}", a.templateMenuItemUpdate)
	protected.HandleFunc("POST /catalog/templates/{templateID}/items/{id}/toggle", a.templateMenuItemToggle)
	protected.HandleFunc("GET /catalog/items/new", a.menuItemForm)
	protected.HandleFunc("POST /catalog/items", a.menuItemCreate)
	protected.HandleFunc("GET /catalog/items/{id}/edit", a.menuItemForm)
	protected.HandleFunc("POST /catalog/items/{id}", a.menuItemUpdate)
	protected.HandleFunc("POST /catalog/items/{id}/toggle", a.menuItemToggle)
	protected.HandleFunc("POST /catalog/items/{id}/ingredients", a.menuItemIngredientAdd)
	protected.HandleFunc("POST /catalog/items/{id}/ingredients/{ingredientID}", a.menuItemIngredientUpdate)
	protected.HandleFunc("POST /catalog/items/{id}/ingredients/{ingredientID}/remove", a.menuItemIngredientRemove)
	protected.HandleFunc("GET /catalog/containers/new", a.containerForm)
	protected.HandleFunc("POST /catalog/containers", a.containerCreate)
	protected.HandleFunc("GET /catalog/containers/{id}/edit", a.containerForm)
	protected.HandleFunc("POST /catalog/containers/{id}", a.containerUpdate)
	protected.HandleFunc("POST /catalog/containers/{id}/toggle", a.containerToggle)
	protected.HandleFunc("GET /events/{id}/menu", a.eventMenuForm)
	protected.HandleFunc("POST /events/{id}/menu", a.eventMenuSave)
	protected.HandleFunc("POST /events/{id}/menu/snapshot/items", a.eventMenuSnapshotItemAdd)
	protected.HandleFunc("POST /events/{id}/menu/snapshot/items/{itemID}", a.eventMenuSnapshotItemUpdate)
	protected.HandleFunc("POST /events/{id}/menu/snapshot/items/{itemID}/remove", a.eventMenuSnapshotItemRemove)
	protected.HandleFunc("POST /events/{id}/menu/snapshot/items/{itemID}/restore", a.eventMenuSnapshotItemRestore)
	protected.HandleFunc("POST /events/{id}/menu/snapshot/items/{itemID}/containers", a.eventMenuContainerSave)
	protected.HandleFunc("POST /events/{id}/menu/snapshot/items/{itemID}/containers/{containerID}/remove", a.eventMenuContainerRemove)
	protected.HandleFunc("POST /events/{id}/menu/snapshot/items/{itemID}/equipment", a.eventMenuEquipmentSave)
	protected.HandleFunc("POST /events/{id}/menu/snapshot/items/{itemID}/equipment/{equipmentID}/remove", a.eventMenuEquipmentRemove)
	protected.HandleFunc("POST /events/{id}/menu/snapshot/cake", a.eventCakeConfigurationSave)
	protected.HandleFunc("POST /events/{id}/menu/snapshot/restore-model", a.eventMenuSnapshotRestoreModel)
	protected.HandleFunc("GET /events/{id}/operation/{stage}", a.operationPage)
	protected.HandleFunc("POST /events/{id}/operation/{stage}", a.operationSave)
	protected.HandleFunc("POST /events/{id}/operation/loading/items/{itemID}", a.mobileLoadingItem)
	protected.HandleFunc("GET /events/{id}/operation", a.operationHub)
	protected.HandleFunc("POST /events/{id}/operation/items", a.operationManualItem)
	protected.HandleFunc("POST /events/{id}/operation/items/{itemID}/quantity", a.operationQuantity)
	protected.HandleFunc("POST /events/{id}/operation/items/{itemID}/shortage", a.operationShortage)
	protected.HandleFunc("POST /events/{id}/operation/shortages/{shortageID}/status", a.operationShortageStatus)
	protected.HandleFunc("GET /events/{id}/return", a.returnPage)
	protected.HandleFunc("POST /events/{id}/return", a.returnSave)
	protected.HandleFunc("POST /events/{id}/finalize", a.returnFinalize)
	protected.HandleFunc("GET /events/{id}/export.csv", a.exportChecklistCSV)
	protected.HandleFunc("GET /events/{id}/pdf", a.pdfViewer)
	protected.HandleFunc("GET /events/{id}/export.pdf", a.exportChecklistPDF)
	protected.HandleFunc("POST /events/{id}/share", a.createEventShare)
	protected.HandleFunc("GET /settings", a.settings)
	protected.HandleFunc("GET /settings/users/new", a.userForm)
	protected.HandleFunc("POST /settings/users", a.userCreate)
	protected.HandleFunc("GET /settings/users/{id}/edit", a.userForm)
	protected.HandleFunc("POST /settings/users/{id}", a.userUpdate)
	protected.HandleFunc("POST /settings/users/{id}/toggle", a.userToggle)
	protected.HandleFunc("POST /settings/categories", a.categorySave)
	protected.HandleFunc("POST /settings/locations", a.locationSave)
	protected.HandleFunc("POST /settings/operational/{key}", a.operationalSettingSave)
	protected.HandleFunc("GET /layouts", a.layoutsPage)
	protected.HandleFunc("GET /layouts/new", a.layoutForm)
	protected.HandleFunc("POST /layouts", a.layoutCreate)
	protected.HandleFunc("GET /layouts/{id}", a.layoutForm)
	protected.HandleFunc("POST /layouts/{id}", a.layoutUpdate)
	protected.HandleFunc("POST /layouts/{id}/archive", a.layoutArchive)
	protected.HandleFunc("GET /events/{id}/layout", a.eventLayoutPage)
	protected.HandleFunc("POST /events/{id}/layout", a.eventLayoutSave)
	protected.HandleFunc("GET /events/{id}/decorations", a.eventDecorationsPage)
	protected.HandleFunc("POST /events/{id}/decorations", a.eventDecorationsSave)
	protected.HandleFunc("POST /events/{id}/decorations/compositions", a.decorationCompositionAdd)
	protected.HandleFunc("POST /events/{id}/decorations/compositions/{compositionID}", a.decorationCompositionUpdate)
	protected.HandleFunc("POST /events/{id}/decorations/compositions/{compositionID}/remove", a.decorationCompositionRemove)
	protected.HandleFunc("POST /events/{id}/decorations/compositions/{compositionID}/items", a.decorationCompositionItemAdd)
	protected.HandleFunc("POST /events/{id}/decorations/items/{itemID}", a.decorationCompositionItemUpdate)
	protected.HandleFunc("POST /events/{id}/decorations/items/{itemID}/remove", a.decorationCompositionItemRemove)
	protected.HandleFunc("POST /events/{id}/decorations/photos", a.decorationPhotosUpload)
	protected.HandleFunc("GET /photos/{photoID}", a.referencePhotoView)
	protected.HandleFunc("POST /events/{id}/decorations/photos/{photoID}/remove", a.referencePhotoRemove)
	protected.HandleFunc("GET /api/offline/bootstrap", a.offlineBootstrap)
	protected.HandleFunc("POST /api/sync/operations", a.syncOperations)
	root.Handle("/", a.requireAuth(protected))
	return a.securityHeaders(root)
}

func (a *App) baseData(request *http.Request, title, nav string) PageData {
	data := PageData{Title: title, CurrentNav: nav, Flash: request.URL.Query().Get("message"), FlashType: request.URL.Query().Get("type")}
	if data.FlashType == "" {
		data.FlashType = "success"
	}
	if user, ok := request.Context().Value(userContextKey).(models.User); ok {
		data.User = user
	}
	data.Workspace = workspaceFor(request, nav)
	return data
}

func (a *App) render(writer http.ResponseWriter, request *http.Request, page string, data PageData) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.renderer.Render(writer, page, data, request.Header.Get("HX-Request") == "true"); err != nil {
		a.logger.Error("render page", "page", page, "error", err)
	}
}

func (a *App) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		cookie, err := request.Cookie("buffet_session")
		if err != nil {
			a.redirect(writer, request, "/login", http.StatusSeeOther)
			return
		}
		user, err := a.auth.Authenticate(request.Context(), cookie.Value)
		if err != nil || !user.Active {
			http.SetCookie(writer, &http.Cookie{Name: "buffet_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
			a.redirect(writer, request, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(writer, request.WithContext(context.WithValue(request.Context(), userContextKey, user)))
	})
}

func (a *App) requireAdmin(request *http.Request) error {
	user, ok := request.Context().Value(userContextKey).(models.User)
	if !ok || (user.Role != "admin" && user.Role != "organizer") {
		return fmt.Errorf("administrator access required")
	}
	return nil
}

func (a *App) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		if strings.HasSuffix(request.URL.Path, "/export.pdf") {
			writer.Header().Set("X-Frame-Options", "SAMEORIGIN")
		} else {
			writer.Header().Set("X-Frame-Options", "DENY")
		}
		writer.Header().Set("Referrer-Policy", "same-origin")
		writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		writer.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self'; font-src 'self'; manifest-src 'self'; worker-src 'self'; frame-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'")
		if request.Method == http.MethodPost && request.Header.Get("Origin") != "" {
			origin := request.Header.Get("Origin")
			if !strings.HasSuffix(origin, "://"+request.Host) {
				http.Error(writer, "Origem inválida.", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(writer, request)
	})
}

func (a *App) redirect(writer http.ResponseWriter, request *http.Request, target string, status int) {
	if request.Header.Get("HX-Request") == "true" {
		writer.Header().Set("HX-Redirect", target)
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(writer, request, target, status)
}

func currentUser(request *http.Request) models.User {
	user, _ := request.Context().Value(userContextKey).(models.User)
	return user
}

func pathID(request *http.Request) (int64, error) {
	value, err := strconv.ParseInt(request.PathValue("id"), 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid id")
	}
	return value, nil
}

func parseFloat(value string) float64 {
	value = strings.ReplaceAll(strings.TrimSpace(value), ",", ".")
	result, _ := strconv.ParseFloat(value, 64)
	return result
}

func parseOptionalFloat(value string) sql.NullFloat64 {
	if strings.TrimSpace(value) == "" {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: parseFloat(value), Valid: true}
}

func parseOptionalInt(value string) sql.NullInt64 {
	if strings.TrimSpace(value) == "" {
		return sql.NullInt64{}
	}
	result, err := strconv.ParseInt(value, 10, 64)
	return sql.NullInt64{Int64: result, Valid: err == nil}
}

func boolForm(value string) bool { return value == "on" || value == "true" || value == "1" }

func databaseErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "Registro não encontrado."
	case strings.Contains(message, "UNIQUE constraint failed"):
		return "Já existe um registro com esse código ou identificador."
	case strings.Contains(message, "CHECK constraint failed"):
		return "Revise os valores informados."
	default:
		return "Não foi possível concluir a operação. Revise os dados e tente novamente."
	}
}
