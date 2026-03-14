package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// MockRoundTripper is a mock HTTP round tripper for testing
type MockRoundTripper struct {
	mu          sync.RWMutex
	responses   map[string]*http.Response
	defaultResp *http.Response
	err         error
}

// NewMockRoundTripper creates a new mock round tripper
func NewMockRoundTripper() *MockRoundTripper {
	return &MockRoundTripper{
		responses: make(map[string]*http.Response),
	}
}

// Reset clears all mock responses
func (m *MockRoundTripper) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = make(map[string]*http.Response)
	m.defaultResp = nil
	m.err = nil
}

// SetDefaultResponse sets a default response for all requests
func (m *MockRoundTripper) SetDefaultResponse(resp *http.Response) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultResp = resp
}

// SetError sets a fake error to test error handling
func (m *MockRoundTripper) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// ClearError clears any set error
func (m *MockRoundTripper) ClearError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = nil
}

// AddResponse adds a specific response for a URL
func (m *MockRoundTripper) AddResponse(url string, resp *http.Response) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[url] = resp
}

// RoundTrip implements http.RoundTripper
func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.err != nil {
		return nil, m.err
	}

	// Check for specific URL response
	if resp, ok := m.responses[req.URL.String()]; ok {
		return resp, nil
	}

	// Return default response
	if m.defaultResp != nil {
		return m.defaultResp, nil
	}

	// Return 404 if no response set
	return &http.Response{
		Status:        "404 Not Found",
		StatusCode:    404,
		Body:          http.NoBody,
		Header:        make(http.Header),
		Request:       req,
		Proto:         req.Proto,
		ProtoMajor:    req.ProtoMajor,
		ProtoMinor:    req.ProtoMinor,
		ContentLength: -1,
	}, nil
}

// MockHTTPHandler is a mock HTTP handler for testing API endpoints
type MockHTTPHandler struct {
	mu        sync.RWMutex
	requests  []MockHTTPRequest
	responses []MockHTTPResponse
	err       error
}

// MockHTTPRequest represents a recorded HTTP request
type MockHTTPRequest struct {
	Method    string
	URL       string
	Headers   map[string]string
	Body      string
	Timestamp int64
}

// MockHTTPResponse represents a recorded HTTP response
type MockHTTPResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

// NewMockHTTPHandler creates a new mock HTTP handler
func NewMockHTTPHandler() *MockHTTPHandler {
	return &MockHTTPHandler{
		requests:  make([]MockHTTPRequest, 0),
		responses: make([]MockHTTPResponse, 0),
	}
}

// Reset clears all recorded data
func (m *MockHTTPHandler) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = make([]MockHTTPRequest, 0)
	m.responses = make([]MockHTTPResponse, 0)
	m.err = nil
}

// ServeHTTP implements http.Handler
func (m *MockHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.err != nil {
		http.Error(w, m.err.Error(), http.StatusInternalServerError)
		return
	}

	// Record the request
	req := MockHTTPRequest{
		Method:    r.Method,
		URL:       r.URL.String(),
		Headers:   make(map[string]string),
		Timestamp: time.Now().Unix(),
	}

	// Copy headers
	for k, v := range r.Header {
		req.Headers[k] = v[0]
	}

	// Read body
	body, _ := io.ReadAll(r.Body)
	req.Body = string(body)

	m.requests = append(m.requests, req)

	// Return default success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "ok"}`)

	// Record the response
	resp := MockHTTPResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
		Headers:    make(map[string]string),
	}
	m.responses = append(m.responses, resp)
}

// GetRequests returns all recorded requests
func (m *MockHTTPHandler) GetRequests() []MockHTTPRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]MockHTTPRequest{}, m.requests...)
}

// GetResponses returns all recorded responses
func (m *MockHTTPHandler) GetResponses() []MockHTTPResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]MockHTTPResponse{}, m.responses...)
}

// SetError sets a fake error to test error handling
func (m *MockHTTPHandler) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// ClearError clears any set error
func (m *MockHTTPHandler) ClearError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = nil
}

// MockResponse creates an HTTP response with JSON body
func MockResponse(statusCode int, data interface{}) (*http.Response, error) {
	var body io.Reader

	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(jsonData)
	} else {
		body = bytes.NewReader([]byte("{}"))
	}

	return &http.Response{
		Status:        http.StatusText(statusCode),
		StatusCode:    statusCode,
		Body:          io.NopCloser(body),
		Header:        make(http.Header),
		ContentLength: int64(body.(*bytes.Reader).Len()),
	}, nil
}

// TestServer creates a test HTTP server with custom handler
func TestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// JSONResponse creates a JSON HTTP response
func JSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// ErrorJSONResponse creates a JSON error response
func ErrorJSONResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
