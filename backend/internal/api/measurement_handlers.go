package api

import "net/http"

// measurements возвращает измерения из датчиков и/или импортов по query-фильтрам
// from, to, source_type, sensor_ids, import_file_ids, limit и offset.
func (a *API) measurements(w http.ResponseWriter, r *http.Request) {
	filter := queryFilter(r)
	items, err := a.store.QueryMeasurements(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
