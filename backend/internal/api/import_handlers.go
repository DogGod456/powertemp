package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"powertemp/backend/internal/importutil"
)

// importFile валидирует загруженный XLSX целиком. Если найдена хотя бы одна
// ошибка структуры или данных, строки не сохраняются в БД.
func (a *API) importFile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("не удалось прочитать multipart form: %w", err))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("поле file обязательно"))
		return
	}
	defer file.Close()

	parsed, err := importutil.ParseUploadedFile(file, header.Filename)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(parsed.Errors) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"message": "Файл содержит ошибки. Проверьте структуру и значения данных.",
			"errors":  parsed.Errors,
		})
		return
	}

	importFile, err := a.store.CreateImportFile(r.Context(), header.Filename, len(parsed.Measurements))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	for i := range parsed.Measurements {
		parsed.Measurements[i].ImportFileID = &importFile.ID
		parsed.Measurements[i].SourceType = "file"
	}
	if err := a.store.InsertMeasurements(r.Context(), parsed.Measurements); err != nil {
		_ = a.store.DeleteImportFile(context.Background(), importFile.ID)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"import": importFile, "rows": len(parsed.Measurements)})
}

// listImports отдает список успешно импортированных XLSX-источников.
func (a *API) listImports(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListImportFiles(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// deleteImport удаляет запись импорта; связанные измерения удаляются каскадно
// ограничением ON DELETE CASCADE в схеме PostgreSQL.
func (a *API) deleteImport(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := a.store.DeleteImportFile(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
