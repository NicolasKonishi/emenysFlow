package handlers

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	kind := checklistPDFKindFromRequest(r)
	checklist.Items = checklistItemsForPDF(checklist.Items, kind)
	eventNotes, err := a.store.ListEventNotes(r.Context(), id)
	if err != nil {
		http.Error(w, databaseErrorMessage(err), http.StatusInternalServerError)
		return
	}
	var decorationProfile models.DecorationProfile
	if kind == checklistPDFDecoration {
		decorationProfile, err = a.store.GetDecorationProfile(r.Context(), id)
		if err != nil {
			http.Error(w, databaseErrorMessage(err), http.StatusInternalServerError)
			return
		}
	}
	document := a.buildChecklistPDF(event, checklist, eventNotes, decorationProfile, kind)
	w.Header().Set("Content-Type", "application/pdf")
	disposition := "inline"
	if r.URL.Query().Get("download") == "1" {
		disposition = "attachment"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, checklistPDFFilename(id, kind)))
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
	data.ActiveTab = string(checklistPDFKindFromRequest(r))
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
	shareURL := a.publicBaseURL(r) + "/share/" + token
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

type checklistPDFKind string

const (
	checklistPDFOperational checklistPDFKind = "operational"
	checklistPDFDecoration  checklistPDFKind = "decoration"
)

func checklistPDFKindFromRequest(r *http.Request) checklistPDFKind {
	switch strings.ToLower(strings.TrimSpace(r.URL.Query().Get("checklist"))) {
	case "decoration", "decoracao", "decoração":
		return checklistPDFDecoration
	default:
		return checklistPDFOperational
	}
}

func checklistPDFFilename(eventID int64, kind checklistPDFKind) string {
	if kind == checklistPDFDecoration {
		return fmt.Sprintf("checklist-decoracao-evento-%d.pdf", eventID)
	}
	// Keep the original filename for the default route, which has historically
	// represented the team's principal checklist.
	return fmt.Sprintf("checklist-evento-%d.pdf", eventID)
}

func checklistItemsForPDF(items []models.ChecklistItem, kind checklistPDFKind) []models.ChecklistItem {
	operational, decoration := splitEventChecklists(models.Checklist{Items: items})
	if kind == checklistPDFDecoration {
		return decoration.Items
	}
	return operational.Items
}

// buildSimplePDF remains for callers that need the original all-items document.
// New exports use buildChecklistPDF so the two operational views stay separate.
func buildSimplePDF(event models.Event, checklist models.Checklist) []byte {
	return buildChecklistPDFDocument(event, checklist, nil, models.DecorationProfile{}, "")
}

func buildChecklistPDF(event models.Event, checklist models.Checklist, eventNotes []models.EventNote, profile models.DecorationProfile, kind checklistPDFKind) []byte {
	title := "CHECKLIST OPERACIONAL"
	if kind == checklistPDFDecoration {
		title = "CHECKLIST DE DECORACAO"
	}
	return buildChecklistPDFDocument(event, checklist, eventNotes, profile, title)
}

// buildChecklistPDF is intentionally kept as a package-level function for
// callers that only need the textual document. The HTTP export has access to
// the private uploads directory and can safely add the original references.
func (a *App) buildChecklistPDF(event models.Event, checklist models.Checklist, eventNotes []models.EventNote, profile models.DecorationProfile, kind checklistPDFKind) []byte {
	title := "CHECKLIST OPERACIONAL"
	if kind == checklistPDFDecoration {
		title = "CHECKLIST DE DECORACAO"
	}
	photos := loadPDFReferenceImages(pdfReferencePhotosForChecklist(eventNotes, profile, kind), a.uploadsDir)
	return buildChecklistPDFDocumentWithImages(event, checklist, eventNotes, profile, title, photos)
}

func buildChecklistPDFDocument(event models.Event, checklist models.Checklist, eventNotes []models.EventNote, profile models.DecorationProfile, title string) []byte {
	return buildChecklistPDFDocumentWithImages(event, checklist, eventNotes, profile, title, nil)
}

func buildChecklistPDFDocumentWithImages(event models.Event, checklist models.Checklist, eventNotes []models.EventNote, profile models.DecorationProfile, title string, photos []pdfReferenceImage) []byte {
	const perPage = 48
	lines := make([]string, 0, 96)
	if title != "" {
		lines = append(lines, title, "")
	}
	lines = append(lines, "BUFFET - "+event.ClientName+" - "+strconv.Itoa(event.GuestCount)+" pessoas", event.Name+" | "+event.Venue, event.StartsAt.Local().Format("02/01/2006 15:04"), "")
	if event.HasCake {
		flavor := strings.TrimSpace(event.CakeNotes)
		if flavor == "" {
			flavor = "sabor a definir"
		}
		lines = append(lines, "BOLO: "+flavor, "")
	} else {
		lines = append(lines, "SEM BOLO", "")
	}
	if strings.TrimSpace(event.Notes) != "" {
		lines = append(lines, "OBSERVACOES DA CHECKLIST")
		lines = appendPDFParagraph(lines, event.Notes)
		lines = append(lines, "")
	}
	lines = appendEventNotesToPDF(lines, eventNotes)
	if title == "CHECKLIST DE DECORACAO" {
		lines = appendDecorationContextToPDF(lines, profile)
	}
	for _, group := range groupChecklistForPDF(checklist.Items) {
		if used := len(lines) % perPage; used != 0 && perPage-used < 3 {
			for len(lines)%perPage != 0 {
				lines = append(lines, "")
			}
		}
		lines = append(lines, "", strings.ToUpper(group.Category))
		for _, item := range group.Items {
			line := fmt.Sprintf("[  ] %g %s - %s", item.RequiredQuantity, item.Unit, item.Name)
			if item.CategoryName != "" {
				line += " | " + item.CategoryName
			}
			if item.LocationSnapshot != "" {
				line += " | " + item.LocationSnapshot
			}
			if item.MissingQuantity > 0 {
				line += fmt.Sprintf(" | FALTAM %g", item.MissingQuantity)
			}
			lines = append(lines, line)
		}
	}
	return renderChecklistPDF(lines, photos)
}

const (
	pdfReferenceImageLimit     = 8
	pdfReferenceImageMaxPixels = 24_000_000
	pdfReferenceImageMaxSide   = 1200
)

type pdfReferencePhotoSource struct {
	storagePath string
	caption     string
	original    string
	section     string
}

type pdfReferenceImage struct {
	jpeg    []byte
	width   int
	height  int
	caption string
	section string
}

type renderedPDFPage struct {
	content    string
	imageIndex int
}

// pdfReferencePhotosForChecklist keeps the reference selection aligned with
// the two checklists: event-note references are useful to both teams, while
// the event decoration references belong only to the decoration document.
func pdfReferencePhotosForChecklist(eventNotes []models.EventNote, profile models.DecorationProfile, kind checklistPDFKind) []pdfReferencePhotoSource {
	sources := make([]pdfReferencePhotoSource, 0)
	for _, note := range eventNotes {
		title := strings.TrimSpace(note.Title)
		if title == "" {
			title = "Anotação"
		}
		for _, photo := range note.Photos {
			sources = append(sources, pdfReferencePhotoSource{
				storagePath: photo.StoragePath,
				caption:     photo.Caption,
				original:    photo.OriginalName,
				section:     "Anotação: " + title,
			})
		}
	}
	if kind != checklistPDFDecoration {
		return sources
	}
	sources = appendPDFReferencePhotoSources(sources, "Referências gerais", profile.Photos)
	for _, composition := range profile.Compositions {
		name := strings.TrimSpace(composition.Name)
		if name == "" {
			name = "Decoração"
		}
		sources = appendPDFReferencePhotoSources(sources, "Referências de "+name, composition.Photos)
	}
	return sources
}

func appendPDFReferencePhotoSources(sources []pdfReferencePhotoSource, section string, photos []models.ReferencePhoto) []pdfReferencePhotoSource {
	for _, photo := range photos {
		sources = append(sources, pdfReferencePhotoSource{
			storagePath: photo.StoragePath,
			caption:     photo.Caption,
			original:    photo.OriginalName,
			section:     section,
		})
	}
	return sources
}

func loadPDFReferenceImages(sources []pdfReferencePhotoSource, uploadsDir string) []pdfReferenceImage {
	photos := make([]pdfReferenceImage, 0, min(len(sources), pdfReferenceImageLimit))
	for _, source := range sources {
		if len(photos) >= pdfReferenceImageLimit {
			break
		}
		photo, ok := loadPDFReferenceImage(source, uploadsDir)
		if ok {
			photos = append(photos, photo)
		}
	}
	return photos
}

// loadPDFReferenceImage deliberately ignores an unavailable or unsupported
// image. Its caption has already been rendered in the text section, so a
// WEBP file or an old missing upload never blocks a useful PDF export.
func loadPDFReferenceImage(source pdfReferencePhotoSource, uploadsDir string) (pdfReferenceImage, bool) {
	path, ok := pdfReferencePhotoPath(source.storagePath, uploadsDir)
	if !ok {
		return pdfReferenceImage{}, false
	}
	file, err := os.Open(path)
	if err != nil {
		return pdfReferenceImage{}, false
	}
	defer file.Close()
	config, format, err := image.DecodeConfig(file)
	if err != nil || (format != "jpeg" && format != "png") || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > pdfReferenceImageMaxPixels {
		return pdfReferenceImage{}, false
	}
	if _, err := file.Seek(0, 0); err != nil {
		return pdfReferenceImage{}, false
	}
	decoded, format, err := image.Decode(file)
	if err != nil || (format != "jpeg" && format != "png") {
		return pdfReferenceImage{}, false
	}
	width, height := scaledPDFReferenceDimensions(config.Width, config.Height)
	thumbnail := resizePDFReferenceImage(decoded, width, height)
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, thumbnail, &jpeg.Options{Quality: 78}); err != nil {
		return pdfReferenceImage{}, false
	}
	return pdfReferenceImage{
		jpeg:    encoded.Bytes(),
		width:   width,
		height:  height,
		caption: referencePhotoLabel(source.caption, source.original),
		section: source.section,
	}, true
}

// pdfReferencePhotoPath applies the same uploads-root boundary used by the
// private image endpoints. Exporting a PDF must never turn a stale database
// value into a file read outside that directory.
func pdfReferencePhotoPath(storagePath, uploadsDir string) (string, bool) {
	if strings.TrimSpace(storagePath) == "" || strings.TrimSpace(uploadsDir) == "" {
		return "", false
	}
	root, err := filepath.Abs(uploadsDir)
	if err != nil {
		return "", false
	}
	candidate, err := filepath.Abs(storagePath)
	if err != nil {
		return "", false
	}
	if !pathIsWithin(root, candidate) {
		return "", false
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", false
	}
	resolvedCandidate, err := filepath.EvalSymlinks(candidate)
	if err != nil || !pathIsWithin(resolvedRoot, resolvedCandidate) {
		return "", false
	}
	return resolvedCandidate, true
}

func pathIsWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) && !filepath.IsAbs(relative)
}

func scaledPDFReferenceDimensions(width, height int) (int, int) {
	if width <= pdfReferenceImageMaxSide && height <= pdfReferenceImageMaxSide {
		return width, height
	}
	scale := float64(pdfReferenceImageMaxSide) / float64(width)
	if height > width {
		scale = float64(pdfReferenceImageMaxSide) / float64(height)
	}
	return max(1, int(float64(width)*scale+0.5)), max(1, int(float64(height)*scale+0.5))
}

func resizePDFReferenceImage(source image.Image, width, height int) *image.RGBA {
	bounds := source.Bounds()
	thumbnail := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		sourceY := bounds.Min.Y + y*bounds.Dy()/height
		for x := 0; x < width; x++ {
			sourceX := bounds.Min.X + x*bounds.Dx()/width
			pixel := color.NRGBAModel.Convert(source.At(sourceX, sourceY)).(color.NRGBA)
			alpha := int(pixel.A)
			thumbnail.SetRGBA(x, y, color.RGBA{
				R: uint8((int(pixel.R)*alpha + 255*(255-alpha) + 127) / 255),
				G: uint8((int(pixel.G)*alpha + 255*(255-alpha) + 127) / 255),
				B: uint8((int(pixel.B)*alpha + 255*(255-alpha) + 127) / 255),
				A: 255,
			})
		}
	}
	return thumbnail
}

func renderChecklistPDF(lines []string, photos []pdfReferenceImage) []byte {
	const perPage = 48
	pages := make([]renderedPDFPage, 0, (len(lines)+perPage-1)/perPage+len(photos))
	for start := 0; start < len(lines) || (start == 0 && len(lines) == 0); start += perPage {
		end := min(start+perPage, len(lines))
		pages = append(pages, renderedPDFPage{content: textPDFPageContent(lines[start:end]), imageIndex: -1})
		if end == len(lines) {
			break
		}
	}
	for index, photo := range photos {
		pages = append(pages, renderedPDFPage{content: imagePDFPageContent(photo, index+1), imageIndex: index})
	}

	// Objects 1–3 are stable so every page can use the same Helvetica font.
	objects := make([][]byte, 4)
	objects[1] = []byte("<< /Type /Catalog /Pages 2 0 R >>")
	objects[3] = []byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	imageObjects := make([]int, len(photos))
	for index, photo := range photos {
		imageObjects[index] = len(objects)
		objects = append(objects, pdfImageObject(photo))
	}
	pageRefs := make([]string, 0, len(pages))
	for _, page := range pages {
		contentObject := len(objects)
		objects = append(objects, pdfStreamObject(page.content))
		pageObject := len(objects)
		resources := "<< /Font << /F1 3 0 R >>"
		if page.imageIndex >= 0 {
			resources += fmt.Sprintf(" /XObject << /Im%d %d 0 R >>", page.imageIndex+1, imageObjects[page.imageIndex])
		}
		resources += " >>"
		objects = append(objects, []byte(fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources %s /Contents %d 0 R >>", resources, contentObject)))
		pageRefs = append(pageRefs, fmt.Sprintf("%d 0 R", pageObject))
	}
	objects[2] = []byte("<< /Type /Pages /Kids [" + strings.Join(pageRefs, " ") + "] /Count " + strconv.Itoa(len(pages)) + " >>")
	return writePDFObjects(objects)
}

func textPDFPageContent(lines []string) string {
	var content strings.Builder
	content.WriteString("BT /F1 10 Tf 40 800 Td 14 TL ")
	for _, line := range lines {
		content.WriteString("(" + pdfEscape(asciiText(line)) + ") Tj T* ")
	}
	content.WriteString("ET")
	return content.String()
}

func imagePDFPageContent(photo pdfReferenceImage, position int) string {
	const (
		pageWidth  = 595.0
		maxWidth   = 500.0
		maxHeight  = 650.0
		bottomEdge = 56.0
	)
	scale := maxWidth / float64(photo.width)
	if float64(photo.height)*scale > maxHeight {
		scale = maxHeight / float64(photo.height)
	}
	width := float64(photo.width) * scale
	height := float64(photo.height) * scale
	x := (pageWidth - width) / 2
	y := bottomEdge + (maxHeight-height)/2
	var content strings.Builder
	content.WriteString("BT /F1 11 Tf 40 804 Td 15 TL ")
	for _, line := range []string{"REFERENCIA FOTOGRAFICA " + strconv.Itoa(position), photo.section, photo.caption} {
		content.WriteString("(" + pdfEscape(asciiText(line)) + ") Tj T* ")
	}
	content.WriteString("ET ")
	fmt.Fprintf(&content, "q %.2f 0 0 %.2f %.2f %.2f cm /Im%d Do Q", width, height, x, y, position)
	return content.String()
}

func pdfImageObject(photo pdfReferenceImage) []byte {
	var object bytes.Buffer
	fmt.Fprintf(&object, "<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length %d >>\nstream\n", photo.width, photo.height, len(photo.jpeg))
	object.Write(photo.jpeg)
	object.WriteString("\nendstream")
	return object.Bytes()
}

func pdfStreamObject(content string) []byte {
	return []byte(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content))
}

func writePDFObjects(objects [][]byte) []byte {
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects))
	for index := 1; index < len(objects); index++ {
		offsets[index] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", index, objects[index])
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects))
	for index := 1; index < len(objects); index++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[index])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(objects), xref)
	return out.Bytes()
}

func appendEventNotesToPDF(lines []string, eventNotes []models.EventNote) []string {
	if len(eventNotes) == 0 {
		return lines
	}
	lines = append(lines, "ANOTACOES DO EVENTO")
	for _, note := range eventNotes {
		header := strings.TrimSpace(note.Title)
		if header == "" {
			header = "Anotacao"
		}
		lines = append(lines, header)
		lines = appendPDFParagraph(lines, note.Content)
		for _, photo := range note.Photos {
			lines = append(lines, "Foto de referencia: "+referencePhotoLabel(photo.Caption, photo.OriginalName))
		}
	}
	return append(lines, "")
}

func appendDecorationContextToPDF(lines []string, profile models.DecorationProfile) []string {
	lines = append(lines, "CONFIGURACAO DA DECORACAO")
	for _, field := range []struct {
		label string
		value string
	}{
		{"Estilo", profile.Style},
		{"Tema", profile.Theme},
		{"Cor do evento", profile.PrimaryColors},
		{"Responsavel", profile.ResponsibleName},
	} {
		if value := strings.TrimSpace(field.value); value != "" {
			lines = appendPDFParagraph(lines, field.label+": "+value)
		}
	}
	if description := strings.TrimSpace(profile.Description); description != "" {
		lines = appendPDFParagraph(lines, description)
	}
	if notes := strings.TrimSpace(profile.Notes); notes != "" {
		lines = append(lines, "Observacoes da decoracao:")
		lines = appendPDFParagraph(lines, notes)
	}
	lines = appendReferencePhotoCaptionsToPDF(lines, "Referencias gerais", profile.Photos)
	for _, composition := range profile.Compositions {
		lines = append(lines, strings.ToUpper(strings.TrimSpace(composition.Name)))
		if description := strings.TrimSpace(composition.Description); description != "" {
			lines = appendPDFParagraph(lines, description)
		}
		if location := strings.TrimSpace(composition.AssemblyLocation); location != "" {
			lines = appendPDFParagraph(lines, "Montagem: "+location)
		}
		if notes := strings.TrimSpace(composition.Notes); notes != "" {
			lines = appendPDFParagraph(lines, "Observacao: "+notes)
		}
		for _, item := range composition.Items {
			line := fmt.Sprintf("- %g un. %s", item.Quantity, item.Name)
			var details []string
			if color := strings.TrimSpace(item.Color); color != "" {
				details = append(details, "cor: "+color)
			}
			if item.ArrangementKind == "natural" {
				details = append(details, "arranjo natural")
			} else if item.ArrangementKind == "permanent" {
				details = append(details, "arranjo permanente")
			}
			if fakeCakeType := strings.TrimSpace(item.FakeCakeType); fakeCakeType != "" {
				details = append(details, "bolo fake: "+fakeCakeType)
			}
			if item.Origin == "rented" {
				details = append(details, "alugar")
			}
			if len(details) > 0 {
				line += " | " + strings.Join(details, "; ")
			}
			lines = appendPDFParagraph(lines, line)
		}
		lines = appendReferencePhotoCaptionsToPDF(lines, "Referencias de "+composition.Name, composition.Photos)
	}
	return append(lines, "")
}

func appendReferencePhotoCaptionsToPDF(lines []string, title string, photos []models.ReferencePhoto) []string {
	if len(photos) == 0 {
		return lines
	}
	lines = append(lines, title+":")
	for _, photo := range photos {
		lines = append(lines, "Foto de referencia: "+referencePhotoLabel(photo.Caption, photo.OriginalName))
	}
	return lines
}

func referencePhotoLabel(caption, originalName string) string {
	if caption = strings.TrimSpace(caption); caption != "" {
		return caption
	}
	if originalName = strings.TrimSpace(originalName); originalName != "" {
		return originalName
	}
	return "sem legenda"
}

func appendPDFParagraph(lines []string, value string) []string {
	const maxLineLength = 86
	for _, paragraph := range strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n") {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		for len([]rune(paragraph)) > maxLineLength {
			cut := maxLineLength
			runes := []rune(paragraph)
			for cut > 0 && runes[cut] != ' ' {
				cut--
			}
			if cut == 0 {
				cut = maxLineLength
			}
			lines = append(lines, string(runes[:cut]))
			paragraph = strings.TrimSpace(string(runes[cut:]))
		}
		if paragraph != "" {
			lines = append(lines, paragraph)
		}
	}
	return lines
}

func groupChecklistForPDF(items []models.ChecklistItem) []models.ChecklistGroup {
	return groupChecklist(items)
}

func pdfEscape(value string) string {
	return strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)").Replace(value)
}
func asciiText(value string) string {
	return strings.NewReplacer("á", "a", "à", "a", "ã", "a", "â", "a", "Á", "A", "À", "A", "Ã", "A", "Â", "A", "é", "e", "ê", "e", "É", "E", "Ê", "E", "í", "i", "Í", "I", "ó", "o", "ô", "o", "õ", "o", "Ó", "O", "Ô", "O", "Õ", "O", "ú", "u", "Ú", "U", "ç", "c", "Ç", "C", "—", "-", "–", "-").Replace(value)
}
