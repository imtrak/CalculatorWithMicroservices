package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer() http.Handler {
	return New(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func post(body string, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, calculationsRoute, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	newTestServer().ServeHTTP(rec, req)
	return rec
}

func TestCalculateSuccess(t *testing.T) {
	tests := []struct{ body, want string }{
		{`{"expression":"(12 + 3) * 4"}`, `{"expression":"(12 + 3) * 4","result":60}`},
		{`{"expression":"0.1 + 0.2"}`, `{"expression":"0.1 + 0.2","result":0.3}`},
		{`{"expression":"10 / 4"}`, `{"expression":"10 / 4","result":2.5}`},
		{`{"expression":"2^10"}`, `{"expression":"2^10","result":1024}`},
		{`{"expression":"sqrt(144)"}`, `{"expression":"sqrt(144)","result":12}`},
		{`{"expression":"50%"}`, `{"expression":"50%","result":0.5}`},
	}
	for _, tt := range tests {
		rec := post(tt.body)
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200; body %s", tt.body, rec.Code, rec.Body)
		}
		if got := strings.TrimSpace(rec.Body.String()); got != tt.want {
			t.Errorf("%s: body = %s, want %s", tt.body, got, tt.want)
		}
	}
}

func TestCalculateErrors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{"division by zero", `{"expression":"1/0"}`, 400, "division_by_zero"},
		{"negative square root", `{"expression":"sqrt(-1)"}`, 400, "negative_square_root"},
		{"invalid power", `{"expression":"(-8)^0.5"}`, 400, "invalid_power"},
		{"unmatched parenthesis", `{"expression":"(1 + 2"}`, 400, "unmatched_parenthesis"},
		{"invalid syntax", `{"expression":"1 + * 2"}`, 400, "invalid_syntax"},
		{"invalid characters", `{"expression":"1; 2"}`, 400, "invalid_characters"},
		{"empty expression", `{"expression":""}`, 400, "empty_expression"},
		{"missing expression", `{}`, 400, "empty_expression"},
		{"malformed json", `{"expression":`, 400, "invalid_request"},
		{"wrong field type", `{"expression":42}`, 400, "invalid_request"},
		{"unknown field", `{"expression":"1","x":1}`, 400, "invalid_request"},
		{"trailing data", `{"expression":"1"}{}`, 400, "invalid_request"},
		{"body too large", `{"expression":"` + strings.Repeat("1", maxBodyBytes) + `"}`, 413, "request_too_large"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := post(tt.body)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body %s", rec.Code, tt.wantStatus, rec.Body)
			}
			if !strings.Contains(rec.Body.String(), `"code":"`+tt.wantCode+`"`) {
				t.Errorf("body = %s, want code %q", rec.Body, tt.wantCode)
			}
		})
	}
}

func TestCalculateRejectsWrongContentType(t *testing.T) {
	for _, contentType := range []string{"text/plain", ""} {
		req := httptest.NewRequest(http.MethodPost, calculationsRoute, strings.NewReader(`{"expression":"1+1"}`))
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		rec := httptest.NewRecorder()
		newTestServer().ServeHTTP(rec, req)

		if rec.Code != http.StatusUnsupportedMediaType || !strings.Contains(rec.Body.String(), `"code":"unsupported_media_type"`) {
			t.Errorf("Content-Type %q: status = %d, body %s; want 415 unsupported_media_type", contentType, rec.Code, rec.Body)
		}
	}
}

func TestCalculateAcceptsContentTypeWithCharset(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, calculationsRoute, strings.NewReader(`{"expression":"1+1"}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rec := httptest.NewRecorder()
	newTestServer().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
}

func TestExpressionIsEchoedAsReceived(t *testing.T) {
	rec := post(`{"expression":"  2 +  3 "}`)
	if got, want := strings.TrimSpace(rec.Body.String()), `{"expression":"  2 +  3 ","result":5}`; got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func TestRequestID(t *testing.T) {
	if got := post(`{"expression":"1"}`, "X-Request-ID", "abc-123").Header().Get("X-Request-ID"); got != "abc-123" {
		t.Errorf("echoed id = %q, want abc-123", got)
	}
	got := post(`{"expression":"1"}`, "X-Request-ID", "bad id!").Header().Get("X-Request-ID")
	if got == "" || got == "bad id!" {
		t.Errorf("id = %q, want a generated one", got)
	}
}

func TestHealthEndpoints(t *testing.T) {
	for _, path := range []string{"/healthz", "/readyz", "/api/healthz", "/api/readyz"} {
		rec := httptest.NewRecorder()
		newTestServer().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, rec.Code)
		}
	}
}

func TestAPIInfo(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestServer().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	var info struct {
		Service   string `json:"service"`
		Version   string `json:"version"`
		Endpoints []struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		} `json:"endpoints"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatalf("invalid JSON %q: %v", rec.Body, err)
	}
	if info.Service != "calculator-service" || info.Version != "v1" {
		t.Errorf("info = %+v", info)
	}
	if len(info.Endpoints) != 1 || info.Endpoints[0].Method != "POST" || info.Endpoints[0].Path != calculationsRoute {
		t.Errorf("endpoints = %+v, want the calculations endpoint", info.Endpoints)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestServer().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, calculationsRoute, nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestPanicIsRecovered(t *testing.T) {
	h := recoverPanics(slog.New(slog.NewTextHandler(io.Discard, nil)), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}
