package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func parseAuthID(r *http.Request, header string) (int64, error) {
	val := strings.TrimSpace(r.Header.Get(header))
	if val == "" {
		return 0, fmt.Errorf("missing %s", header)
	}
	id, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", header)
	}
	return id, nil
}

func parsePaging(r *http.Request) (int, int, error) {
	limit := 50
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		l, err := strconv.Atoi(v)
		if err != nil || l <= 0 {
			return 0, 0, fmt.Errorf("invalid limit")
		}
		limit = l
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		o, err := strconv.Atoi(v)
		if err != nil || o < 0 {
			return 0, 0, fmt.Errorf("invalid offset")
		}
		offset = o
	}
	return limit, offset, nil
}

func nullableString(src *string) sql.NullString {
	if src == nil {
		return sql.NullString{}
	}
	trimmed := strings.TrimSpace(*src)
	if trimmed == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: trimmed, Valid: true}
}

func nullToPtr(ns sql.NullString) *string {
	if ns.Valid {
		val := ns.String
		return &val
	}
	return nil
}

func nullInt64ToPtr(ni sql.NullInt64) *int64 {
	if ni.Valid {
		val := ni.Int64
		return &val
	}
	return nil
}

func nullFloat64ToPtr(nf sql.NullFloat64) *float64 {
	if nf.Valid {
		val := nf.Float64
		return &val
	}
	return nil
}

func nullBoolToPtr(nb sql.NullBool) *bool {
	if nb.Valid {
		val := nb.Bool
		return &val
	}
	return nil
}

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if nt.Valid {
		val := nt.Time
		return &val
	}
	return nil
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func contextWithTimeout(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 5*time.Second)
}
