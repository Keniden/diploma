package wmsmock

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func testRequest(t *testing.T, method, url, body, key string) (*http.Response, error) {
	t.Helper()

	req, err := http.NewRequest(method, url, bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}

	return http.DefaultClient.Do(req)
}

func TestServerURLAndAddr(t *testing.T) {
	s := New()
	defer func() { _ = s.Close() }()

	if s.URL() == "" || s.Addr() == "" {
		t.Fatalf("empty URL(%q) or Addr(%q)", s.URL(), s.Addr())
	}
}

func TestServerAcceptsLoad(t *testing.T) {
	s := New()
	defer func() { _ = s.Close() }()

	body := `[{"tare_sticker":"TRBX123","office_id":1,"country_id":643,"loaded_dt":"2026-10-10T12:00:00+03:00","driver_name":"Ivan","driver_phone":"+79990000000"}]`

	res, err := testRequest(t, http.MethodPost, s.URL()+"/load", body, "key-1")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	events := s.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.Kind != "load" {
		t.Fatalf("expected kind load, got %q", event.Kind)
	}

	if string(event.Body) != body {
		t.Fatalf("body mismatch: %s != %s", event.Body, body)
	}

	if event.IdempotencyKey != "key-1" {
		t.Fatalf("idempotency key mismatch: %q", event.IdempotencyKey)
	}
}

func TestServerAcceptsUnload(t *testing.T) {
	s := New()
	defer func() { _ = s.Close() }()

	body := `[{"tare_sticker":"TRBX456","office_id":2,"country_id":643,"unloaded_dt":"2026-10-10T12:00:00+03:00"}]`

	res, err := testRequest(t, http.MethodPost, s.URL()+"/unload", body, "")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	events := s.Events()
	if len(events) != 1 || events[0].Kind != "unload" {
		t.Fatalf("unexpected events: %+v", events)
	}

	var tares []map[string]any
	if err := json.Unmarshal(events[0].Body, &tares); err != nil {
		t.Fatalf("body must be a JSON array: %v", err)
	}

	if len(tares) != 1 || tares[0]["tare_sticker"] != "TRBX456" {
		t.Fatalf("unexpected tares: %+v", tares)
	}
}

func TestServerFailNext(t *testing.T) {
	s := New()
	defer func() { _ = s.Close() }()

	s.FailNext(2)

	for i := 0; i < 2; i++ {
		res, err := testRequest(t, http.MethodPost, s.URL()+"/load", `[]`, "k")
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		_ = res.Body.Close()

		if res.StatusCode != http.StatusInternalServerError {
			t.Fatalf("expected 500 on failed request %d, got %d", i, res.StatusCode)
		}
	}

	res, err := testRequest(t, http.MethodPost, s.URL()+"/load", `[]`, "k")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 after fail window, got %d", res.StatusCode)
	}

	if len(s.Events()) != 1 {
		t.Fatalf("failed requests must not be recorded, events: %+v", s.Events())
	}
}

func TestServerIdempotencyKey(t *testing.T) {
	s := New()
	defer func() { _ = s.Close() }()

	for i := 0; i < 3; i++ {
		res, err := testRequest(t, http.MethodPost, s.URL()+"/load", `[{"tare_sticker":"TRBX1"}]`, "same-key")
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		_ = res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("duplicate event must be accepted without error, got %d", res.StatusCode)
		}
	}

	if events := s.Events(); len(events) != 1 {
		t.Fatalf("duplicate events must be recorded once, got %d", len(events))
	}
}

func TestServerRejectsUnknownPathAndMethod(t *testing.T) {
	s := New()
	defer func() { _ = s.Close() }()

	res, err := testRequest(t, http.MethodPost, s.URL()+"/unknown", `[]`, "")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}

	res, err = testRequest(t, http.MethodGet, s.URL()+"/load", ``, "")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = res.Body.Close()

	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", res.StatusCode)
	}

	if len(s.Events()) != 0 {
		t.Fatalf("rejected requests must not be recorded, got %d", len(s.Events()))
	}
}

func TestServerRejectsInvalidJSON(t *testing.T) {
	s := New()
	defer func() { _ = s.Close() }()

	res, err := testRequest(t, http.MethodPost, s.URL()+"/load", `{invalid`, "")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestServerEventsReturnsCopy(t *testing.T) {
	s := New()
	defer func() { _ = s.Close() }()

	_, err := testRequest(t, http.MethodPost, s.URL()+"/load", `[]`, "k")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	events := s.Events()
	events[0].Kind = "mutated"

	if got := s.Events(); got[0].Kind != "load" {
		t.Fatalf("Events() must return a copy, got kind %q", got[0].Kind)
	}
}
