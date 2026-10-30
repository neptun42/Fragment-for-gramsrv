package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"lunax/internal/domain"
	"lunax/internal/network/jsonhttp"
	"lunax/internal/service"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "invalid", Message: "bad json: " + err.Error()})
		return false
	}
	return true
}

type apiError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// доменная ошибка -> http статус и короткий код для фронта
func writeErr(w http.ResponseWriter, log *slog.Logger, err error) {
	status, code := http.StatusInternalServerError, "internal"
	var remote *jsonhttp.Error
	switch {
	case errors.Is(err, domain.ErrNoFunds):
		status, code = http.StatusPaymentRequired, "no_funds"
	case errors.Is(err, domain.ErrNotFound):
		status, code = http.StatusNotFound, "not_found"
	case errors.Is(err, domain.ErrInvalid):
		status, code = http.StatusBadRequest, "invalid"
	case errors.Is(err, domain.ErrConflict):
		status, code = http.StatusConflict, "conflict"
	case errors.Is(err, domain.ErrForbidden):
		status, code = http.StatusForbidden, "forbidden"
	case errors.Is(err, domain.ErrUnauthorized):
		status, code = http.StatusUnauthorized, "unauthorized"
	case errors.Is(err, service.ErrTooManyCodes):
		status, code = http.StatusTooManyRequests, "rate_limited"
	case errors.Is(err, domain.ErrServerUnavailable), errors.As(err, &remote):
		status, code = http.StatusBadGateway, "server_unavailable"
	}
	msg := err.Error()
	if status == http.StatusInternalServerError {
		log.Error("request failed", "err", err)
		msg = "internal error"
	}
	writeJSON(w, status, apiError{Error: code, Message: msg})
}

func pathID(r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	return id, err == nil && id > 0
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if t, ok := strings.CutPrefix(h, "Bearer "); ok {
		return strings.TrimSpace(t)
	}
	return ""
}
