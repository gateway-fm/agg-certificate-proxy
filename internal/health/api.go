package health

import (
	"encoding/json"
	"net/http"
)

type Api struct {
	statusService *Service
}

func NewApi(statusService *Service) *Api {
	return &Api{
		statusService: statusService,
	}
}

func (api *Api) RegisterHandlers() {
	http.HandleFunc("/health", api.GetHealth)
}

func (api *Api) GetHealth(w http.ResponseWriter, r *http.Request) {
	if api.statusService.IsShuttingDown() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "shutting down",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}
