package db

import (
	"strings"

	"powertemp/backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// rowScanner позволяет использовать одни и те же scan-функции для QueryRow и
// rows.Next(), потому что оба типа поддерживают метод Scan.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanSensor преобразует SQL-строку в доменную модель Sensor.
func scanSensor(row rowScanner) (domain.Sensor, error) {
	var s domain.Sensor
	err := row.Scan(&s.ID, &s.Code, &s.Name, &s.IsActive, &s.FrequencySeconds, &s.CollectionStartedAt, &s.LastCollectedAt, &s.CurrentTemperature, &s.CurrentConsumption, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

// scanSensors читает весь набор строк датчиков.
func scanSensors(rows pgx.Rows) ([]domain.Sensor, error) {
	result := make([]domain.Sensor, 0)
	for rows.Next() {
		item, err := scanSensor(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// scanMeasurement восстанавливает nullable UUID-ссылки из текстовых значений,
// которые SQL-запросы отдают через COALESCE.
func scanMeasurement(row rowScanner) (domain.Measurement, error) {
	var m domain.Measurement
	var sensorID string
	var importFileID string
	err := row.Scan(&m.ID, &m.SourceType, &sensorID, &importFileID, &m.SensorCode, &m.MeasuredAt, &m.TemperatureC, &m.ConsumptionKWh, &m.CreatedAt)
	if err != nil {
		return m, err
	}
	if strings.TrimSpace(sensorID) != "" {
		id, parseErr := uuid.Parse(sensorID)
		if parseErr != nil {
			return m, parseErr
		}
		m.SensorID = &id
	}
	if strings.TrimSpace(importFileID) != "" {
		id, parseErr := uuid.Parse(importFileID)
		if parseErr != nil {
			return m, parseErr
		}
		m.ImportFileID = &id
	}
	return m, nil
}

// scanMeasurements читает весь набор строк измерений.
func scanMeasurements(rows pgx.Rows) ([]domain.Measurement, error) {
	result := make([]domain.Measurement, 0)
	for rows.Next() {
		item, err := scanMeasurement(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// scanImportFile преобразует SQL-строку в ImportFile.
func scanImportFile(row rowScanner) (domain.ImportFile, error) {
	var item domain.ImportFile
	err := row.Scan(&item.ID, &item.OriginalFilename, &item.StoredFilename, &item.RowsCount, &item.ImportedAt, &item.Status, &item.ErrorMessage)
	return item, err
}
