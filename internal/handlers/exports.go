package handlers

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"buffetflow/internal/models"
)

func (a *App) exportChecklistCSV(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	event, err := a.store.GetEvent(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	checklist, err := a.store.GetChecklistByEvent(r.Context(), id)
	if err != nil {
		http.Error(w, databaseErrorMessage(err), 500)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="checklist-evento-%d.csv"`, id))
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(w)
	writer.Comma = ';'
	_ = writer.Write([]string{"Evento", event.Name, "Cliente", event.ClientName, "Local", event.Venue, "Convidados", strconv.Itoa(event.GuestCount)})
	_ = writer.Write([]string{"Categoria", "Item", "Quantidade", "Unidade", "Disponível", "Faltante", "Localização", "Status", "Origem"})
	for _, item := range checklist.Items {
		_ = writer.Write([]string{item.CategoryName, item.Name, fmt.Sprintf("%g", item.RequiredQuantity), item.Unit, fmt.Sprintf("%g", item.AvailableQuantity), fmt.Sprintf("%g", item.MissingQuantity), item.LocationSnapshot, item.Status, item.CalculationOrigin})
	}
	writer.Flush()
}

func (a *App) exportChecklistPDF(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	event, err := a.store.GetEvent(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	checklist, err := a.store.GetChecklistByEvent(r.Context(), id)
	if err != nil {
		http.Error(w, databaseErrorMessage(err), 500)
		return
	}
	document := buildSimplePDF(event, checklist)
	w.Header().Set("Content-Type", "application/pdf")
	disposition := "inline"
	if r.URL.Query().Get("download") == "1" {
		disposition = "attachment"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename="checklist-evento-%d.pdf"`, disposition, id))
	w.Header().Set("Content-Length", strconv.Itoa(len(document)))
	_, _ = w.Write(document)
}

func (a *App) pdfViewer(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	event, err := a.store.GetEvent(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if _, err := a.store.GetChecklistByEvent(r.Context(), id); err != nil {
		http.Error(w, databaseErrorMessage(err), http.StatusInternalServerError)
		return
	}
	data := a.baseData(r, "Visualizar PDF", "events")
	data.Event = event
	a.render(w, r, "pdf_viewer", data)
}

func (a *App) createEventShare(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	raw := make([]byte, 24)
	if _, err = rand.Read(raw); err != nil {
		http.Error(w, "Não foi possível gerar o link.", 500)
		return
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(token))
	user := currentUser(r)
	if err = a.store.CreateEventShare(r.Context(), id, hex.EncodeToString(sum[:]), user.ID); err != nil {
		http.Error(w, databaseErrorMessage(err), 500)
		return
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	shareURL := scheme + "://" + r.Host + "/share/" + token
	a.redirect(w, r, fmt.Sprintf("/events/%d?message=%s", id, url.QueryEscape("Link somente leitura: "+shareURL)), http.StatusSeeOther)
}

func (a *App) sharedEvent(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	sum := sha256.Sum256([]byte(token))
	eventID, err := a.store.EventByShareToken(r.Context(), hex.EncodeToString(sum[:]))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	event, err := a.store.GetEvent(r.Context(), eventID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	checklist, err := a.store.GetChecklistByEvent(r.Context(), eventID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data := PageData{Title: "Checklist compartilhada", Public: true, Event: event, Checklist: checklist, Groups: groupChecklist(checklist.Items)}
	a.render(w, r, "shared_event", data)
}

func buildSimplePDF(event models.Event, checklist models.Checklist) []byte {
	lines := []string{"BUFFET - " + event.ClientName + " - " + strconv.Itoa(event.GuestCount) + " pessoas", event.Name + " | " + event.Venue, event.StartsAt.Local().Format("02/01/2006 15:04"), ""}
	category := ""
	for _, item := range checklist.Items {
		if item.CategoryName != category {
			category = item.CategoryName
			lines = append(lines, "", strings.ToUpper(category))
		}
		line := fmt.Sprintf("[  ] %g %s - %s", item.RequiredQuantity, item.Unit, item.Name)
		if item.LocationSnapshot != "" {
			line += " | " + item.LocationSnapshot
		}
		if item.MissingQuantity > 0 {
			line += fmt.Sprintf(" | FALTAM %g", item.MissingQuantity)
		}
		lines = append(lines, line)
	}
	const perPage = 48
	pages := (len(lines) + perPage - 1) / perPage
	if pages < 1 {
		pages = 1
	}
	objects := make([][]byte, 4+pages*2)
	pageRefs := make([]string, pages)
	objects[1] = []byte("<< /Type /Catalog /Pages 2 0 R >>")
	objects[3] = []byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	for p := 0; p < pages; p++ {
		pageObj := 4 + p*2
		contentObj := pageObj + 1
		pageRefs[p] = fmt.Sprintf("%d 0 R", pageObj)
		start := p * perPage
		end := start + perPage
		if end > len(lines) {
			end = len(lines)
		}
		var content strings.Builder
		content.WriteString("BT /F1 10 Tf 40 800 Td 14 TL ")
		for _, line := range lines[start:end] {
			content.WriteString("(" + pdfEscape(asciiText(line)) + ") Tj T* ")
		}
		content.WriteString("ET")
		stream := content.String()
		objects[pageObj] = []byte(fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>", contentObj))
		objects[contentObj] = []byte(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
	}
	objects[2] = []byte("<< /Type /Pages /Kids [" + strings.Join(pageRefs, " ") + "] /Count " + strconv.Itoa(pages) + " >>")
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects))
	for i := 1; i < len(objects); i++ {
		if objects[i] == nil {
			continue
		}
		offsets[i] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i, objects[i])
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects))
	for i := 1; i < len(objects); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(objects), xref)
	return out.Bytes()
}

func pdfEscape(value string) string {
	return strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)").Replace(value)
}
func asciiText(value string) string {
	return strings.NewReplacer("á", "a", "à", "a", "ã", "a", "â", "a", "Á", "A", "é", "e", "ê", "e", "É", "E", "í", "i", "Í", "I", "ó", "o", "ô", "o", "õ", "o", "Ó", "O", "ú", "u", "Ú", "U", "ç", "c", "Ç", "C", "—", "-", "–", "-").Replace(value)
}
