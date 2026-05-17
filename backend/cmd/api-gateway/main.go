// API Gateway - основной backend-процесс PowerTemp.
// Он поднимает HTTP API, подключается к PostgreSQL, запускает сборщик
// виртуальных измерений и публикует новые точки в live-канал SSE.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"powertemp/backend/internal/api"
	"powertemp/backend/internal/collector"
	"powertemp/backend/internal/config"
	"powertemp/backend/internal/db"
	"powertemp/backend/internal/realtime"
)

func main() {
	// Конфигурация читается из переменных окружения, а при локальном запуске
	// использует значения по умолчанию из internal/config.
	cfg := config.Load()

	// Общий контекст приложения закрывается по Ctrl+C / SIGINT и передается
	// всем фоновым процессам, чтобы они завершались вместе с сервером.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Store инкапсулирует доступ к PostgreSQL; при старте также создаются
	// таблицы и пять демонстрационных датчиков D-1...D-5.
	store, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer store.Close()

	if err := store.EnsureSchema(ctx); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}
	if err := store.SeedDefaultSensors(ctx); err != nil {
		log.Fatalf("seed sensors: %v", err)
	}

	// Hub держит открытые SSE-подключения фронтенда и рассылает новые
	// измерения сразу после записи в базу.
	hub := realtime.NewHub()
	go hub.Run(ctx)

	// Collector периодически спрашивает сервис sensor-simulator о новых
	// значениях для активных датчиков и сохраняет их в PostgreSQL.
	col := collector.New(store, cfg.SimulatorURL, hub)
	go col.Start(ctx)

	// Router связывает слой HTTP с хранилищем, realtime-hub и конфигурацией.
	router := api.NewRouter(store, hub, cfg)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("PowerTemp API Gateway listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// После остановки контекста даем серверу до 10 секунд на мягкое закрытие
	// текущих HTTP-запросов.
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
