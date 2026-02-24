package observability

import (
	"net/http"
)

func HealthLiveHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func HealthReadyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// In gateway, maybe check connectivity to downstream services?
		// For now just return OK
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}
