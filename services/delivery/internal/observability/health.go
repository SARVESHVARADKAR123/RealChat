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
		// Delivery ready check (could check Redis/Kafka connectivity)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}
