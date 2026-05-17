package db

import (
	"context"

	"powertemp/backend/internal/domain"

	"github.com/google/uuid"
)

// CreateImportFile создает метаданные успешного XLSX-импорта.
func (s *Store) CreateImportFile(ctx context.Context, original string, rows int) (domain.ImportFile, error) {
	id := uuid.New()
	_, err := s.pool.Exec(ctx, `INSERT INTO import_files (id, original_filename, rows_count, status) VALUES ($1,$2,$3,'success')`, id, original, rows)
	if err != nil {
		return domain.ImportFile{}, err
	}
	return s.GetImportFile(ctx, id)
}

// GetImportFile читает один импорт по UUID.
func (s *Store) GetImportFile(ctx context.Context, id uuid.UUID) (domain.ImportFile, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, original_filename, stored_filename, rows_count, imported_at, status, error_message FROM import_files WHERE id=$1`, id)
	return scanImportFile(row)
}

// ListImportFiles возвращает только XLSX-импорты, которые участвуют в UI.
func (s *Store) ListImportFiles(ctx context.Context) ([]domain.ImportFile, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, original_filename, stored_filename, rows_count, imported_at, status, error_message FROM import_files WHERE lower(original_filename) LIKE '%.xlsx' ORDER BY imported_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.ImportFile, 0)
	for rows.Next() {
		item, err := scanImportFile(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// DeleteImportFile удаляет импорт и связанные с ним измерения через каскадное
// ограничение таблицы measurements.
func (s *Store) DeleteImportFile(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM import_files WHERE id=$1`, id)
	return err
}
