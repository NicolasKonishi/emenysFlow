package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"buffetflow/internal/models"
)

func (a *App) rules(writer http.ResponseWriter, request *http.Request) {
	data := a.baseData(request, "Regras de cálculo", "rules")
	rules, err := a.store.ListRules(request.Context(), true)
	if err != nil {
		data.Error = databaseErrorMessage(err)
	} else {
		data.Rules = rules
	}
	a.render(writer, request, "rules", data)
}

func (a *App) ruleForm(writer http.ResponseWriter, request *http.Request) {
	if err := a.requireAdmin(request); err != nil {
		http.Error(writer, "Acesso restrito ao administrador.", http.StatusForbidden)
		return
	}
	data := a.baseData(request, "Nova regra", "rules")
	data.FormAction = "/rules"
	data.Rule = models.CalculationRule{CalculationType: "fixed", Divisor: 1, Multiplier: 1, ConditionJSON: "{}", Priority: 100, Active: true}
	if request.PathValue("id") != "" {
		id, err := pathID(request)
		if err != nil {
			http.NotFound(writer, request)
			return
		}
		rule, err := a.store.GetRule(request.Context(), id)
		if err != nil {
			http.NotFound(writer, request)
			return
		}
		data.Title = "Editar regra"
		data.Rule = rule
		data.IsEdit = true
		data.FormAction = fmt.Sprintf("/rules/%d", id)
	}
	data.Categories, _ = a.store.ListCategories(request.Context())
	data.Items, _ = a.store.ListInventory(request.Context(), "", "", false)
	a.render(writer, request, "rule_form", data)
}

func (a *App) parseRuleForm(request *http.Request, id int64) (models.CalculationRule, error) {
	if err := request.ParseForm(); err != nil {
		return models.CalculationRule{}, err
	}
	category := parseOptionalInt(request.FormValue("category_id"))
	item := parseOptionalInt(request.FormValue("result_inventory_item_id"))
	rule := models.CalculationRule{ID: id, RuleKey: strings.TrimSpace(request.FormValue("rule_key")), Name: strings.TrimSpace(request.FormValue("name")), Description: strings.TrimSpace(request.FormValue("description")),
		CalculationType: request.FormValue("calculation_type"), BaseValue: parseFloat(request.FormValue("base_value")), Divisor: parseFloat(request.FormValue("divisor")), Multiplier: parseFloat(request.FormValue("multiplier")),
		MinimumQuantity: parseOptionalFloat(request.FormValue("minimum_quantity")), MaximumQuantity: parseOptionalFloat(request.FormValue("maximum_quantity")), SafetyPercent: parseFloat(request.FormValue("safety_percent")),
		ConditionJSON: strings.TrimSpace(request.FormValue("condition_json")), Priority: int(parseFloat(request.FormValue("priority"))), Active: boolForm(request.FormValue("active"))}
	if category.Valid {
		rule.CategoryID = category.Int64
	}
	if item.Valid {
		rule.ResultInventoryItemID = item.Int64
	}
	if rule.RuleKey == "" || rule.Name == "" || rule.CategoryID == 0 || rule.ResultInventoryItemID == 0 {
		return rule, fmt.Errorf("Identificador, nome, categoria e item resultante são obrigatórios.")
	}
	if rule.Divisor <= 0 {
		return rule, fmt.Errorf("O divisor deve ser maior que zero.")
	}
	return rule, nil
}

func (a *App) ruleCreate(writer http.ResponseWriter, request *http.Request) {
	a.saveRule(writer, request, 0)
}
func (a *App) ruleUpdate(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	a.saveRule(writer, request, id)
}
func (a *App) saveRule(writer http.ResponseWriter, request *http.Request, id int64) {
	if err := a.requireAdmin(request); err != nil {
		http.Error(writer, "Acesso restrito.", http.StatusForbidden)
		return
	}
	rule, err := a.parseRuleForm(request, id)
	if err == nil {
		user := currentUser(request)
		err = a.store.SaveRule(request.Context(), &rule, user.ID)
	}
	if err != nil {
		data := a.baseData(request, "Revisar regra", "rules")
		data.Rule = rule
		data.Error = databaseErrorMessage(err)
		if strings.Contains(err.Error(), "obrigatórios") || strings.Contains(err.Error(), "divisor") {
			data.Error = err.Error()
		}
		data.IsEdit = id > 0
		data.FormAction = "/rules"
		if id > 0 {
			data.FormAction = fmt.Sprintf("/rules/%d", id)
		}
		data.Categories, _ = a.store.ListCategories(request.Context())
		data.Items, _ = a.store.ListInventory(request.Context(), "", "", false)
		a.render(writer, request, "rule_form", data)
		return
	}
	a.redirect(writer, request, "/rules?message="+url.QueryEscape("Regra salva. Recalcule os eventos desejados."), http.StatusSeeOther)
}
func (a *App) ruleToggle(writer http.ResponseWriter, request *http.Request) {
	if err := a.requireAdmin(request); err != nil {
		http.Error(writer, "Acesso restrito.", http.StatusForbidden)
		return
	}
	id, err := pathID(request)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	user := currentUser(request)
	err = a.store.ToggleRule(request.Context(), id, user.ID)
	target := "/rules?message=" + url.QueryEscape("Status da regra atualizado.")
	if err != nil {
		target = "/rules?type=danger&message=" + url.QueryEscape(databaseErrorMessage(err))
	}
	a.redirect(writer, request, target, http.StatusSeeOther)
}
