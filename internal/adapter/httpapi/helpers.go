package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	domain "github.com/shvadoodi/lecture-order-demo/internal/order"
)

// decodeJSONBody accepts one bounded JSON object and rejects unknown fields.
func decodeJSONBody(r *http.Request, dst any) error {
	const maxBodyBytes = 1 << 20
	// Read one extra byte to distinguish an exact-limit body from an oversized one.
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	if err != nil {
		return errors.New("invalid request body: " + err.Error())
	}
	if len(body) > maxBodyBytes {
		return errors.New("request body must not exceed 1 MiB")
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return errors.New("request body must contain exactly one JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return errors.New("invalid request body: " + err.Error())
	}
	// A second decode must reach EOF; otherwise another JSON value follows.
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("request body must contain exactly one JSON object")
	}
	return nil
}

// orderIDFromPath rejects missing IDs and additional path segments.
func orderIDFromPath(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/orders/"))
	if id == "" || strings.Contains(id, "/") {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "a single order id is required"})
		return "", false
	}
	return id, true
}

// writeJSON sets headers before the status, then encodes the response body.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Once headers are sent we cannot change the status; log encoding failures.
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("write response error: %v", err)
	}
}

// writeOrderError keeps infrastructure errors out of client responses.
func writeOrderError(w http.ResponseWriter, err error) {
	var validationError *domain.ValidationError
	if errors.As(err, &validationError) {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	log.Printf("order operation error: %v", err)
	writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
}
