// Package wmsmock is a dev/e2e in-process WMS server for the last-mile-task
// skeleton. It does NOT exist in the real repository.
//
// It implements the same HTTP contract that pkg/wms Client uses:
//   - POST {base}/load   -> accepts a JSON array of loaded tares
//   - POST {base}/unload -> accepts a JSON array of unloaded tares
//
// See pkg/wms LoadedTare / UnloadedTare (json tags) for the request shape.
package wmsmock

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
)

const (
	kindLoad   = "load"
	kindUnload = "unload"
)

// Event is a request accepted by the server.
type Event struct {
	// Kind is "load" or "unload" (the request path suffix).
	Kind string
	// Body is the raw JSON request body.
	Body json.RawMessage
	// IdempotencyKey is the Idempotency-Key header sent by the Client.
	IdempotencyKey string
	// Country is the CountryCode header host the request was sent to, kept
	// for debugging multi-endpoint scenarios.
	Country string
}

// Server is an in-process WMS mock.
type Server struct {
	srv *httptest.Server

	mu        sync.Mutex
	events    []Event
	failNext  int
	processed map[string]struct{}
}

// New starts a new mock server on a random localhost port.
func New() *Server {
	s := &Server{
		processed: make(map[string]struct{}),
	}

	s.srv = httptest.NewServer(http.HandlerFunc(s.handle))

	return s
}

// Addr returns the server's host:port address.
func (s *Server) Addr() string {
	return s.srv.Listener.Addr().String()
}

// URL returns the server's base URL, e.g. http://127.0.0.1:54321.
func (s *Server) URL() string {
	return s.srv.URL
}

// Events returns a copy of all accepted events in arrival order.
func (s *Server) Events() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	events := make([]Event, len(s.events))
	copy(events, s.events)

	return events
}

// FailNext makes the next n requests respond with HTTP 500 before the event
// is recorded. A negative count is treated as 0.
func (s *Server) FailNext(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if n < 0 {
		n = 0
	}

	s.failNext = n
}

// Close shuts the server down.
func (s *Server) Close() error {
	s.srv.Close()

	return nil
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	if r.URL.Path != "/"+kindLoad && r.URL.Path != "/"+kindUnload {
		http.NotFound(w, r)

		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)

		return
	}

	if !json.Valid(body) {
		http.Error(w, "invalid json", http.StatusBadRequest)

		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failNext > 0 {
		s.failNext--

		http.Error(w, "wmsmock: simulated failure", http.StatusInternalServerError)

		return
	}

	key := r.Header.Get("Idempotency-Key")

	// Events with the same idempotency key are accepted without error but are
	// recorded only once (best-effort, dedup is not required by the contract).
	if key != "" {
		if _, ok := s.processed[key]; ok {
			http.Error(w, "", http.StatusOK)

			return
		}

		s.processed[key] = struct{}{}
	}

	s.events = append(s.events, Event{
		Kind:           r.URL.Path[1:],
		Body:           json.RawMessage(body),
		IdempotencyKey: key,
		Country:        r.Host,
	})

	w.WriteHeader(http.StatusOK)
}
