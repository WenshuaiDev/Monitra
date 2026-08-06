// Package httpinterface owns Foundation's browser-facing HTTP policy and routes.
package httpinterface

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"log/slog"
	"net/http"
)

const (
	APIMajor            = 1
	startupHandshakeURL = "/api/v1/startup-handshake"
)

type handler struct {
	releaseIdentity string
	logger          *slog.Logger
}

type requestIDContextKey struct{}

type responseEnvelope struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	RequestID string `json:"request_id"`
}

type startupData struct {
	ReleaseIdentity string `json:"release_identity"`
	APIMajor        int    `json:"api_major"`
}

// NewHandler composes the handwritten versioned HTTP routes.
func NewHandler(releaseIdentity string, logger *slog.Logger) http.Handler {
	return &handler{releaseIdentity: releaseIdentity, logger: logger}
}

func (application *handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	request = request.WithContext(context.WithValue(request.Context(), requestIDContextKey{}, rand.Text()))
	requestID := requestIDFrom(request)
	if request.URL.Path != startupHandshakeURL {
		application.writeFailure(response, request, http.StatusNotFound, "FOUNDATION_NOT_FOUND", "resource not found")
		return
	}
	if request.Method != http.MethodGet {
		response.Header().Set("Allow", http.MethodGet)
		application.writeFailure(response, request, http.StatusMethodNotAllowed, "FOUNDATION_METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	application.writeJSON(response, http.StatusOK, responseEnvelope{
		Code:    "FOUNDATION_STARTUP_READY",
		Message: "startup handshake succeeded",
		Data: startupData{
			ReleaseIdentity: application.releaseIdentity,
			APIMajor:        APIMajor,
		},
		RequestID: requestID,
	})
	application.logRequest(request, http.StatusOK)
}

func requestIDFrom(request *http.Request) string {
	requestID, _ := request.Context().Value(requestIDContextKey{}).(string)
	return requestID
}

func (application *handler) writeFailure(
	response http.ResponseWriter,
	request *http.Request,
	status int,
	code string,
	message string,
) {
	requestID := requestIDFrom(request)
	application.writeJSON(response, status, responseEnvelope{
		Code:      code,
		Message:   message,
		Data:      nil,
		RequestID: requestID,
	})
	application.logRequest(request, status)
}

func (application *handler) writeJSON(response http.ResponseWriter, status int, body responseEnvelope) {
	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("X-Request-ID", body.RequestID)
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(body)
}

func (application *handler) logRequest(request *http.Request, status int) {
	application.logger.Info(
		"application request completed",
		"request_id", requestIDFrom(request),
		"method", request.Method,
		"path", request.URL.Path,
		"status", status,
	)
}
