package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type sensorCreateRequest struct {
	Name             string `json:"name"`
	FrequencySeconds int    `json:"frequency_seconds"`
}

type sensorUpdateRequest struct {
	Name             *string `json:"name"`
	FrequencySeconds *int    `json:"frequency_seconds"`
}

// listSensors возвращает все виртуальные датчики с текущим состоянием и
// последними измерениями.
func (a *API) listSensors(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListSensors(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// createSensor создает новый виртуальный датчик; код D-N назначает слой БД.
func (a *API) createSensor(w http.ResponseWriter, r *http.Request) {
	var req sensorCreateRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	item, err := a.store.CreateSensor(r.Context(), req.Name, req.FrequencySeconds)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

// updateSensor меняет имя и/или частоту сбора данных датчика.
func (a *API) updateSensor(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var req sensorUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := a.store.UpdateSensor(r.Context(), id, req.Name, req.FrequencySeconds)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// updateSensorFrequency - узкий endpoint для изменения только частоты сбора.
func (a *API) updateSensorFrequency(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var req struct {
		FrequencySeconds int `json:"frequency_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := a.store.UpdateSensor(r.Context(), id, nil, &req.FrequencySeconds)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// deleteSensor удаляет датчик и все связанные с ним измерения.
func (a *API) deleteSensor(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := a.store.DeleteSensor(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (a *API) startSensor(w http.ResponseWriter, r *http.Request) {
	a.setSensorActive(w, r, true)
}

func (a *API) stopSensor(w http.ResponseWriter, r *http.Request) {
	a.setSensorActive(w, r, false)
}

// setSensorActive включает или выключает генерацию новых измерений для датчика.
func (a *API) setSensorActive(w http.ResponseWriter, r *http.Request, active bool) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := a.store.SetSensorActive(r.Context(), id, active)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// clearSensorMeasurements очищает историю конкретного датчика и сбрасывает его
// последние показания на карточке.
func (a *API) clearSensorMeasurements(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := a.store.ClearSensorMeasurements(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

// sensorMeasurements применяет общий фильтр измерений, но принудительно
// ограничивает выборку одним датчиком из URL.
func (a *API) sensorMeasurements(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	filter := queryFilter(r)
	filter.SensorIDs = []uuid.UUID{id}
	items, err := a.store.QueryMeasurements(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
