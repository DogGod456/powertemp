package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store - тонкая обертка над пулом PostgreSQL. Остальные пакеты не работают с
// pgx напрямую, а вызывают методы Store.
type Store struct {
	pool *pgxpool.Pool
}

// New создает пул подключений, проверяет доступность базы и возвращает Store.
func New(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

// Close закрывает пул соединений PostgreSQL при остановке приложения.
func (s *Store) Close() {
	s.pool.Close()
}
