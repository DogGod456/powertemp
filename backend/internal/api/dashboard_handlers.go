package api

import "net/http"

// dashboard собирает компактные счетчики и последнее sensor-измерение для
// главной страницы приложения.
func (a *API) dashboard(w http.ResponseWriter, r *http.Request) {
	data, err := a.store.Dashboard(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}
