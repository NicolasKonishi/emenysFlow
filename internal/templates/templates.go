package templates

import (
	"fmt"
	"html/template"
	"io"
	"math"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"buffetflow/internal/models"
)

type Renderer struct {
	functions template.FuncMap
}

func NewRenderer() *Renderer {
	return &Renderer{functions: template.FuncMap{
		"dateTime": func(value time.Time) string {
			if value.IsZero() {
				return "—"
			}
			return value.Local().Format("02/01/2006 · 15:04")
		},
		"monthYear": func(value time.Time) string {
			months := []string{"", "janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho", "agosto", "setembro", "outubro", "novembro", "dezembro"}
			if value.IsZero() {
				return ""
			}
			return months[int(value.Month())] + " de " + fmt.Sprintf("%d", value.Year())
		},
		"dateTimeInput": func(value time.Time) string {
			if value.IsZero() {
				return ""
			}
			return value.Local().Format("2006-01-02T15:04")
		},
		"dateInput": func(value time.Time) string {
			if value.IsZero() {
				return ""
			}
			return value.Local().Format("2006-01-02")
		},
		"timeInput": func(value time.Time) string {
			if value.IsZero() {
				return ""
			}
			return value.Local().Format("15:04")
		},
		"number": func(value float64) string {
			if math.Abs(value-math.Round(value)) < 0.0001 {
				return fmt.Sprintf("%.0f", value)
			}
			return strings.ReplaceAll(fmt.Sprintf("%.2f", value), ".", ",")
		},
		"statusLabel":             statusLabel,
		"statusClass":             statusClass,
		"kindLabel":               kindLabel,
		"calcTypeLabel":           calculationTypeLabel,
		"percent":                 func(value float64) int { return int(math.Round(value)) },
		"positive":                func(value float64) bool { return value > 0 },
		"checklistDone":           checklistDone,
		"observationLines":        observationLines,
		"cakeSection":             isCakeSection,
		"omitEventMenuItem":       omitEventMenuItem,
		"inputNumber":             func(value float64) string { return fmt.Sprintf("%g", value) },
		"money":                   func(cents int64) string { return fmt.Sprintf("R$ %.2f", float64(cents)/100) },
		"div":                     func(value int64, divisor float64) float64 { return float64(value) / divisor },
		"sub":                     func(a, b float64) float64 { return math.Max(0, a-b) },
		"fixedRechaudSection":     fixedRechaudSection,
		"noContainerSection":      noContainerSection,
		"rechaudContainerID":      rechaudContainerID,
		"shortageResolutionLabel": shortageResolutionLabel,
		"itemColor":               itemColor,
		"icon":                    iconSVG,
		"eq":                      func(a, b any) bool { return fmt.Sprint(a) == fmt.Sprint(b) },
		"hasRole":                 func(user models.User, slug string) bool { return user.HasRole(slug) },
		"can":                     func(user models.User, permission string) bool { return user.Can(permission) },
		"roleLabels":              func(user models.User) string { return user.RoleLabels() },
		"decorationColors":        decorationColors,
		"arrangementColors":       arrangementColors,
		"fakeCakeTypes":           fakeCakeTypes,
		"decorationSuggestions":   decorationSuggestions,
		"decorationAreaHint":      decorationAreaHint,
		"operationQty": func(item models.ChecklistItem, stage string) float64 {
			switch stage {
			case "separating":
				if item.SeparatedQuantity > 0 {
					return item.SeparatedQuantity
				}
				return item.RequiredQuantity
			case "checking":
				return item.SeparatedQuantity
			case "loading":
				if item.LoadingDecision != "" || item.LoadedQuantity > 0 {
					return item.LoadedQuantity
				}
				return item.SeparatedQuantity
			}
			return item.LoadedQuantity
		},
	}}
}

func isCakeSection(value string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(value)), "bolo")
}

func omitEventMenuItem(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "metriê", "metrie", "maître", "maitre",
		"garçom", "garçons", "cozinheira", "cozinheiras",
		"copeira", "copeiras", "líder", "líderes", "colíder", "colíderes":
		return true
	default:
		return false
	}
}

func observationLines(value string) []string {
	var result []string
	for _, line := range strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			result = append(result, line)
		}
	}
	if len(result) == 0 {
		return []string{""}
	}
	return result
}

func checklistDone(status string) bool {
	switch status {
	case "separated", "checked", "loaded", "at_event", "returned", "not_applicable":
		return true
	default:
		return false
	}
}

func fixedRechaudSection(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	return normalized == "buffet principal" || normalized == "acompanhamentos" || normalized == "pratos principais"
}

func noContainerSection(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	switch normalized {
	case "bebidas", "mesa do café", "mesa de café", "doces", "material", "materiais", "equipe":
		return true
	default:
		return false
	}
}

func rechaudContainerID(containers []models.ContainerType) int64 {
	for _, container := range containers {
		normalized := strings.ToLower(strings.TrimSpace(container.Name))
		if normalized == "cuba gn 1/1" {
			return container.ID
		}
	}
	return 0
}

func itemColor(notes string) string {
	for _, line := range observationLines(notes) {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) < 5 {
			continue
		}
		if strings.HasPrefix(strings.ToLower(trimmed), "cor:") {
			return strings.TrimSpace(trimmed[4:])
		}
	}
	return ""
}

func decorationColors() []string {
	return []string{"Ratan natural (acervo)", "Branco", "Bege", "Areia", "Nude", "Rosa", "Rosé", "Terracota", "Verde", "Azul", "Lilás", "Vermelho", "Preto", "Dourado", "Prata"}
}

func arrangementColors() []string {
	return []string{"Branco", "Bege", "Areia", "Nude", "Rosa", "Rosé", "Terracota", "Verde", "Azul", "Lilás", "Vermelho", "Preto", "Dourado", "Prata"}
}

func fakeCakeTypes() []string {
	return []string{"Clássico branco", "Rústico", "Floral", "Dourado", "Minimalista", "Personalizado"}
}

func decorationSuggestions(kind string) []string {
	switch kind {
	case "cake_table":
		return []string{"Vaso dourado", "Vaso prata", "Vaso de vidro", "Vaso de cerâmica", "Samambaia", "Bandeja para doces", "Bolo fake", "Boleira", "Mesa de decoração", "Arranjo da mesa do bolo"}
	case "guest_tables":
		return []string{"Arranjo de mesa", "Vaso de mesa", "Sousplat", "Toalha de mesa", "Guardanapo", "Número de mesa"}
	case "ceremony":
		return []string{"Tapete da cerimônia", "Cachepô", "Arranjo da cerimônia", "Estrutura do altar"}
	default:
		return []string{"Lounge", "Pranchão", "Painel", "Cordão de luzes", "Peça personalizada"}
	}
}

func decorationAreaHint(kind string) string {
	switch kind {
	case "cake_table":
		return "Vasos, bandejas, bolo fake, boleiras, mesas e arranjos."
	case "guest_tables":
		return "Arranjos, vasos, sousplats, toalhas e detalhes das mesas."
	case "ceremony":
		return "Tapete, cachepôs, altar e arranjos da cerimônia."
	default:
		return "Lounge, pranchão e demais composições especiais."
	}
}

func shortageResolutionLabel(value string) string {
	labels := map[string]string{"purchase": "Comprar", "rental": "Alugar", "substitution": "Substituir", "stock_transfer": "Transferir de outro estoque", "production": "Produzir", "wait_return": "Aguardar devolução", "other": "Outro"}
	if label, ok := labels[value]; ok {
		return label
	}
	return value
}

func (r *Renderer) Render(writer io.Writer, page string, data any, partial bool) error {
	basePath, pagePath := templatePaths(page)
	componentsPath := filepath.Join(filepath.Dir(basePath), "components.html")
	tmpl, err := template.New("base.html").Funcs(r.functions).ParseFiles(basePath, componentsPath, filepath.Join(filepath.Dir(basePath), "layout_editor.html"), pagePath)
	if err != nil {
		return fmt.Errorf("parse template %s: %w", page, err)
	}
	name := "base.html"
	if partial {
		name = "content"
	}
	if err := tmpl.ExecuteTemplate(writer, name, data); err != nil {
		return fmt.Errorf("execute template %s: %w", page, err)
	}
	return nil
}

func templatePaths(page string) (string, string) {
	_, sourceFile, _, ok := runtime.Caller(0)
	directory := filepath.Join("internal", "templates")
	if ok {
		directory = filepath.Dir(sourceFile)
	}
	return filepath.Join(directory, "base.html"), filepath.Join(directory, "pages", page+".html")
}

func statusLabel(value string) string {
	labels := map[string]string{
		"planning": "Planejamento", "reserved": "Reserva", "separating": "Em separação", "checking": "Conferência",
		"loading": "Carregamento", "in_progress": "Em andamento", "returning": "Retorno", "post_event_check": "Conferência pós-evento",
		"completed": "Finalizado", "cancelled": "Cancelado", "pending": "Pendente", "separated": "Separado", "checked": "Conferido",
		"loaded": "Carregado", "at_event": "No evento", "returned": "Retornado", "damaged": "Danificado", "lost": "Perdido",
		"not_applicable": "Não terá",
		"purchasing":     "Em compra", "renting": "Em aluguel", "resolved": "Solucionado",
		"in": "Entrada", "out": "Saída", "adjustment": "Ajuste", "damage": "Dano", "loss": "Perda",
	}
	if label, ok := labels[value]; ok {
		return label
	}
	return value
}

func statusClass(value string) string {
	switch value {
	case "completed", "returned", "checked", "separated":
		return "success"
	case "cancelled", "damaged", "lost":
		return "danger"
	case "reserved", "loaded", "at_event", "in_progress":
		return "info"
	case "separating", "checking", "loading", "returning", "post_event_check":
		return "warning"
	default:
		return "neutral"
	}
}

func kindLabel(value string) string {
	labels := map[string]string{"reusable": "Reutilizável", "consumable": "Consumível", "rented": "Alugado", "outsourced": "Terceirizado"}
	if label, ok := labels[value]; ok {
		return label
	}
	return value
}

func calculationTypeLabel(value string) string {
	labels := map[string]string{
		"fixed": "Fixa por evento", "per_person": "Por pessoa", "group_of_people": "Por grupo de pessoas",
		"per_waiter": "Por garçom", "per_menu_item": "Por item do cardápio", "per_table": "Por mesa",
		"per_dessert": "Por sobremesa", "per_starter": "Por entrada", "per_dish": "Por prato",
		"per_equipment": "Por equipamento", "percentage_distribution": "Distribuição percentual", "custom": "Fórmula personalizada",
	}
	if label, ok := labels[value]; ok {
		return label
	}
	return value
}
