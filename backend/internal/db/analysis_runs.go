package db

import (
	"context"

	"powertemp/backend/internal/domain"

	"github.com/google/uuid"
)

// CreateAnalysisRun сохраняет краткую сводку успешного анализа.
func (s *Store) CreateAnalysisRun(ctx context.Context, run domain.AnalysisRun) (domain.AnalysisRun, error) {
	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO analysis_runs (id, mode, period_from, period_to, points_count, correlation, regression_a, regression_b, r_squared, summary)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		run.ID, run.Mode, run.PeriodFrom, run.PeriodTo, run.PointsCount, run.Correlation, run.RegressionA, run.RegressionB, run.RSquared, run.Summary)
	if err != nil {
		return domain.AnalysisRun{}, err
	}
	return s.GetAnalysisRun(ctx, run.ID)
}

// GetAnalysisRun читает сохраненный результат анализа по UUID.
func (s *Store) GetAnalysisRun(ctx context.Context, id uuid.UUID) (domain.AnalysisRun, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, mode, period_from, period_to, points_count, correlation, regression_a, regression_b, r_squared, summary, created_at FROM analysis_runs WHERE id=$1`, id)
	var run domain.AnalysisRun
	err := row.Scan(&run.ID, &run.Mode, &run.PeriodFrom, &run.PeriodTo, &run.PointsCount, &run.Correlation, &run.RegressionA, &run.RegressionB, &run.RSquared, &run.Summary, &run.CreatedAt)
	return run, err
}
