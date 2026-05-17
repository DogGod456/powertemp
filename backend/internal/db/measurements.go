package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"powertemp/backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// InsertMeasurement сохраняет одну точку измерения и возвращает запись с id и
// created_at, которые назначила база.
func (s *Store) InsertMeasurement(ctx context.Context, m domain.Measurement) (domain.Measurement, error) {
	row := s.pool.QueryRow(ctx, `INSERT INTO measurements (source_type, sensor_id, import_file_id, sensor_code, measured_at, temperature_c, consumption_kwh)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, source_type, COALESCE(sensor_id::text, ''), COALESCE(import_file_id::text, ''), sensor_code, measured_at, temperature_c, consumption_kwh, created_at`,
		m.SourceType, m.SensorID, m.ImportFileID, m.SensorCode, m.MeasuredAt, m.TemperatureC, m.ConsumptionKWh)
	return scanMeasurement(row)
}

// InsertMeasurements пакетно записывает строки импортированного файла в одной
// транзакции: либо сохраняется весь валидный импорт, либо ничего.
func (s *Store) InsertMeasurements(ctx context.Context, measurements []domain.Measurement) error {
	if len(measurements) == 0 {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	batch := &pgx.Batch{}
	for _, m := range measurements {
		batch.Queue(`INSERT INTO measurements (source_type, sensor_id, import_file_id, sensor_code, measured_at, temperature_c, consumption_kwh) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			m.SourceType, m.SensorID, m.ImportFileID, m.SensorCode, m.MeasuredAt, m.TemperatureC, m.ConsumptionKWh)
	}
	br := tx.SendBatch(ctx, batch)
	if err := br.Close(); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// UpdateSensorReading обновляет последние значения датчика после успешной
// записи нового измерения.
func (s *Store) UpdateSensorReading(ctx context.Context, id uuid.UUID, temperature float64, consumption float64, measuredAt time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE sensors SET current_temperature=$1, current_consumption=$2, last_collected_at=$3, updated_at=NOW() WHERE id=$4`, temperature, consumption, measuredAt, id)
	return err
}

// QueryMeasurements собирает SQL-запрос по фильтрам источников, периода и
// пагинации. Значения передаются параметрами, а не вставляются в SQL строкой.
func (s *Store) QueryMeasurements(ctx context.Context, filter domain.MeasurementFilter) ([]domain.Measurement, error) {
	query := `SELECT id, source_type, COALESCE(sensor_id::text, ''), COALESCE(import_file_id::text, ''), sensor_code, measured_at, temperature_c, consumption_kwh, created_at FROM measurements`
	where := []string{}
	args := []any{}
	idx := 1

	if filter.SourceType != "" {
		where = append(where, fmt.Sprintf("source_type=$%d", idx))
		args = append(args, filter.SourceType)
		idx++
	}
	sourceParts := []string{}
	if len(filter.SensorIDs) > 0 {
		placeholders := []string{}
		for _, id := range filter.SensorIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
			args = append(args, id)
			idx++
		}
		sourceParts = append(sourceParts, fmt.Sprintf("sensor_id IN (%s)", strings.Join(placeholders, ",")))
	}
	if len(filter.ImportFileIDs) > 0 {
		placeholders := []string{}
		for _, id := range filter.ImportFileIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
			args = append(args, id)
			idx++
		}
		sourceParts = append(sourceParts, fmt.Sprintf("import_file_id IN (%s)", strings.Join(placeholders, ",")))
	}
	if len(sourceParts) > 0 {
		where = append(where, "("+strings.Join(sourceParts, " OR ")+")")
	}
	if filter.From != nil {
		where = append(where, fmt.Sprintf("measured_at >= $%d", idx))
		args = append(args, *filter.From)
		idx++
	}
	if filter.To != nil {
		where = append(where, fmt.Sprintf("measured_at <= $%d", idx))
		args = append(args, *filter.To)
		idx++
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY measured_at ASC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", idx)
		args = append(args, filter.Limit)
		idx++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", idx)
		args = append(args, filter.Offset)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMeasurements(rows)
}
