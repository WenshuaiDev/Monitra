package httpinterface_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/legacy"

	"monitra/internal/foundation/httpinterface"
)

const startupHandshakePath = "/api/v1/startup-handshake"

type startupResponse struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Data      startupData `json:"data"`
	RequestID string      `json:"request_id"`
}

type startupData struct {
	ReleaseIdentity string `json:"release_identity"`
	APIMajor        int    `json:"api_major"`
}

type failureResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	RequestID string `json:"request_id"`
}

func TestStartupHandshakeReturnsReleaseCompatibilityContract(t *testing.T) {
	handler := httpinterface.NewHandler("2026.08.06-test", slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, startupHandshakePath, nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	var body startupResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != "FOUNDATION_STARTUP_READY" || body.Message != "startup handshake succeeded" {
		t.Fatalf("result = (%q, %q), want FOUNDATION_STARTUP_READY startup handshake succeeded", body.Code, body.Message)
	}
	if body.Data.ReleaseIdentity != "2026.08.06-test" || body.Data.APIMajor != 1 {
		t.Fatalf("data = %+v, want release identity 2026.08.06-test and API major 1", body.Data)
	}
	if body.RequestID == "" {
		t.Fatal("request_id is empty")
	}
}

func TestStartupHandshakeUsesOneServerGeneratedRequestID(t *testing.T) {
	var logs bytes.Buffer
	handler := httpinterface.NewHandler("2026.08.06-test", slog.New(slog.NewJSONHandler(&logs, nil)))
	request := httptest.NewRequest(http.MethodGet, startupHandshakePath, nil)
	request.Header.Set("X-Request-ID", "client-selected-id")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	var body startupResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	headerID := response.Header().Get("X-Request-ID")
	if headerID == "" || headerID == "client-selected-id" {
		t.Fatalf("server request ID = %q, want a new server-generated value", headerID)
	}
	if body.RequestID != headerID {
		t.Fatalf("body request_id = %q, want header request ID %q", body.RequestID, headerID)
	}

	var logRecord map[string]any
	if err := json.NewDecoder(&logs).Decode(&logRecord); err != nil {
		t.Fatalf("decode request log: %v", err)
	}
	if got, _ := logRecord["request_id"].(string); got != headerID {
		t.Fatalf("log request_id = %q, want header request ID %q", got, headerID)
	}
	if got, _ := logRecord["msg"].(string); got != "application request completed" {
		t.Fatalf("log message = %q, want application request completed", got)
	}
}

func TestApplicationRoutesReturnFlatFailures(t *testing.T) {
	handler := httpinterface.NewHandler("2026.08.06-test", slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)))
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantCode   string
		wantAllow  string
	}{
		{
			name:       "unsupported method",
			method:     http.MethodPost,
			path:       startupHandshakePath,
			wantStatus: http.StatusMethodNotAllowed,
			wantCode:   "FOUNDATION_METHOD_NOT_ALLOWED",
			wantAllow:  http.MethodGet,
		},
		{
			name:       "unknown path",
			method:     http.MethodGet,
			path:       "/api/v1/unknown",
			wantStatus: http.StatusNotFound,
			wantCode:   "FOUNDATION_NOT_FOUND",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if got := response.Header().Get("Allow"); got != test.wantAllow {
				t.Fatalf("Allow = %q, want %q", got, test.wantAllow)
			}
			var body failureResponse
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Code != test.wantCode {
				t.Fatalf("code = %q, want %q", body.Code, test.wantCode)
			}
			if body.Data != nil {
				t.Fatalf("data = %#v, want null", body.Data)
			}
			if body.RequestID == "" || body.RequestID != response.Header().Get("X-Request-ID") {
				t.Fatalf("body request_id = %q, header request ID = %q", body.RequestID, response.Header().Get("X-Request-ID"))
			}
		})
	}
}

func TestStartupHandshakeConformsToOpenAPI(t *testing.T) {
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromFile(filepath.Join("..", "..", "..", "api", "openapi.yaml"))
	if err != nil {
		t.Fatalf("load OpenAPI document: %v", err)
	}
	if err := document.Validate(context.Background()); err != nil {
		t.Fatalf("validate OpenAPI document: %v", err)
	}
	router, err := legacy.NewRouter(document)
	if err != nil {
		t.Fatalf("create OpenAPI router: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, startupHandshakePath, nil)
	route, pathParameters, err := router.FindRoute(request)
	if err != nil {
		t.Fatalf("find OpenAPI route: %v", err)
	}
	response := httptest.NewRecorder()
	handler := httpinterface.NewHandler("2026.08.06-test", slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)))
	handler.ServeHTTP(response, request)

	validation := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    request,
			PathParams: pathParameters,
			Route:      route,
		},
		Status: response.Code,
		Header: response.Header(),
	}
	validation.SetBodyBytes(response.Body.Bytes())
	if err := openapi3filter.ValidateResponse(context.Background(), validation); err != nil {
		t.Fatalf("response violates OpenAPI: %v", err)
	}
}

func TestOpenAPIFailureContractRequiresNullData(t *testing.T) {
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromFile(filepath.Join("..", "..", "..", "api", "openapi.yaml"))
	if err != nil {
		t.Fatalf("load OpenAPI document: %v", err)
	}
	if err := document.Validate(context.Background()); err != nil {
		t.Fatalf("validate OpenAPI document: %v", err)
	}
	schema := document.Components.Schemas["ErrorResponse"].Value

	failure := map[string]any{
		"code":       "FOUNDATION_NOT_FOUND",
		"message":    "resource not found",
		"data":       nil,
		"request_id": "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
	}
	if err := document.ValidateSchemaJSON(schema, failure); err != nil {
		t.Fatalf("null failure data violates OpenAPI: %v", err)
	}
	failure["data"] = map[string]any{}
	if err := document.ValidateSchemaJSON(schema, failure); err == nil {
		t.Fatal("OpenAPI accepted object failure data, want only null")
	}
}
