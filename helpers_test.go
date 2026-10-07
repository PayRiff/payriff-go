package payriff

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type recorded struct {
	Method  string
	URL     string
	Headers http.Header
	Body    []byte
}

func (r recorded) JSON(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(r.Body, &m); err != nil {
		t.Fatalf("request body is not a JSON object: %v: %s", err, r.Body)
	}
	return m
}

type stub struct {
	status  int
	headers map[string]string
	body    []byte
}

type mockServer struct {
	*httptest.Server
	mu       sync.Mutex
	stubs    map[string]stub
	requests []recorded
}

func newMockServer(t *testing.T) *mockServer {
	t.Helper()
	m := &mockServer{stubs: map[string]stub{}}
	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		m.mu.Lock()
		m.requests = append(m.requests, recorded{Method: r.Method, URL: r.URL.RequestURI(), Headers: r.Header.Clone(), Body: body})
		s, ok := m.stubs[r.Method+" "+r.URL.RequestURI()]
		if !ok {
			s, ok = m.stubs[r.Method+" "+r.URL.Path]
		}
		m.mu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		for k, v := range s.headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(s.status)
		_, _ = w.Write(s.body)
	}))
	t.Cleanup(m.Close)
	return m
}

func (m *mockServer) stub(method, url string, status int, headers map[string]string, body any) {
	var raw []byte
	switch b := body.(type) {
	case nil:
	case string:
		raw = []byte(b)
	case []byte:
		raw = b
	default:
		raw, _ = json.Marshal(b)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stubs[method+" "+url] = stub{status: status, headers: headers, body: raw}
}

func (m *mockServer) ok(method, url string, payload any) {
	m.stub(method, url, 200, nil, map[string]any{"code": "00000", "message": "Operation performed successfully", "payload": payload})
}

func (m *mockServer) last(t *testing.T) recorded {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.requests) == 0 {
		t.Fatal("no request recorded")
	}
	return m.requests[len(m.requests)-1]
}

func (m *mockServer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.requests)
}

func (m *mockServer) client(t *testing.T, opts ...Option) *Client {
	t.Helper()
	all := append([]Option{WithMerchantID("ES1000000"), WithBaseURL(m.URL)}, opts...)
	c, err := NewClient("app-key", all...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

var ctx = context.Background()

func mustJSONEqual(t *testing.T, want string, got []byte) {
	t.Helper()
	var w, g any
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("bad expected JSON: %v", err)
	}
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("bad actual JSON: %v: %s", err, got)
	}
	wb, _ := json.Marshal(w)
	gb, _ := json.Marshal(g)
	if string(wb) != string(gb) {
		t.Fatalf("JSON mismatch\nwant: %s\n got: %s", wb, gb)
	}
}

func mustInvalid(t *testing.T, err error, message string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected invalid-argument error %q, got nil", message)
	}
	if !strings.HasSuffix(err.Error(), message) || !isInvalid(err) {
		t.Fatalf("expected invalid-argument error %q, got %v", message, err)
	}
}
