package health

import (
	"encoding/json"
	"net/http"
	"log/slog"
)

type Api struct {
}

func NewApi() *Api {
	return &Api{}
}

func (api *Api) RegisterHandlers() {
	http.HandleFunc("/health", api.GetHealth)
}

func (api *Api) GetHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	select {
	case <-ctx.Done():
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "shutting down",
		})
		return
	default:
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	}); err != nil {
		slog.Error("encoding health response failed", "err", err)
	}
}
