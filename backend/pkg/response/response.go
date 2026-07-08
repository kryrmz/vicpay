// Package response provides a uniform JSON envelope for the HTTP API so every
// endpoint returns the same shape: {"data": ...} on success or
// {"error": {"code","message"}} on failure.
package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Envelope is the top-level JSON response shape.
type Envelope struct {
	Data  any    `json:"data,omitempty"`
	Error *Error `json:"error,omitempty"`
}

// Error is a machine-readable error with a stable code and a human message.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes data as a success envelope with the given status.
func JSON(w http.ResponseWriter, status int, data any) {
	write(w, status, Envelope{Data: data})
}

// Fail writes an error envelope with a stable code and message.
func Fail(w http.ResponseWriter, status int, code, message string) {
	write(w, status, Envelope{Error: &Error{Code: code, Message: message}})
}

func write(w http.ResponseWriter, status int, env Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(env); err != nil {
		slog.Error("response: encode failed", "err", err)
	}
}

// Decode reads a JSON body into dst, rejecting unknown fields and oversized
// bodies. It returns false and writes a 400 on failure.
func Decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		Fail(w, http.StatusBadRequest, "invalid_body", "the request body could not be parsed")
		return false
	}
	return true
}
