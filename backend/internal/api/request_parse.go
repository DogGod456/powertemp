package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"powertemp/backend/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// queryFilter переводит query string HTTP-запроса в доменный фильтр измерений,
// который затем используется SQL-слоем.
func queryFilter(r *http.Request) domain.MeasurementFilter {
	q := r.URL.Query()
	filter := domain.MeasurementFilter{}
	filter.SourceType = q.Get("source_type")
	filter.SensorIDs, _ = parseUUIDList(splitCommaList(q.Get("sensor_ids")))
	filter.ImportFileIDs, _ = parseUUIDList(splitCommaList(q.Get("import_file_ids")))
	if from, err := parseQueryTime(q.Get("from")); err == nil && !from.IsZero() {
		filter.From = &from
	}
	if to, err := parseQueryTime(q.Get("to")); err == nil && !to.IsZero() {
		filter.To = &to
	}
	filter.Limit, _ = strconv.Atoi(q.Get("limit"))
	filter.Offset, _ = strconv.Atoi(q.Get("offset"))
	if filter.Limit == 0 {
		filter.Limit = 2000
	}
	return filter
}

// parseQueryTime поддерживает RFC3339 из frontend и простой формат с пробелом,
// удобный для ручных запросов к API.
func parseQueryTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, errors.New("invalid time")
}

// parseUUIDParam извлекает UUID из path-параметров chi.
func parseUUIDParam(r *http.Request, name string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, name))
}

// parseUUIDList превращает список строк в UUID и возвращает понятную ошибку,
// если пользователь передал некорректный идентификатор.
func parseUUIDList(values []string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("некорректный UUID: %s", value)
		}
		result = append(result, id)
	}
	return result, nil
}

// splitCommaList разбирает query-параметры вида id1,id2,id3.
func splitCommaList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
