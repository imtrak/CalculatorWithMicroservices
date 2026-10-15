// Package server exposes the calculator's public REST API.
package server

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"calculator/calculator-service/internal/expression"
)

const (
	maxBodyBytes      = 4 << 10
	apiRoot           = "/api/v1"
	calculationsRoute = apiRoot + "/calculations"
)

type handler struct {
	log *slog.Logger
}

// New returns the root handler with all routes and middleware.
func New(logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	h := &handler{log: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("POST "+calculationsRoute, h.calculate)
	mux.HandleFunc("GET "+apiRoot, h.info)
	// Kubernetes probes use the root paths. The /api copies exist because the
	// Ingress only sends /api/... to this service, so they are the ones a person
	// can reach from outside the cluster.
	for _, path := range []string{"/healthz", "/readyz", "/api/healthz", "/api/readyz"} {
		mux.HandleFunc("GET "+path, h.health)
	}

	return recoverPanics(logger, withRequestID(logRequests(logger, mux)))
}

type calculationRequest struct {
	Expression string `json:"expression"`
}

type calculationResponse struct {
	Expression string  `json:"expression"`
	Result     float64 `json:"result"`
}

func (h *handler) calculate(w http.ResponseWriter, r *http.Request) {
	if mediaType, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";"); strings.TrimSpace(strings.ToLower(mediaType)) != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json.")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req calculationRequest
	if err := dec.Decode(&req); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "Request body is too large.")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_request", "Request body must be a JSON object with a string \"expression\" field.")
		return
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_request", "Request body must contain a single JSON object.")
		return
	}

	result, err := expression.Evaluate(req.Expression)
	if err != nil {
		var exprErr *expression.Error
		if errors.As(err, &exprErr) {
			writeError(w, http.StatusBadRequest, exprErr.Code, exprErr.Message)
			return
		}
		h.log.Error("evaluation failed", "request_id", requestIDFrom(r.Context()), "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Unexpected server error.")
		return
	}

	writeJSON(w, http.StatusOK, calculationResponse{Expression: req.Expression, Result: result})
}

// info describes the API so that opening /api/v1 in a browser is informative.
func (h *handler) info(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, apiInfo{
		Service: "calculator-service",
		Version: "v1",
		Endpoints: []endpointInfo{
			{
				Method:      http.MethodPost,
				Path:        calculationsRoute,
				Description: "Evaluate an arithmetic expression, e.g. {\"expression\": \"(12 + 3) * 4\"}.",
			},
		},
	})
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type apiInfo struct {
	Service   string         `json:"service"`
	Version   string         `json:"version"`
	Endpoints []endpointInfo `json:"endpoints"`
}

type endpointInfo struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorBody{Error: errorDetail{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
