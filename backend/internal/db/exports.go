package db

import (
	"context"
	"strings"

	"powertemp/backend/internal/domain"

	"github.com/google/uuid"
)

// CreateExportFile регистрирует созданный XLSX-файл в истории выгрузок.
func (s *Store) CreateExportFile(ctx context.Context, file domain.ExportFile) (domain.ExportFile, error) {
	if file.ID == uuid.Nil {
		file.ID = uuid.New()
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO export_files (id, analysis_id, filename, format, file_path) VALUES ($1,$2,$3,$4,$5)`, file.ID, file.AnalysisID, file.Filename, file.Format, file.FilePath)
	if err != nil {
		return domain.ExportFile{}, err
	}
	return s.GetExportFile(ctx, file.ID)
}

// GetExportFile читает метаданные выгрузки и восстанавливает nullable analysis_id.
func (s *Store) GetExportFile(ctx context.Context, id uuid.UUID) (domain.ExportFile, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, COALESCE(analysis_id::text, ''), filename, format, file_path, created_at FROM export_files WHERE id=$1`, id)
	var f domain.ExportFile
	var analysisID string
	err := row.Scan(&f.ID, &analysisID, &f.Filename, &f.Format, &f.FilePath, &f.CreatedAt)
	if err != nil {
		return f, err
	}
	if strings.TrimSpace(analysisID) != "" {
		id, parseErr := uuid.Parse(analysisID)
		if parseErr != nil {
			return f, parseErr
		}
		f.AnalysisID = &id
	}
	return f, nil
}

// ListExportFiles возвращает историю XLSX-файлов в обратном хронологическом
// порядке для страницы "Выгрузки".
func (s *Store) ListExportFiles(ctx context.Context) ([]domain.ExportFile, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, COALESCE(analysis_id::text, ''), filename, format, file_path, created_at FROM export_files WHERE format='xlsx' ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.ExportFile, 0)
	for rows.Next() {
		var f domain.ExportFile
		var analysisID string
		if err := rows.Scan(&f.ID, &analysisID, &f.Filename, &f.Format, &f.FilePath, &f.CreatedAt); err != nil {
			return nil, err
		}
		if strings.TrimSpace(analysisID) != "" {
			id, parseErr := uuid.Parse(analysisID)
			if parseErr != nil {
				return nil, parseErr
			}
			f.AnalysisID = &id
		}
		result = append(result, f)
	}
	return result, rows.Err()
}
