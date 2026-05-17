package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"powertemp/backend/internal/analysis"
	"powertemp/backend/internal/domain"

	"github.com/google/uuid"
)

type analysisRequest struct {
	Mode                 string    `json:"mode"`
	SensorIDs            []string  `json:"sensor_ids"`
	ImportFileIDs        []string  `json:"import_file_ids"`
	From                 time.Time `json:"from"`
	To                   time.Time `json:"to"`
	ForecastTemperatures []float64 `json:"forecast_temperatures"`
}

type analysisResponse struct {
	Mode       string                `json:"mode"`
	PeriodFrom time.Time             `json:"period_from"`
	PeriodTo   time.Time             `json:"period_to"`
	Results    []analysis.Summary    `json:"results"`
	Forecasts  []analysis.Forecast   `json:"forecasts"`
	Points     []analysis.ChartPoint `json:"points"`
	Warnings   []string              `json:"warnings"`
}

// runAnalysis принимает параметры пользователя, запускает расчет и сохраняет
// краткую запись об успешном анализе в таблицу analysis_runs.
func (a *API) runAnalysis(w http.ResponseWriter, r *http.Request) {
	var req analysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	resp, allMeasurements, err := a.computeAnalysis(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	_ = allMeasurements
	if len(resp.Results) > 0 && !resp.Results[0].InsufficientData {
		r := resp.Results[0]
		corr, aCoef, bCoef, r2 := r.Correlation, r.RegressionA, r.RegressionB, r.RSquared
		_, _ = a.store.CreateAnalysisRun(context.Background(), domain.AnalysisRun{
			Mode:        resp.Mode,
			PeriodFrom:  resp.PeriodFrom,
			PeriodTo:    resp.PeriodTo,
			PointsCount: r.PointsCount,
			Correlation: &corr,
			RegressionA: &aCoef,
			RegressionB: &bCoef,
			RSquared:    &r2,
			Summary:     r.Interpretation,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// computeAnalysis собирает измерения из выбранных датчиков и импортов,
// выбирает режим расчета и возвращает готовые данные для карточек и графиков.
func (a *API) computeAnalysis(ctx context.Context, req analysisRequest) (analysisResponse, []domain.Measurement, error) {
	if req.Mode == "" {
		req.Mode = "combined"
	}
	if req.From.IsZero() {
		req.From = time.Now().Add(-1 * time.Hour).UTC()
	}
	if req.To.IsZero() {
		req.To = time.Now().UTC()
	}
	if req.To.Before(req.From) {
		return analysisResponse{}, nil, errors.New("дата окончания периода не может быть раньше даты начала")
	}
	if len(req.ForecastTemperatures) == 0 {
		req.ForecastTemperatures = []float64{-10, 0, 10, 20}
	}

	sensorIDs, err := parseUUIDList(req.SensorIDs)
	if err != nil {
		return analysisResponse{}, nil, err
	}
	importIDs, err := parseUUIDList(req.ImportFileIDs)
	if err != nil {
		return analysisResponse{}, nil, err
	}
	if len(sensorIDs) == 0 && len(importIDs) == 0 {
		return analysisResponse{}, nil, errors.New("выберите хотя бы один датчик или импортированный файл")
	}
	from, to := req.From, req.To
	filter := domain.MeasurementFilter{SensorIDs: sensorIDs, ImportFileIDs: importIDs, From: &from, To: &to}
	all, err := a.store.QueryMeasurements(ctx, filter)
	if err != nil {
		return analysisResponse{}, nil, err
	}

	resp := newAnalysisResponse(req.Mode, req.From, req.To)

	switch req.Mode {
	case "separate", "comparison":
		for _, sid := range sensorIDs {
			items, err := a.store.QueryMeasurements(ctx, domain.MeasurementFilter{SensorIDs: []uuid.UUID{sid}, From: &from, To: &to})
			if err != nil {
				return analysisResponse{}, nil, err
			}
			label := sid.String()
			if sensor, err := a.store.GetSensor(ctx, sid); err == nil {
				label = fmt.Sprintf("%s — %s", sensor.Code, sensor.Name)
			}
			resp.Results = append(resp.Results, analysis.Compute(label, items))
		}
		for _, iid := range importIDs {
			items, err := a.store.QueryMeasurements(ctx, domain.MeasurementFilter{ImportFileIDs: []uuid.UUID{iid}, From: &from, To: &to})
			if err != nil {
				return analysisResponse{}, nil, err
			}
			label := iid.String()
			if im, err := a.store.GetImportFile(ctx, iid); err == nil {
				label = "Файл: " + im.OriginalFilename
			}
			resp.Results = append(resp.Results, analysis.Compute(label, items))
		}
		combined := analysis.Compute("Общий набор для графика", all)
		resp.Points = analysis.BuildChartPoints(all, combined, 2000)
		if len(resp.Results) > 0 && !resp.Results[0].InsufficientData {
			resp.Forecasts, _ = analysis.ForecastValues(resp.Results[0], req.ForecastTemperatures)
		}
	case "combined":
		fallthrough
	default:
		summary := analysis.Compute("Общий набор данных", all)
		resp.Results = []analysis.Summary{summary}
		resp.Points = analysis.BuildChartPoints(all, summary, 2000)
		if !summary.InsufficientData {
			resp.Forecasts, _ = analysis.ForecastValues(summary, req.ForecastTemperatures)
		}
	}

	resp.normalizeSlices()
	if len(all) == 0 {
		resp.Warnings = append(resp.Warnings, "За выбранный период нет измерений.")
	}
	return resp, all, nil
}

// newAnalysisResponse создает ответ с пустыми слайсами, чтобы frontend всегда
// получал массивы, а не null.
func newAnalysisResponse(mode string, from time.Time, to time.Time) analysisResponse {
	return analysisResponse{
		Mode:       mode,
		PeriodFrom: from,
		PeriodTo:   to,
		Results:    []analysis.Summary{},
		Forecasts:  []analysis.Forecast{},
		Points:     []analysis.ChartPoint{},
		Warnings:   []string{},
	}
}

// normalizeSlices дополнительно защищает JSON-ответ от nil-слайсов после
// ветвлений по режимам анализа.
func (r *analysisResponse) normalizeSlices() {
	if r.Results == nil {
		r.Results = []analysis.Summary{}
	}
	if r.Forecasts == nil {
		r.Forecasts = []analysis.Forecast{}
	}
	if r.Points == nil {
		r.Points = []analysis.ChartPoint{}
	}
	if r.Warnings == nil {
		r.Warnings = []string{}
	}
}
