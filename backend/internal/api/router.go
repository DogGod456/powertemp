package api

import (
	"net/http"

	"powertemp/backend/internal/config"
	"powertemp/backend/internal/db"
	"powertemp/backend/internal/realtime"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type API struct {
	store *db.Store
	hub   *realtime.Hub
	cfg   config.Config
}

// NewRouter собирает HTTP-приложение: подключает middleware, health-check,
// REST-маршруты приложения и SSE-канал live-измерений.
func NewRouter(store *db.Store, hub *realtime.Hub, cfg config.Config) http.Handler {
	a := &API{store: store, hub: hub, cfg: cfg}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api", func(r chi.Router) {
		a.registerDashboardRoutes(r)
		a.registerSensorRoutes(r)
		a.registerImportRoutes(r)
		a.registerMeasurementRoutes(r)
		a.registerAnalysisRoutes(r)
		a.registerExportRoutes(r)

		r.Get("/live/events", hub.ServeHTTP)
	})
	return r
}

// registerDashboardRoutes отдает агрегированные показатели для главного экрана.
func (a *API) registerDashboardRoutes(r chi.Router) {
	r.Get("/dashboard", a.dashboard)
}

// registerSensorRoutes управляет жизненным циклом виртуальных датчиков и их
// измерениями.
func (a *API) registerSensorRoutes(r chi.Router) {
	r.Get("/sensors", a.listSensors)
	r.Post("/sensors", a.createSensor)
	r.Patch("/sensors/{id}", a.updateSensor)
	r.Delete("/sensors/{id}", a.deleteSensor)
	r.Post("/sensors/{id}/start", a.startSensor)
	r.Post("/sensors/{id}/stop", a.stopSensor)
	r.Patch("/sensors/{id}/frequency", a.updateSensorFrequency)
	r.Delete("/sensors/{id}/measurements", a.clearSensorMeasurements)
	r.Get("/sensors/{id}/measurements", a.sensorMeasurements)
}

// registerImportRoutes отвечает за загрузку XLSX-источников и их удаление.
func (a *API) registerImportRoutes(r chi.Router) {
	r.Post("/imports", a.importFile)
	r.Get("/imports", a.listImports)
	r.Delete("/imports/{id}", a.deleteImport)
}

// registerMeasurementRoutes предоставляет общую выборку измерений с фильтрами.
func (a *API) registerMeasurementRoutes(r chi.Router) {
	r.Get("/measurements", a.measurements)
}

// registerAnalysisRoutes запускает корреляционно-регрессионный анализ.
func (a *API) registerAnalysisRoutes(r chi.Router) {
	r.Post("/analysis/run", a.runAnalysis)
}

// registerExportRoutes создает, перечисляет и скачивает XLSX-выгрузки.
func (a *API) registerExportRoutes(r chi.Router) {
	r.Post("/exports", a.createExport)
	r.Get("/exports", a.listExports)
	r.Get("/exports/{id}/download", a.downloadExport)
}
