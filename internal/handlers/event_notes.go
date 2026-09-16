package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"buffetflow/internal/models"
)

func (a *App) eventNoteCreate(w http.ResponseWriter, r *http.Request) { a.saveEventNote(w, r, 0) }

func (a *App) eventNoteUpdate(w http.ResponseWriter, r *http.Request) {
	noteID, _ := strconv.ParseInt(r.PathValue("noteID"), 10, 64)
	a.saveEventNote(w, r, noteID)
}

func (a *App) saveEventNote(w http.ResponseWriter, r *http.Request, noteID int64) {
	eventID, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		r.Body = http.MaxBytesReader(w, r.Body, 32<<20)
		err = r.ParseMultipartForm(32 << 20)
	} else {
		err = r.ParseForm()
	}
	if err != nil {
		eventNotesRedirect(w, r, eventID, err, "")
		return
	}
	note := models.EventNote{
		ID: noteID, EventID: eventID, Category: "general",
		Title: strings.TrimSpace(r.FormValue("title")), Content: strings.TrimSpace(r.FormValue("content")),
	}
	if note.Title == "" {
		err = fmt.Errorf("informe um título para a anotação")
	}
	if err == nil && note.ID > 0 && !a.store.EventNoteBelongsToEvent(r.Context(), note.ID, eventID) {
		err = fmt.Errorf("anotação não encontrada")
	}
	if err == nil {
		err = a.store.SaveEventNote(r.Context(), &note, currentUser(r).ID, int(parseFloat(r.FormValue("version"))))
	}
	if err == nil && r.MultipartForm != nil {
		files := r.MultipartForm.File["photos"]
		for index, header := range files {
			if err = a.saveEventNotePhoto(r, eventID, note.ID, header, index, len(files)); err != nil {
				break
			}
		}
	}
	if wantsJSON(r) {
		writeJSON(w, statusForOperationError(err), map[string]any{"ok": err == nil, "id": note.ID, "error": errorText(err)})
		return
	}
	eventNotesRedirect(w, r, eventID, err, "Anotação salva.")
}

func (a *App) eventNoteDelete(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	noteID, _ := strconv.ParseInt(r.PathValue("noteID"), 10, 64)
	if err == nil {
		err = a.store.DeleteEventNote(r.Context(), eventID, noteID)
	}
	eventNotesRedirect(w, r, eventID, err, "Anotação removida.")
}

func eventNotesRedirect(w http.ResponseWriter, r *http.Request, eventID int64, err error, message string) {
	kind := "success"
	if err != nil {
		kind, message = "danger", databaseErrorMessage(err)
	}
	http.Redirect(w, r, fmt.Sprintf("/events/%d?type=%s&message=%s#event-notes", eventID, kind, url.QueryEscape(message)), http.StatusSeeOther)
}

func (a *App) eventNotePhotoUpload(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	noteID, _ := strconv.ParseInt(r.PathValue("noteID"), 10, 64)
	if err == nil && !a.store.EventNoteBelongsToEvent(r.Context(), noteID, eventID) {
		err = fmt.Errorf("anotação não encontrada")
	}
	if err == nil {
		r.Body = http.MaxBytesReader(w, r.Body, 32<<20)
		err = r.ParseMultipartForm(32 << 20)
	}
	if err == nil {
		files := r.MultipartForm.File["photos"]
		if len(files) == 0 {
			err = fmt.Errorf("selecione ao menos uma foto")
		}
		for index, header := range files {
			if err != nil {
				break
			}
			err = a.saveEventNotePhoto(r, eventID, noteID, header, index, len(files))
		}
	}
	if wantsJSON(r) {
		writeJSON(w, statusForOperationError(err), map[string]any{"ok": err == nil, "error": errorText(err)})
		return
	}
	eventNotesRedirect(w, r, eventID, err, "Foto de referência enviada.")
}

func (a *App) saveEventNotePhoto(r *http.Request, eventID, noteID int64, header *multipart.FileHeader, index, total int) error {
	if header.Size <= 0 || header.Size > 8<<20 {
		return fmt.Errorf("cada foto deve ter no máximo 8 MB")
	}
	source, err := header.Open()
	if err != nil {
		return err
	}
	defer source.Close()
	head := make([]byte, 512)
	read, _ := source.Read(head)
	if _, err := source.Seek(0, 0); err != nil {
		return err
	}
	mime := http.DetectContentType(head[:read])
	extensions := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}
	extension, ok := extensions[mime]
	if !ok {
		return fmt.Errorf("use imagens JPG, PNG ou WEBP")
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	directory := filepath.Join(a.uploadsDir, "events", strconv.FormatInt(eventID, 10), "notes")
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return err
	}
	storagePath := filepath.Join(directory, hex.EncodeToString(random)+extension)
	target, err := os.OpenFile(storagePath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o640)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(target, source)
	closeErr := target.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	clientID := r.FormValue("client_upload_id")
	if total > 1 {
		clientID = fmt.Sprintf("%s-%d", clientID, index)
	}
	return a.store.SaveEventNotePhoto(r.Context(), &models.EventNotePhoto{
		ClientUploadID: clientID, EventNoteID: noteID, StoragePath: storagePath,
		OriginalName: filepath.Base(header.Filename), MIMEType: mime, FileSize: written,
		Caption: strings.TrimSpace(r.FormValue("caption")),
	}, currentUser(r).ID)
}

func (a *App) eventNotePhotoView(w http.ResponseWriter, r *http.Request) {
	photoID, _ := strconv.ParseInt(r.PathValue("photoID"), 10, 64)
	photo, err := a.store.GetEventNotePhoto(r.Context(), photoID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	absolute, err := filepath.Abs(photo.StoragePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	root, _ := filepath.Abs(a.uploadsDir)
	if !strings.HasPrefix(absolute, root+string(os.PathSeparator)) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", photo.MIMEType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	http.ServeFile(w, r, absolute)
}
