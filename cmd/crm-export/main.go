// crm-export writes the menus, decorations, quantity rules and floor plans of
// this system as one JSON file for the CRM (crm.emenyseventos.com.br), which
// imports it with:
//
//	emenys import-checklists --file emenysflow-checklists.json
//
// The CRM has a simpler model: a rule is "N of an item, per event or for every
// M guests", limited to some menus or to events with decoration. This command
// translates what emenysFlow computes from calculation rules, menu templates,
// containers, equipment and recipes into that model. The file holds business
// data: keep it out of Git.
//
//	go run ./cmd/crm-export -database data/buffetflow.db -out emenysflow-checklists.json
package main

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

type bundle struct {
	Version     int          `json:"version"`
	Source      string       `json:"source"`
	Menus       []named      `json:"menus"`
	Decorations []named      `json:"decorations"`
	Rules       []rule       `json:"rules"`
	Layouts     []floorPlan  `json:"layouts"`
	Skipped     []skipReason `json:"-"`
}

type named struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Active    bool   `json:"active"`
	SortOrder int    `json:"sort_order"`
}

type rule struct {
	Key                string   `json:"key"`
	ItemName           string   `json:"item_name"`
	Unit               string   `json:"unit"`
	Category           string   `json:"category"`
	Quantity           float64  `json:"quantity"`
	PeoplePerUnit      int      `json:"people_per_unit"`
	Rounding           string   `json:"rounding"`
	ExtraPeople        int      `json:"extra_people"`
	MinimumQuantity    *float64 `json:"minimum_quantity"`
	MaximumQuantity    *float64 `json:"maximum_quantity"`
	OnlyWithDecoration bool     `json:"only_with_decoration"`
	Notes              string   `json:"notes"`
	Active             bool     `json:"active"`
	SortOrder          int      `json:"sort_order"`
	MenuKeys           []string `json:"menu_keys"`
}

type floorPlan struct {
	Key        string          `json:"key"`
	Venue      string          `json:"venue"`
	Guests     int             `json:"guests"`
	LayoutJSON json.RawMessage `json:"layout_json"`
}

type skipReason struct{ What, Why string }

func main() {
	databasePath := flag.String("database", "data/buffetflow.db", "SQLite database path")
	outPath := flag.String("out", "", "output file (default: stdout)")
	flag.Parse()

	db, err := sql.Open("sqlite", "file:"+*databasePath+"?mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	exported, err := export(db)
	if err != nil {
		log.Fatal(err)
	}
	var out io.Writer = os.Stdout
	if *outPath != "" {
		file, err := os.OpenFile(*outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		out = file
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(exported); err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stderr, "Exportado: %d cardápios, %d decorações, %d regras, %d layouts\n",
		len(exported.Menus), len(exported.Decorations), len(exported.Rules), len(exported.Layouts))
	for _, skipped := range exported.Skipped {
		fmt.Fprintf(os.Stderr, "  ignorado: %s (%s)\n", skipped.What, skipped.Why)
	}
}

func export(db *sql.DB) (*bundle, error) {
	result := &bundle{Version: 1, Source: "emenysFlow"}
	settings := map[string]float64{"additional_staff_margin": 20, "people_per_coffee_spoon_kit": 50}
	if rows, err := db.Query(`SELECT setting_key, numeric_value FROM operational_settings`); err == nil {
		for rows.Next() {
			var key string
			var value float64
			if rows.Scan(&key, &value) == nil {
				settings[key] = value
			}
		}
		rows.Close()
	}
	if err := exportGeneralRules(db, result, settings); err != nil {
		return nil, fmt.Errorf("general rules: %w", err)
	}
	if err := exportMenus(db, result, settings); err != nil {
		return nil, fmt.Errorf("menus: %w", err)
	}
	if err := exportDecorations(db, result); err != nil {
		return nil, fmt.Errorf("decorations: %w", err)
	}
	if err := exportLayouts(db, result); err != nil {
		return nil, fmt.Errorf("layouts: %w", err)
	}
	return result, nil
}

// categoryFor maps the inventory categories of emenysFlow to the four CRM
// checklist tables.
func categoryFor(inventoryCategory string) string {
	switch strings.ToLower(strings.TrimSpace(inventoryCategory)) {
	case "descartáveis":
		return "disposable"
	case "comidas", "bolo e doces", "sobremesas":
		return "food"
	case "decoração":
		return "decoration"
	default:
		return "equipment"
	}
}

func exportGeneralRules(db *sql.DB, result *bundle, settings map[string]float64) error {
	waiterDivisor := 18.0
	_ = db.QueryRow(`SELECT divisor FROM calculation_rules WHERE rule_key = 'waiters'`).Scan(&waiterDivisor)

	rows, err := db.Query(`
		SELECT r.id, r.name, r.calculation_type, r.base_value, r.divisor, r.multiplier, r.minimum_quantity, r.maximum_quantity,
			r.condition_json, r.priority, i.name, i.unit, cat.name
		FROM calculation_rules r
		JOIN inventory_items i ON i.id = r.result_inventory_item_id
		JOIN inventory_categories cat ON cat.id = r.category_id
		WHERE r.active = 1
		ORDER BY r.priority, r.id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, priority int
		var name, calculation, conditionJSON, item, unit, category string
		var base, divisor, multiplier float64
		var minimum, maximum sql.NullFloat64
		if err := rows.Scan(&id, &name, &calculation, &base, &divisor, &multiplier, &minimum, &maximum, &conditionJSON, &priority, &item, &unit, &category); err != nil {
			return err
		}
		// Staff (garçons, coordenador) is not material to pick up.
		if unit == "profissional" {
			result.Skipped = append(result.Skipped, skipReason{name, "equipe não entra no checklist de material"})
			continue
		}
		out := rule{Key: fmt.Sprintf("rule:%d", id), ItemName: item, Unit: unit, Category: categoryFor(category), Rounding: "group", Active: true, SortOrder: priority, MenuKeys: []string{}}
		if minimum.Valid {
			out.MinimumQuantity = &minimum.Float64
		}
		if maximum.Valid {
			out.MaximumQuantity = &maximum.Float64
		}
		var conditions map[string]any
		_ = json.Unmarshal([]byte(conditionJSON), &conditions)
		if _, ok := conditions["distribution_group"]; ok {
			result.Skipped = append(result.Skipped, skipReason{name, "distribuição percentual"})
			continue
		}
		var notes []string
		for key, label := range map[string]string{
			"welcome_drinks": "welcome drinks", "dadinho_tapioca": "dadinho de tapioca", "coffee_table": "mesa de café",
			"uses_glassware": "taças", "requires_dessert_spoon": "sobremesa de colher",
		} {
			if conditions[key] == true {
				out.Active = false
				notes = append(notes, "no emenysFlow valia só com "+label)
			}
		}
		if conditions["has_main_buffet"] == true {
			notes = append(notes, "no emenysFlow valia só com buffet principal")
		}
		if conditions["use_additional_guest_margin"] == true {
			out.ExtraPeople = int(settings["additional_staff_margin"])
		}
		switch calculation {
		case "fixed":
			out.Quantity = base
			if out.Quantity <= 0 {
				out.Quantity = multiplier
			}
		case "per_person":
			out.Quantity, out.PeoplePerUnit = multiplier, int(math.Max(1, divisor))
		case "group_of_people":
			out.Quantity, out.PeoplePerUnit = multiplier, int(math.Max(1, divisor))
			if conditions["use_coffee_kit_divisor"] == true {
				out.PeoplePerUnit = int(settings["people_per_coffee_spoon_kit"])
			}
		case "per_waiter":
			// N per waiter, with one waiter for every waiterDivisor guests.
			out.Quantity, out.PeoplePerUnit = multiplier/math.Max(1, divisor), int(waiterDivisor)
			notes = append(notes, fmt.Sprintf("por garçom: 1 garçom a cada %.0f pessoas", waiterDivisor))
		default:
			result.Skipped = append(result.Skipped, skipReason{name, "tipo de cálculo " + calculation})
			continue
		}
		if out.Quantity <= 0 {
			result.Skipped = append(result.Skipped, skipReason{name, "quantidade zero"})
			continue
		}
		sort.Strings(notes)
		out.Notes = strings.Join(notes, "; ")
		result.Rules = append(result.Rules, out)
	}
	return rows.Err()
}

type menuDish struct {
	id                        int
	name, categorySlug        string
	containerItem, transport  sql.NullString
	containerUnit, transportU sql.NullString
	containerMode             sql.NullString
	containerCapacity         sql.NullFloat64
	containerFixed            sql.NullFloat64
	calculation, group        string
	divisor, multiplier       float64
	weight                    float64
	resultItem, resultUnit    sql.NullString
}

// exportMenus exports each active menu template and turns what its fixed
// dishes require into rules limited to that menu. Identical rules of several
// menus become one rule for all of them.
func exportMenus(db *sql.DB, result *bundle, settings map[string]float64) error {
	rows, err := db.Query(`SELECT id, name FROM menu_templates WHERE active = 1 AND deleted_at IS NULL ORDER BY id`)
	if err != nil {
		return err
	}
	type menuRow struct {
		id   int
		name string
	}
	var menus []menuRow
	for rows.Next() {
		var menu menuRow
		if err := rows.Scan(&menu.id, &menu.name); err != nil {
			rows.Close()
			return err
		}
		menus = append(menus, menu)
	}
	rows.Close()

	var cubaItem, cubaUnit string
	_ = db.QueryRow(`SELECT name, unit FROM inventory_items WHERE internal_code = 'CUB-001'`).Scan(&cubaItem, &cubaUnit)

	merged := map[string]*rule{}
	var order []string
	add := func(menuKey string, candidate rule) {
		candidate.Quantity = math.Round(candidate.Quantity*100) / 100
		if candidate.Quantity <= 0 {
			return
		}
		signature := fmt.Sprintf("%s|%s|%s|%.2f|%d|%s|%d|%v|%v|%s", candidate.ItemName, candidate.Unit, candidate.Category, candidate.Quantity,
			candidate.PeoplePerUnit, candidate.Rounding, candidate.ExtraPeople, floatOrNil(candidate.MinimumQuantity), floatOrNil(candidate.MaximumQuantity), candidate.Notes)
		if existing, ok := merged[signature]; ok {
			existing.MenuKeys = append(existing.MenuKeys, menuKey)
			return
		}
		sum := sha1.Sum([]byte(signature))
		candidate.Key = "menu-rule:" + hex.EncodeToString(sum[:8])
		candidate.Active, candidate.MenuKeys, candidate.SortOrder = true, []string{menuKey}, 500+len(order)
		merged[signature] = &candidate
		order = append(order, signature)
	}

	for index, menu := range menus {
		menuKey := fmt.Sprintf("menu:%d", menu.id)
		result.Menus = append(result.Menus, named{Key: menuKey, Name: menu.name, Active: true, SortOrder: index + 1})

		dishes, err := menuDishes(db, menu.id)
		if err != nil {
			return fmt.Errorf("dishes of %s: %w", menu.name, err)
		}
		cubas := 0
		type containerKey struct {
			item, unit, mode string
			capacity         float64
		}
		containers := map[containerKey]float64{}
		transports := map[[2]string]float64{}
		distributions := map[string][]menuDish{}
		for _, dish := range dishes {
			if dish.categorySlug == "main_courses" || dish.categorySlug == "sides" {
				cubas++
			} else if dish.containerItem.Valid {
				key := containerKey{dish.containerItem.String, dish.containerUnit.String, dish.containerMode.String, dish.containerCapacity.Float64}
				switch key.mode {
				case "per_event_type", "fixed":
					key.capacity = 0
					value := 1.0
					if key.mode == "fixed" && dish.containerFixed.Valid {
						value = dish.containerFixed.Float64
					}
					containers[key] = math.Max(containers[key], value)
				default:
					containers[key]++
				}
			}
			if dish.transport.Valid {
				transports[[2]string{dish.transport.String, dish.transportU.String}]++
			}
			if dish.calculation == "category_distribution" && dish.resultItem.Valid {
				distributions[dish.group] = append(distributions[dish.group], dish)
			}
		}
		if cubas > 0 && cubaItem != "" {
			maximum := float64(2 * cubas)
			add(menuKey, rule{ItemName: cubaItem, Unit: cubaUnit, Category: "equipment", Quantity: float64(cubas), PeoplePerUnit: 100, Rounding: "group",
				MaximumQuantity: &maximum, Notes: "uma por prato principal ou acompanhamento; duas filas acima de 100 convidados"})
		}
		for key, count := range containers {
			candidate := rule{ItemName: key.item, Unit: key.unit, Category: "equipment", Rounding: "group"}
			switch key.mode {
			case "per_event_type":
				candidate.Quantity, candidate.PeoplePerUnit, candidate.ExtraPeople = 1, 1, int(settings["additional_staff_margin"])
			case "fixed":
				candidate.Quantity = count
			case "per_serving":
				candidate.Quantity, candidate.PeoplePerUnit = count, 1
			default:
				candidate.Quantity, candidate.PeoplePerUnit = count, int(math.Max(1, key.capacity))
			}
			add(menuKey, candidate)
		}
		for key, count := range transports {
			add(menuKey, rule{ItemName: key[0], Unit: key[1], Category: "equipment", Quantity: count, Rounding: "group", Notes: "recipiente de transporte dos pratos"})
		}
		for group, items := range distributions {
			totalWeight := 0.0
			for _, item := range items {
				totalWeight += math.Max(item.weight, 1)
			}
			for _, item := range items {
				share := item.multiplier * math.Max(item.weight, 1) / totalWeight
				add(menuKey, rule{ItemName: item.resultItem.String, Unit: item.resultUnit.String, Category: "equipment", Quantity: share,
					PeoplePerUnit: int(math.Max(1, item.divisor)), Rounding: "ceil", Notes: "bebidas: divisão " + group})
			}
		}
		equipment, err := menuEquipment(db, menu.id)
		if err != nil {
			return err
		}
		for _, item := range equipment {
			add(menuKey, item)
		}
		ingredients, err := menuIngredients(db, menu.id)
		if err != nil {
			return err
		}
		for _, item := range ingredients {
			add(menuKey, item)
		}
	}
	for _, signature := range order {
		sort.Strings(merged[signature].MenuKeys)
		result.Rules = append(result.Rules, *merged[signature])
	}
	return nil
}

// fixedDishFilter selects the dishes always served by a menu template, not the
// ones the client picks from a choice group.
const fixedDishFilter = `
	FROM menu_template_items ti
	JOIN menu_template_sections s ON s.id = ti.menu_template_section_id
	JOIN menu_items mi ON mi.id = ti.menu_item_id
	WHERE s.menu_template_id = ? AND s.active = 1 AND s.deleted_at IS NULL
	  AND ti.active = 1 AND ti.deleted_at IS NULL AND ti.included = 1 AND ti.optional = 0`

func menuDishes(db *sql.DB, menuID int) ([]menuDish, error) {
	rows, err := db.Query(`
		SELECT DISTINCT mi.id, COALESCE(NULLIF(mi.display_name, ''), mi.name), cat.slug,
			ci.name, ci.unit, ct.quantity_mode, COALESCE(mi.container_capacity_portions, ct.capacity_portions), ct.fixed_quantity,
			ti_item.name, ti_item.unit,
			mi.calculation_type, COALESCE(mi.calculation_group, ''), COALESCE(mi.calculation_divisor, 1), COALESCE(mi.calculation_multiplier, 1),
			COALESCE(mi.calculation_weight, 1), ri.name, ri.unit
		`+strings.Replace(fixedDishFilter, "JOIN menu_items mi ON mi.id = ti.menu_item_id", `JOIN menu_items mi ON mi.id = ti.menu_item_id
		JOIN menu_categories cat ON cat.id = mi.category_id
		LEFT JOIN container_types ct ON ct.id = mi.container_type_id
		LEFT JOIN inventory_items ci ON ci.id = ct.inventory_item_id
		LEFT JOIN inventory_items ti_item ON ti_item.id = mi.transport_inventory_item_id
		LEFT JOIN inventory_items ri ON ri.id = mi.result_inventory_item_id`, 1), menuID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dishes []menuDish
	for rows.Next() {
		var dish menuDish
		if err := rows.Scan(&dish.id, &dish.name, &dish.categorySlug, &dish.containerItem, &dish.containerUnit, &dish.containerMode,
			&dish.containerCapacity, &dish.containerFixed, &dish.transport, &dish.transportU, &dish.calculation, &dish.group,
			&dish.divisor, &dish.multiplier, &dish.weight, &dish.resultItem, &dish.resultUnit); err != nil {
			return nil, err
		}
		dishes = append(dishes, dish)
	}
	return dishes, rows.Err()
}

func menuEquipment(db *sql.DB, menuID int) ([]rule, error) {
	rows, err := db.Query(`
		SELECT i.name, i.unit, MAX(me.quantity)
		FROM (SELECT DISTINCT mi.id `+fixedDishFilter+`) dish
		JOIN menu_item_equipment me ON me.menu_item_id = dish.id AND me.required = 1
		JOIN equipment e ON e.id = me.equipment_id
		JOIN inventory_items i ON i.id = e.inventory_item_id
		GROUP BY i.id ORDER BY i.name`, menuID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []rule
	for rows.Next() {
		var item rule
		if err := rows.Scan(&item.ItemName, &item.Unit, &item.Quantity); err != nil {
			return nil, err
		}
		item.Category, item.Rounding, item.Notes = "equipment", "group", "equipamento obrigatório dos pratos"
		rules = append(rules, item)
	}
	return rules, rows.Err()
}

// menuIngredients aggregates the recipes of the fixed dishes: proportional
// ingredients add up per person, fixed ones go once.
func menuIngredients(db *sql.DB, menuID int) ([]rule, error) {
	rows, err := db.Query(`
		SELECT i.name, i.unit, cat.name, ing.calculation_type, ing.quantity, ing.people_divisor
		FROM (SELECT DISTINCT mi.id `+fixedDishFilter+`) dish
		JOIN menu_item_ingredients ing ON ing.menu_item_id = dish.id AND ing.active = 1
		JOIN inventory_items i ON i.id = ing.inventory_item_id
		JOIN inventory_categories cat ON cat.id = i.category_id
		ORDER BY i.name`, menuID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type aggregate struct {
		unit, category      string
		proportional        []float64 // quantity per person of each dish
		divisors            []int
		fixed, group        float64
		groupDivisor        int
		hasProportional     bool
		hasFixed, hasGroups bool
	}
	byItem := map[string]*aggregate{}
	var names []string
	for rows.Next() {
		var name, unit, category, calculation string
		var quantity, divisor float64
		if err := rows.Scan(&name, &unit, &category, &calculation, &quantity, &divisor); err != nil {
			return nil, err
		}
		entry := byItem[name]
		if entry == nil {
			entry = &aggregate{unit: unit, category: categoryFor(category)}
			if entry.category == "equipment" {
				entry.category = "food"
			}
			byItem[name] = entry
			names = append(names, name)
		}
		divisor = math.Max(1, divisor)
		switch calculation {
		case "proportional":
			entry.hasProportional = true
			entry.proportional = append(entry.proportional, quantity)
			entry.divisors = append(entry.divisors, int(divisor))
		case "fixed":
			entry.hasFixed = true
			entry.fixed = math.Max(entry.fixed, quantity)
		default:
			entry.hasGroups = true
			entry.group += quantity
			entry.groupDivisor = int(divisor)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var rules []rule
	for _, name := range names {
		entry := byItem[name]
		switch {
		case entry.hasProportional:
			per := 1
			for _, divisor := range entry.divisors {
				per = lcm(per, divisor)
			}
			if per > 1000 {
				per = entry.divisors[0]
			}
			total := 0.0
			for index, quantity := range entry.proportional {
				total += quantity * float64(per) / float64(entry.divisors[index])
			}
			rules = append(rules, rule{ItemName: name, Unit: entry.unit, Category: entry.category, Quantity: total, PeoplePerUnit: per, Rounding: "exact", Notes: "receita dos pratos"})
		case entry.hasGroups:
			rules = append(rules, rule{ItemName: name, Unit: entry.unit, Category: entry.category, Quantity: entry.group, PeoplePerUnit: entry.groupDivisor, Rounding: "group", Notes: "receita dos pratos"})
		case entry.hasFixed:
			rules = append(rules, rule{ItemName: name, Unit: entry.unit, Category: entry.category, Quantity: entry.fixed, Rounding: "group", Notes: "receita dos pratos"})
		}
	}
	return rules, nil
}

func exportDecorations(db *sql.DB, result *bundle) error {
	rows, err := db.Query(`SELECT id, name FROM decorations WHERE active = 1 ORDER BY name, id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	seen := map[string]bool{}
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}
		name = strings.TrimSpace(name)
		if seen[strings.ToLower(name)] || len([]rune(name)) < 2 {
			continue
		}
		seen[strings.ToLower(name)] = true
		result.Decorations = append(result.Decorations, named{Key: fmt.Sprintf("decoration:%d", id), Name: name, Active: true, SortOrder: len(result.Decorations) + 1})
	}
	return rows.Err()
}

func exportLayouts(db *sql.DB, result *bundle) error {
	rows, err := db.Query(`
		SELECT 'standalone-layout:' || id, COALESCE(NULLIF(venue, ''), name), guest_count, waiter_count, waiter_names_json, layout_json
		FROM standalone_floor_layouts WHERE active = 1
		UNION ALL
		SELECT 'event-layout:' || l.event_id, e.venue, e.guest_count, 0, '[]', l.layout_json
		FROM event_floor_layouts l JOIN events e ON e.id = l.event_id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var plan floorPlan
		var waiterCount int
		var waiterNames, layoutJSON string
		if err := rows.Scan(&plan.Key, &plan.Venue, &plan.Guests, &waiterCount, &waiterNames, &layoutJSON); err != nil {
			return err
		}
		var document map[string]any
		if err := json.Unmarshal([]byte(layoutJSON), &document); err != nil || document == nil {
			result.Skipped = append(result.Skipped, skipReason{plan.Key, "layout com JSON inválido"})
			continue
		}
		var names []string
		if json.Unmarshal([]byte(waiterNames), &names) == nil && len(names) > 0 {
			document["waiters"] = names
		}
		if waiterCount > 0 {
			document["waiter_count"] = waiterCount
		}
		encoded, err := json.Marshal(document)
		if err != nil {
			return err
		}
		plan.LayoutJSON = encoded
		result.Layouts = append(result.Layouts, plan)
	}
	return rows.Err()
}

func floatOrNil(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func lcm(a, b int) int {
	if a <= 0 || b <= 0 {
		return max(a, b, 1)
	}
	return a / gcd(a, b) * b
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
