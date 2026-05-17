package exportutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"powertemp/backend/internal/analysis"
	"powertemp/backend/internal/domain"

	"github.com/xuri/excelize/v2"
)

// Payload объединяет все данные, которые должны попасть в XLSX-отчет:
// исходные измерения, результаты анализа, прогнозы и точки графиков.
type Payload struct {
	Measurements []domain.Measurement
	Results      []analysis.Summary
	Forecasts    []analysis.Forecast
	Points       []analysis.ChartPoint
	PeriodFrom   time.Time
	PeriodTo     time.Time
	Mode         string
}

// EnsureDir создает каталог выгрузок перед записью файла.
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// FileName формирует уникальное имя отчета по времени создания.
func FileName(prefix, format string, now time.Time) string {
	stamp := now.Format("2006-01-02_15-04-05")
	format = strings.TrimPrefix(strings.ToLower(format), ".")
	return fmt.Sprintf("%s_%s.%s", prefix, stamp, format)
}

// WriteXLSX создает отчет из четырех листов: Measurements, Analysis, Forecast
// и ChartData. Первый лист совместим с форматом обратного импорта.
func WriteXLSX(path string, payload Payload) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	measurementsSheet := "Measurements"
	index, _ := f.NewSheet(measurementsSheet)
	f.SetActiveSheet(index)
	_ = f.DeleteSheet("Sheet1")
	writeMeasurementsSheet(f, measurementsSheet, payload.Measurements)

	analysisSheet := "Analysis"
	_, _ = f.NewSheet(analysisSheet)
	writeAnalysisSheet(f, analysisSheet, payload)

	forecastSheet := "Forecast"
	_, _ = f.NewSheet(forecastSheet)
	writeForecastSheet(f, forecastSheet, payload.Forecasts)

	chartDataSheet := "ChartData"
	_, _ = f.NewSheet(chartDataSheet)
	writeChartDataSheet(f, chartDataSheet, payload.Points)

	for _, sheet := range []string{measurementsSheet, analysisSheet, forecastSheet, chartDataSheet} {
		_ = f.SetColWidth(sheet, "A", "F", 22)
	}
	return f.SaveAs(path)
}

// writeMeasurementsSheet выгружает исходные измерения в том же формате
// столбцов, который ожидает импорт XLSX.
func writeMeasurementsSheet(f *excelize.File, sheet string, measurements []domain.Measurement) {
	headers := []string{"measured_at", "sensor_code", "temperature_c", "consumption_kwh"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	for i, m := range measurements {
		row := i + 2
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), m.MeasuredAt.Format("2006-01-02 15:04:05"))
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), m.SensorCode)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row), m.TemperatureC)
		_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", row), m.ConsumptionKWh)
	}
}

// writeAnalysisSheet записывает параметры расчета и все summary-блоки, чтобы
// отчет был читаем без запуска приложения.
func writeAnalysisSheet(f *excelize.File, sheet string, payload Payload) {
	rows := [][]any{
		{"Показатель", "Значение"},
		{"Режим анализа", payload.Mode},
		{"Период с", payload.PeriodFrom.Format("2006-01-02 15:04:05")},
		{"Период по", payload.PeriodTo.Format("2006-01-02 15:04:05")},
		{"Количество измерений", len(payload.Measurements)},
		{"", ""},
	}
	for _, r := range payload.Results {
		rows = append(rows,
			[]any{"Источник", r.SourceLabel},
			[]any{"Записей", r.PointsCount},
			[]any{"Мин. температура", r.MinTemperature},
			[]any{"Макс. температура", r.MaxTemperature},
			[]any{"Средняя температура", r.AvgTemperature},
			[]any{"Мин. потребление", r.MinConsumption},
			[]any{"Макс. потребление", r.MaxConsumption},
			[]any{"Среднее потребление", r.AvgConsumption},
			[]any{"Коэффициент корреляции Пирсона", r.Correlation},
			[]any{"Коэффициент регрессии a", r.RegressionA},
			[]any{"Свободный коэффициент b", r.RegressionB},
			[]any{"R²", r.RSquared},
			[]any{"Уравнение", r.RegressionEquation},
			[]any{"Интерпретация", r.Interpretation},
			[]any{"", ""},
		)
	}
	for i, row := range rows {
		for j, value := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+1)
			_ = f.SetCellValue(sheet, cell, value)
		}
	}
}

// writeForecastSheet сохраняет прогнозы потребления для заданных температур.
func writeForecastSheet(f *excelize.File, sheet string, forecasts []analysis.Forecast) {
	_ = f.SetCellValue(sheet, "A1", "temperature_c")
	_ = f.SetCellValue(sheet, "B1", "predicted_consumption_kwh")
	for i, item := range forecasts {
		row := i + 2
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), item.TemperatureC)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), item.PredictedConsumptionKWh)
	}
}

// writeChartDataSheet кладет фактические и прогнозные точки, по которым frontend
// строит графики анализа.
func writeChartDataSheet(f *excelize.File, sheet string, points []analysis.ChartPoint) {
	headers := []string{"measured_at", "sensor_code", "temperature_c", "actual_consumption_kwh", "predicted_consumption_kwh"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	for i, p := range points {
		row := i + 2
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), p.MeasuredAt)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), p.SensorCode)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row), p.TemperatureC)
		_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", row), p.ConsumptionKWh)
		_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), p.PredictedConsumptionKWh)
	}
}
