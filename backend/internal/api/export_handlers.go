package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"powertemp/backend/internal/domain"
	"powertemp/backend/internal/exportutil"
)

type exportRequest struct {
	Format               string    `json:"format"`
	Mode                 string    `json:"mode"`
	SensorIDs            []string  `json:"sensor_ids"`
	ImportFileIDs        []string  `json:"import_file_ids"`
	From                 time.Time `json:"from"`
	To                   time.Time `json:"to"`
	ForecastTemperatures []float64 `json:"forecast_temperatures"`
}

// toAnalysisRequest переиспользует те же фильтры, что и ручной запуск анализа:
// экспорт сначала считает результат, а затем записывает его в XLSX.
func (r exportRequest) toAnalysisRequest() analysisRequest {
	return analysisRequest{
		Mode:                 r.Mode,
		SensorIDs:            r.SensorIDs,
		ImportFileIDs:        r.ImportFileIDs,
		From:                 r.From,
		To:                   r.To,
		ForecastTemperatures: r.ForecastTemperatures,
	}
}

// createExport выполняет анализ по выбранным источникам и сохраняет XLSX-файл
// с измерениями, результатами, прогнозом и данными для графиков.
func (a *API) createExport(w http.ResponseWriter, r *http.Request) {
	var req exportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	format := strings.ToLower(req.Format)
	if format == "" {
		format = "xlsx"
	}
	if format != "xlsx" {
		writeError(w, http.StatusBadRequest, errors.New("format должен быть xlsx"))
		return
	}
	resp, measurements, err := a.computeAnalysis(r.Context(), req.toAnalysisRequest())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := exportutil.EnsureDir(a.cfg.ExportDir); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	filename := exportutil.FileName("analysis", format, time.Now())
	path := filepath.Join(a.cfg.ExportDir, filename)
	payload := exportutil.Payload{
		Measurements: measurements,
		Results:      resp.Results,
		Forecasts:    resp.Forecasts,
		Points:       resp.Points,
		PeriodFrom:   resp.PeriodFrom,
		PeriodTo:     resp.PeriodTo,
		Mode:         resp.Mode,
	}
	if err := exportutil.WriteXLSX(path, payload); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	exportFile, err := a.store.CreateExportFile(r.Context(), domain.ExportFile{
		Filename: filename,
		Format:   format,
		FilePath: path,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"export": exportFile, "analysis": resp})
}

// listExports возвращает историю созданных XLSX-выгрузок.
func (a *API) listExports(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListExportFiles(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// downloadExport проверяет запись в БД и наличие файла на диске, после чего
// отдает его как attachment для скачивания из браузера.
func (a *API) downloadExport(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	file, err := a.store.GetExportFile(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := os.Stat(file.FilePath); err != nil {
		writeError(w, http.StatusNotFound, errors.New("файл экспорта не найден на диске"))
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", file.Filename))
	http.ServeFile(w, r, file.FilePath)
}
