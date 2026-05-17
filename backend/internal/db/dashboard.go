package db

import "context"

// Dashboard собирает легкие агрегаты для стартовой страницы: число измерений,
// активных датчиков, импортов, выгрузок и последнее sensor-измерение.
func (s *Store) Dashboard(ctx context.Context) (map[string]any, error) {
	result := map[string]any{}
	var total int64
	var active int64
	var imports int64
	var exports int64
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM measurements WHERE source_type='sensor'`).Scan(&total); err != nil {
		return nil, err
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM sensors WHERE is_active=true`).Scan(&active); err != nil {
		return nil, err
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM import_files WHERE lower(original_filename) LIKE '%.xlsx'`).Scan(&imports); err != nil {
		return nil, err
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM export_files WHERE format='xlsx'`).Scan(&exports); err != nil {
		return nil, err
	}
	result["measurements_count"] = total
	result["active_sensors"] = active
	result["imports_count"] = imports
	result["exports_count"] = exports

	last, err := scanMeasurement(s.pool.QueryRow(ctx, `SELECT id, source_type, COALESCE(sensor_id::text, ''), COALESCE(import_file_id::text, ''), sensor_code, measured_at, temperature_c, consumption_kwh, created_at FROM measurements WHERE source_type='sensor' ORDER BY measured_at DESC LIMIT 1`))
	if err == nil {
		result["last_measurement"] = last
	}
	return result, nil
}
