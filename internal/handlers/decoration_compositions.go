package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"buffetflow/internal/models"
)

func (a *App) decorationCompositionAdd(w http.ResponseWriter, r *http.Request) {
	a.saveDecorationComposition(w, r, 0)
}
func (a *App) decorationCompositionUpdate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("compositionID"), 10, 64)
	a.saveDecorationComposition(w, r, id)
}
func (a *App) saveDecorationComposition(w http.ResponseWriter, r *http.Request, id int64) {
	eventID, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	order, _ := strconv.Atoi(r.FormValue("sort_order"))
	item := models.DecorationComposition{ID: id, Name: strings.TrimSpace(r.FormValue("name")), CompositionType: r.FormValue("composition_type"), Description: strings.TrimSpace(r.FormValue("description")), AssemblyLocation: strings.TrimSpace(r.FormValue("assembly_location")), Notes: strings.TrimSpace(r.FormValue("notes")), SortOrder: order}
	if item.Name == "" {
		err = fmt.Errorf("nome da composição é obrigatório")
	} else {
		err = a.store.SaveDecorationComposition(r.Context(), eventID, &item)
	}
	if err == nil {
		_, err = a.checklist.GenerateTracked(r.Context(), eventID, "decoration_composition_saved", currentUser(r).ID)
	}
	decorationRedirect(w, r, eventID, err, "Composição salva.")
}
func (a *App) decorationCompositionRemove(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	compositionID, _ := strconv.ParseInt(r.PathValue("compositionID"), 10, 64)
	if err == nil {
		err = a.store.RemoveDecorationComposition(r.Context(), eventID, compositionID)
	}
	if err == nil {
		_, err = a.checklist.GenerateTracked(r.Context(), eventID, "decoration_composition_removed", currentUser(r).ID)
	}
	decorationRedirect(w, r, eventID, err, "Composição removida.")
}
func (a *App) decorationCompositionItemAdd(w http.ResponseWriter, r *http.Request) {
	compositionID, _ := strconv.ParseInt(r.PathValue("compositionID"), 10, 64)
	a.saveDecorationCompositionItem(w, r, 0, compositionID)
}
func (a *App) decorationCompositionItemUpdate(w http.ResponseWriter, r *http.Request) {
	itemID, _ := strconv.ParseInt(r.PathValue("itemID"), 10, 64)
	compositionID, _ := strconv.ParseInt(r.FormValue("composition_id"), 10, 64)
	a.saveDecorationCompositionItem(w, r, itemID, compositionID)
}
func (a *App) saveDecorationCompositionItem(w http.ResponseWriter, r *http.Request, itemID, compositionID int64) {
	eventID, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	if compositionID == 0 {
		compositionID, _ = strconv.ParseInt(r.FormValue("composition_id"), 10, 64)
	}
	order, _ := strconv.Atoi(r.FormValue("sort_order"))
	item := models.DecorationCompositionItem{ID: itemID, CompositionID: compositionID, DecorationID: parseOptionalInt(r.FormValue("decoration_id")), InventoryItemID: parseOptionalInt(r.FormValue("inventory_item_id")), Name: strings.TrimSpace(r.FormValue("name")), Quantity: parseFloat(r.FormValue("quantity")), Origin: r.FormValue("origin"), SupplierName: strings.TrimSpace(r.FormValue("supplier_name")), OrderReference: strings.TrimSpace(r.FormValue("order_reference")), RentalStatus: r.FormValue("rental_status"), Notes: strings.TrimSpace(r.FormValue("notes")), SortOrder: order}
	if value := r.FormValue("estimated_cost"); value != "" {
		item.EstimatedCostCents = sql.NullInt64{Int64: int64(parseFloat(value)*100 + 0.5), Valid: true}
	}
	if raw := r.FormValue("pickup_at"); raw != "" {
		item.PickupAt, _ = time.ParseInLocation("2006-01-02T15:04", raw, a.location)
	}
	if raw := r.FormValue("return_at"); raw != "" {
		item.ReturnAt, _ = time.ParseInLocation("2006-01-02T15:04", raw, a.location)
	}
	err = a.store.SaveDecorationCompositionItem(r.Context(), eventID, &item)
	if err == nil {
		_, err = a.checklist.GenerateTracked(r.Context(), eventID, "decoration_item_saved", currentUser(r).ID)
	}
	decorationRedirect(w, r, eventID, err, "Item da decoração salvo.")
}
func (a *App) decorationCompositionItemRemove(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	itemID, _ := strconv.ParseInt(r.PathValue("itemID"), 10, 64)
	if err == nil {
		err = a.store.RemoveDecorationCompositionItem(r.Context(), eventID, itemID)
	}
	if err == nil {
		_, err = a.checklist.GenerateTracked(r.Context(), eventID, "decoration_item_removed", currentUser(r).ID)
	}
	decorationRedirect(w, r, eventID, err, "Item removido da composição.")
}

func (a *App) decorationPhotosUpload(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32<<20)
	if err = r.ParseMultipartForm(32 << 20); err != nil {
		decorationRedirect(w, r, eventID, fmt.Errorf("arquivo excede o limite permitido"), "")
		return
	}
	compositionID := parseOptionalInt(r.FormValue("composition_id"))
	itemID := parseOptionalInt(r.FormValue("composition_item_id"))
	caption := strings.TrimSpace(r.FormValue("caption"))
	files := r.MultipartForm.File["photos"]
	if len(files) == 0 {
		decorationRedirect(w, r, eventID, fmt.Errorf("selecione ao menos uma foto"), "")
		return
	}
	for index, header := range files {
		if header.Size <= 0 || header.Size > 8<<20 {
			err = fmt.Errorf("cada foto deve ter no máximo 8 MB")
			break
		}
		source, openErr := header.Open()
		if openErr != nil {
			err = openErr
			break
		}
		head := make([]byte, 512)
		read, _ := source.Read(head)
		_, _ = source.Seek(0, 0)
		mime := http.DetectContentType(head[:read])
		extensions := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}
		extension, ok := extensions[mime]
		if !ok {
			source.Close()
			err = fmt.Errorf("use imagens JPG, PNG ou WEBP")
			break
		}
		random := make([]byte, 16)
		_, _ = rand.Read(random)
		directory := filepath.Join("data", "uploads", "events", strconv.FormatInt(eventID, 10))
		if mkdirErr := os.MkdirAll(directory, 0o750); mkdirErr != nil {
			source.Close()
			err = mkdirErr
			break
		}
		storagePath := filepath.Join(directory, hex.EncodeToString(random)+extension)
		target, createErr := os.OpenFile(storagePath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o640)
		if createErr != nil {
			source.Close()
			err = createErr
			break
		}
		written, copyErr := io.Copy(target, source)
		closeErr := target.Close()
		source.Close()
		if copyErr != nil {
			err = copyErr
			break
		}
		if closeErr != nil {
			err = closeErr
			break
		}
		clientID := r.FormValue("client_upload_id")
		if len(files) > 1 {
			clientID = fmt.Sprintf("%s-%d", clientID, index)
		}
		photo := models.ReferencePhoto{ClientUploadID: clientID, EventID: eventID, CompositionID: compositionID, CompositionItemID: itemID, StoragePath: storagePath, OriginalName: filepath.Base(header.Filename), MIMEType: mime, FileSize: written, Caption: caption, SortOrder: index, Primary: boolForm(r.FormValue("is_primary"))}
		if err = a.store.SaveReferencePhoto(r.Context(), &photo, currentUser(r).ID); err != nil {
			break
		}
	}
	if wantsJSON(r) {
		writeJSON(w, statusForOperationError(err), map[string]any{"ok": err == nil, "error": errorText(err)})
		return
	}
	decorationRedirect(w, r, eventID, err, "Fotos enviadas.")
}
func (a *App) referencePhotoView(w http.ResponseWriter, r *http.Request) {
	photoID, _ := strconv.ParseInt(r.PathValue("photoID"), 10, 64)
	photo, err := a.store.GetReferencePhoto(r.Context(), photoID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	absolute, err := filepath.Abs(photo.StoragePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	root, _ := filepath.Abs(filepath.Join("data", "uploads"))
	if !strings.HasPrefix(absolute, root+string(os.PathSeparator)) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", photo.MIMEType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	http.ServeFile(w, r, absolute)
}
func (a *App) referencePhotoRemove(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	photoID, _ := strconv.ParseInt(r.PathValue("photoID"), 10, 64)
	if err == nil {
		err = a.store.RemoveReferencePhoto(r.Context(), eventID, photoID)
	}
	decorationRedirect(w, r, eventID, err, "Foto removida.")
}
func decorationRedirect(w http.ResponseWriter, r *http.Request, eventID int64, err error, message string) {
	kind := "success"
	if err != nil {
		kind = "danger"
		message = databaseErrorMessage(err)
	}
	http.Redirect(w, r, fmt.Sprintf("/events/%d/decorations?type=%s&message=%s", eventID, kind, url.QueryEscape(message)), http.StatusSeeOther)
}
