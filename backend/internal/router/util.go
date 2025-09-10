package router

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

// decodeJSON reads and unmarshals the request body into T with sane limits and timeouts.
func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var zero T

	// Optional: enforce a context timeout for reading large bodies
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	r = r.WithContext(ctx)

	// Optional: limit max body size to prevent abuse
	const maxBody = int64(1 << 20) // 1 MiB
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req T
	if err := dec.Decode(&req); err != nil {
		// Provide clearer errors
		if errors.Is(err, io.EOF) {
			return zero, errors.New("empty body")
		}
		return zero, err
	}

	// Ensure there’s no trailing garbage
	if dec.More() {
		return zero, errors.New("multiple JSON values in body")
	}

	return req, nil
}

// writeJSON marshals v and writes it with status and proper headers.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	// Optional: for readability in dev
	// enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// writeJSONError writes a simple error response as JSON.
func writeJSONError(w http.ResponseWriter, status int, err error) {
	type errorResp struct {
		Error   string `json:"error"`
		Status  int    `json:"status"`
		Message string `json:"message,omitempty"`
	}
	writeJSON(w, status, errorResp{
		Error:   http.StatusText(status),
		Status:  status,
		Message: err.Error(),
	})
}
