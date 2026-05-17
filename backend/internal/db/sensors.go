package db

import (
	"context"
	"fmt"
	"strings"

	"powertemp/backend/internal/domain"

	"github.com/google/uuid"
)

// ListSensors возвращает все датчики в стабильном порядке по коду.
func (s *Store) ListSensors(ctx context.Context) ([]domain.Sensor, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, code, name, is_active, frequency_seconds, collection_started_at, last_collected_at, current_temperature, current_consumption, created_at, updated_at FROM sensors ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSensors(rows)
}

// GetSensor читает один датчик по UUID.
func (s *Store) GetSensor(ctx context.Context, id uuid.UUID) (domain.Sensor, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, code, name, is_active, frequency_seconds, collection_started_at, last_collected_at, current_temperature, current_consumption, created_at, updated_at FROM sensors WHERE id=$1`, id)
	return scanSensor(row)
}

// GetActiveSensors выбирает датчики, для которых collector должен получать
// новые измерения.
func (s *Store) GetActiveSensors(ctx context.Context) ([]domain.Sensor, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, code, name, is_active, frequency_seconds, collection_started_at, last_collected_at, current_temperature, current_consumption, created_at, updated_at FROM sensors WHERE is_active=true ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSensors(rows)
}

// CreateSensor нормализует частоту, назначает следующий код D-N и создает
// новый выключенный датчик.
func (s *Store) CreateSensor(ctx context.Context, name string, frequency int) (domain.Sensor, error) {
	if frequency < 1 {
		frequency = 1
	}
	if frequency > 3600 {
		frequency = 3600
	}
	code, err := s.nextSensorCode(ctx)
	if err != nil {
		return domain.Sensor{}, err
	}
	if strings.TrimSpace(name) == "" {
		name = code
	}
	id := uuid.New()
	_, err = s.pool.Exec(ctx, `INSERT INTO sensors (id, code, name, frequency_seconds) VALUES ($1,$2,$3,$4)`, id, code, name, frequency)
	if err != nil {
		return domain.Sensor{}, err
	}
	return s.GetSensor(ctx, id)
}

// nextSensorCode ищет максимальный номер среди кодов D-N и возвращает следующий.
func (s *Store) nextSensorCode(ctx context.Context) (string, error) {
	var maxCode string
	_ = s.pool.QueryRow(ctx, `SELECT code FROM sensors WHERE code LIKE 'D-%' ORDER BY LENGTH(code) DESC, code DESC LIMIT 1`).Scan(&maxCode)
	maxN := 0
	rows, err := s.pool.Query(ctx, `SELECT code FROM sensors WHERE code LIKE 'D-%'`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return "", err
		}
		var n int
		_, _ = fmt.Sscanf(code, "D-%d", &n)
		if n > maxN {
			maxN = n
		}
	}
	return fmt.Sprintf("D-%d", maxN+1), nil
}

// UpdateSensor меняет только переданные поля. Частота дополнительно зажимается
// в диапазон, разрешенный CHECK-ограничением таблицы.
func (s *Store) UpdateSensor(ctx context.Context, id uuid.UUID, name *string, frequency *int) (domain.Sensor, error) {
	parts := []string{}
	args := []any{}
	idx := 1
	if name != nil {
		parts = append(parts, fmt.Sprintf("name=$%d", idx))
		args = append(args, strings.TrimSpace(*name))
		idx++
	}
	if frequency != nil {
		value := *frequency
		if value < 1 {
			value = 1
		}
		if value > 3600 {
			value = 3600
		}
		parts = append(parts, fmt.Sprintf("frequency_seconds=$%d", idx))
		args = append(args, value)
		idx++
	}
	if len(parts) == 0 {
		return s.GetSensor(ctx, id)
	}
	parts = append(parts, "updated_at=NOW()")
	args = append(args, id)
	query := fmt.Sprintf("UPDATE sensors SET %s WHERE id=$%d", strings.Join(parts, ", "), idx)
	_, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return domain.Sensor{}, err
	}
	return s.GetSensor(ctx, id)
}

// SetSensorActive включает или выключает сбор данных; при первом включении
// запоминается время старта коллекции.
func (s *Store) SetSensorActive(ctx context.Context, id uuid.UUID, active bool) (domain.Sensor, error) {
	if active {
		_, err := s.pool.Exec(ctx, `UPDATE sensors SET is_active=true, collection_started_at=COALESCE(collection_started_at, NOW()), updated_at=NOW() WHERE id=$1`, id)
		if err != nil {
			return domain.Sensor{}, err
		}
	} else {
		_, err := s.pool.Exec(ctx, `UPDATE sensors SET is_active=false, updated_at=NOW() WHERE id=$1`, id)
		if err != nil {
			return domain.Sensor{}, err
		}
	}
	return s.GetSensor(ctx, id)
}

// DeleteSensor удаляет датчик; его измерения удаляются каскадно.
func (s *Store) DeleteSensor(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sensors WHERE id=$1`, id)
	return err
}

// ClearSensorMeasurements удаляет историю датчика и очищает последние значения
// в одной транзакции.
func (s *Store) ClearSensorMeasurements(ctx context.Context, id uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM measurements WHERE sensor_id=$1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE sensors SET current_temperature=NULL, current_consumption=NULL, last_collected_at=NULL, updated_at=NOW() WHERE id=$1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
