package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"gorm.io/gorm"
)

type ReadinessCheck func(context.Context) error

type statusResponse struct {
	Status string `json:"status"`
}

type readinessResponse struct {
	Status string   `json:"status"`
	Errors []string `json:"errors,omitempty"`
}

func HealthzHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, statusResponse{Status: "ok"})
	}
}

func ReadyzHandler(checks ...ReadinessCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var errs []string
		for _, check := range checks {
			if check == nil {
				continue
			}
			if err := check(ctx); err != nil {
				errs = append(errs, err.Error())
			}
		}

		if len(errs) > 0 {
			writeJSON(w, http.StatusServiceUnavailable, readinessResponse{
				Status: "not_ready",
				Errors: errs,
			})
			return
		}

		writeJSON(w, http.StatusOK, readinessResponse{Status: "ok"})
	}
}

func DBReadinessCheck(db *gorm.DB) ReadinessCheck {
	return func(ctx context.Context) error {
		if db == nil {
			return errors.New("database is not configured")
		}

		sqlDB, err := db.DB()
		if err != nil {
			return err
		}

		return sqlDB.PingContext(ctx)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
