package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// EnsureSchema создает таблицы и индексы приложения, если их еще нет. Это
// позволяет запускать учебный проект без отдельной системы миграций.
func (s *Store) EnsureSchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS sensors (
			id UUID PRIMARY KEY,
			code TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT FALSE,
			frequency_seconds INT NOT NULL DEFAULT 1 CHECK (frequency_seconds >= 1 AND frequency_seconds <= 3600),
			collection_started_at TIMESTAMPTZ,
			last_collected_at TIMESTAMPTZ,
			current_temperature DOUBLE PRECISION,
			current_consumption DOUBLE PRECISION,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS import_files (
			id UUID PRIMARY KEY,
			original_filename TEXT NOT NULL,
			stored_filename TEXT,
			rows_count INT NOT NULL DEFAULT 0,
			imported_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			status TEXT NOT NULL,
			error_message TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS measurements (
			id BIGSERIAL PRIMARY KEY,
			source_type TEXT NOT NULL CHECK (source_type IN ('sensor', 'file')),
			sensor_id UUID REFERENCES sensors(id) ON DELETE CASCADE,
			import_file_id UUID REFERENCES import_files(id) ON DELETE CASCADE,
			sensor_code TEXT NOT NULL,
			measured_at TIMESTAMPTZ NOT NULL,
			temperature_c DOUBLE PRECISION NOT NULL,
			consumption_kwh DOUBLE PRECISION NOT NULL CHECK (consumption_kwh >= 0),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_measurements_sensor_time ON measurements(sensor_id, measured_at);`,
		`CREATE INDEX IF NOT EXISTS idx_measurements_import_time ON measurements(import_file_id, measured_at);`,
		`CREATE INDEX IF NOT EXISTS idx_measurements_source_time ON measurements(source_type, measured_at);`,
		`CREATE TABLE IF NOT EXISTS analysis_runs (
			id UUID PRIMARY KEY,
			mode TEXT NOT NULL,
			period_from TIMESTAMPTZ NOT NULL,
			period_to TIMESTAMPTZ NOT NULL,
			points_count INT NOT NULL,
			correlation DOUBLE PRECISION,
			regression_a DOUBLE PRECISION,
			regression_b DOUBLE PRECISION,
			r_squared DOUBLE PRECISION,
			summary TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS export_files (
			id UUID PRIMARY KEY,
			analysis_id UUID REFERENCES analysis_runs(id) ON DELETE SET NULL,
			filename TEXT NOT NULL,
			format TEXT NOT NULL,
			file_path TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
	}
	for _, q := range queries {
		if _, err := s.pool.Exec(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

// SeedDefaultSensors добавляет демонстрационные D-1...D-5 только в пустую базу.
func (s *Store) SeedDefaultSensors(ctx context.Context) error {
	var count int
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM sensors`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	for i := 1; i <= 5; i++ {
		id := uuid.New()
		code := fmt.Sprintf("D-%d", i)
		name := code
		_, err := s.pool.Exec(ctx, `INSERT INTO sensors (id, code, name, frequency_seconds) VALUES ($1,$2,$3,$4)`, id, code, name, 1)
		if err != nil {
			return err
		}
	}
	return nil
}
