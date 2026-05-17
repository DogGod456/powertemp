package importutil

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"powertemp/backend/internal/domain"

	"github.com/xuri/excelize/v2"
)

// RequiredHeaders задает контракт XLSX-импорта: без этих столбцов файл не
// принимается и измерения не попадают в базу.
var RequiredHeaders = []string{"measured_at", "sensor_code", "temperature_c", "consumption_kwh"}

// Result возвращает либо готовые к сохранению измерения, либо список ошибок
// валидации с привязкой к строкам файла.
type Result struct {
	Measurements []domain.Measurement `json:"measurements"`
	Errors       []string             `json:"errors"`
}

// ParseUploadedFile выбирает парсер по расширению файла. Сейчас поддерживается
// XLSX, что соответствует формату загрузки в интерфейсе.
func ParseUploadedFile(file multipart.File, filename string) (Result, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".xlsx":
		return parseXLSX(file)
	default:
		return Result{}, fmt.Errorf("неподдерживаемый формат файла: %s. Используйте XLSX", ext)
	}
}

// parseXLSX открывает workbook, берет первый лист и передает строки в общий
// валидатор табличных данных.
func parseXLSX(r io.Reader) (Result, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return Result{}, fmt.Errorf("ошибка чтения XLSX: %w", err)
	}
	defer func() { _ = f.Close() }()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return Result{}, errors.New("XLSX не содержит листов")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return Result{}, fmt.Errorf("ошибка чтения первого листа XLSX: %w", err)
	}
	return parseRows(rows), nil
}

// parseRows проверяет заголовки, типы, диапазоны значений и собирает только
// валидные строки в domain.Measurement. При ошибках импорт отклоняется целиком.
func parseRows(rows [][]string) Result {
	var result Result
	if len(rows) == 0 {
		result.Errors = append(result.Errors, "Файл пустой")
		return result
	}

	headers := normalizeHeaders(rows[0])
	indexes := map[string]int{}
	for i, h := range headers {
		indexes[h] = i
	}
	for _, required := range RequiredHeaders {
		if _, ok := indexes[required]; !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("Нет обязательного столбца %q", required))
		}
	}
	if len(result.Errors) > 0 {
		return result
	}

	for i := 1; i < len(rows); i++ {
		rowNum := i + 1
		row := rows[i]
		if isEmptyRow(row) {
			continue
		}
		measuredRaw := cell(row, indexes["measured_at"])
		sensorCode := strings.TrimSpace(cell(row, indexes["sensor_code"]))
		temperatureRaw := cell(row, indexes["temperature_c"])
		consumptionRaw := cell(row, indexes["consumption_kwh"])

		measuredAt, err := parseTime(measuredRaw)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Строка %d: measured_at имеет неверный формат", rowNum))
			continue
		}
		if sensorCode == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Строка %d: sensor_code не должен быть пустым", rowNum))
			continue
		}
		temperature, err := parseFloat(temperatureRaw)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Строка %d: temperature_c должно быть числом", rowNum))
			continue
		}
		if temperature < -80 || temperature > 80 {
			result.Errors = append(result.Errors, fmt.Sprintf("Строка %d: temperature_c выходит за реалистичный диапазон -80..80", rowNum))
			continue
		}
		consumption, err := parseFloat(consumptionRaw)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Строка %d: consumption_kwh должно быть числом", rowNum))
			continue
		}
		if consumption < 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("Строка %d: consumption_kwh не может быть отрицательным", rowNum))
			continue
		}

		result.Measurements = append(result.Measurements, domain.Measurement{
			SourceType:     "file",
			SensorCode:     sensorCode,
			MeasuredAt:     measuredAt,
			TemperatureC:   temperature,
			ConsumptionKWh: consumption,
		})
	}
	if len(result.Measurements) == 0 && len(result.Errors) == 0 {
		result.Errors = append(result.Errors, "В файле нет строк с данными")
	}
	return result
}

// normalizeHeaders приводит строку заголовков к виду, удобному для сравнения,
// и убирает BOM, который иногда появляется в выгрузках из Excel.
func normalizeHeaders(row []string) []string {
	result := make([]string, len(row))
	for i, v := range row {
		v = strings.TrimSpace(strings.ToLower(v))
		v = strings.TrimPrefix(v, "\ufeff")
		result[i] = v
	}
	return result
}

// isEmptyRow позволяет пропускать полностью пустые строки в конце или середине
// Excel-листа.
func isEmptyRow(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// cell безопасно читает ячейку по индексу: короткие строки не приводят к panic,
// а считаются пустым значением.
func cell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// parseFloat принимает числа с пробелами и десятичной запятой, чтобы импорт был
// устойчив к распространенным русскоязычным Excel-форматам.
func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", ".")
	return strconv.ParseFloat(s, 64)
}

// parseTime поддерживает несколько форматов дат из README и приводит результат
// к UTC для единообразного хранения в PostgreSQL.
func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty time")
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"02.01.2006 15:04:05",
		"02.01.2006 15:04",
		"2006-01-02",
		"02.01.2006",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, errors.New("unsupported time format")
}
