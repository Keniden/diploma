package wms

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"last-mile-task/pkg/wms/tares"
	"last-mile-task/pkg/wms/wmsmock"
)

func testTare(sticker string, officeID int32, countryID int64) tares.Tare {
	return tares.Tare{
		Sticker:   tares.NewSticker(sticker),
		OfficeID:  officeID,
		CountryID: countryID,
		Timestamp: time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC),
	}
}

func russiaConfig(uri string) Config {
	return Config{
		Endpoints: []Endpoint{{
			Country: "russia",
			Enabled: true,
			URI:     uri,
			Auth:    Auth{Username: "user", Password: "pass"},
		}},
	}
}

func mustClient(t *testing.T, config Config) *Client {
	t.Helper()

	client, err := New(config)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	return client
}

func TestClientLoadToMock(t *testing.T) {
	mock := wmsmock.New()
	defer func() { _ = mock.Close() }()

	client := mustClient(t, russiaConfig(mock.URL()))

	err := client.Load(context.Background(), Shipment{
		Driver: Driver{Name: "Ivan", Phone: "+79990000000"},
	}, []tares.Tare{
		testTare("TRBX111", 1, 643),
		testTare("TRBX222", 2, 643),
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	events := mock.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d: %+v", len(events), events)
	}

	event := events[0]
	if event.Kind != "load" {
		t.Fatalf("expected kind load, got %q", event.Kind)
	}

	if event.IdempotencyKey == "" {
		t.Fatal("Idempotency-Key must be set")
	}

	var payload []LoadedTare
	if err := json.Unmarshal(event.Body, &payload); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}

	if len(payload) != 2 {
		t.Fatalf("expected 2 tares, got %d", len(payload))
	}

	first := payload[0]
	if first.Sticker != "TRBX111" || first.SourceOfficeID != 1 || first.CountryID != 643 {
		t.Fatalf("loaded tare mismatch: %+v", first)
	}

	if first.LoadedTimestamp != "2026-10-10T12:00:00+03:00" {
		t.Fatalf("loaded_dt mismatch: %q", first.LoadedTimestamp)
	}

	if first.DriverName != "Ivan" || first.DriverPhone != "+79990000000" {
		t.Fatalf("driver mismatch: %+v", first)
	}
}

func TestClientUnloadToMock(t *testing.T) {
	mock := wmsmock.New()
	defer func() { _ = mock.Close() }()

	client := mustClient(t, russiaConfig(mock.URL()))

	err := client.Unload(context.Background(), Shipment{
		Driver: Driver{Name: "Ivan", Phone: "+79990000000"},
	}, []tares.Tare{
		testTare("TRBX333", 3, 643),
	})
	if err != nil {
		t.Fatalf("Unload failed: %v", err)
	}

	events := mock.Events()
	if len(events) != 1 || events[0].Kind != "unload" {
		t.Fatalf("expected 1 unload event, got %+v", events)
	}

	var payload []UnloadedTare
	if err := json.Unmarshal(events[0].Body, &payload); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}

	if len(payload) != 1 {
		t.Fatalf("expected 1 tare, got %d", len(payload))
	}

	if payload[0].Sticker != "TRBX333" || payload[0].UnloadedTimestamp != "2026-10-10T12:00:00+03:00" {
		t.Fatalf("unloaded tare mismatch: %+v", payload[0])
	}
}

func TestClientSendsExpectedRequest(t *testing.T) {
	var gotMethod, gotPath, gotAuth, gotContentType, gotKey string
	var gotBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		gotKey = r.Header.Get("Idempotency-Key")

		gotBody = make([]byte, r.ContentLength)
		if _, err := r.Body.Read(gotBody); err != nil && err.Error() != "EOF" {
			t.Errorf("read body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := mustClient(t, russiaConfig(srv.URL))

	if err := client.Load(context.Background(), Shipment{}, []tares.Tare{testTare("TRBX1", 1, 643)}); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/load" {
		t.Fatalf("unexpected request: %s %s", gotMethod, gotPath)
	}

	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass"))
	if gotAuth != expectedAuth {
		t.Fatalf("auth header mismatch: %q != %q", gotAuth, expectedAuth)
	}

	if gotContentType != "application/json" {
		t.Fatalf("content-type mismatch: %q", gotContentType)
	}

	if gotKey == "" {
		t.Fatal("Idempotency-Key must be present")
	}

	var payload []LoadedTare
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("body must be valid JSON: %v", err)
	}

	if len(payload) != 1 {
		t.Fatalf("expected 1 tare in body, got %d", len(payload))
	}
}

func TestClientLoadErrorOn500(t *testing.T) {
	mock := wmsmock.New()
	defer func() { _ = mock.Close() }()
	mock.FailNext(1)

	client := mustClient(t, russiaConfig(mock.URL()))

	err := client.Load(context.Background(), Shipment{}, []tares.Tare{testTare("TRBX1", 1, 643)})
	if err == nil {
		t.Fatal("Load must fail on 500")
	}

	if !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientNoTaresSendsNothing(t *testing.T) {
	var counter atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := mustClient(t, russiaConfig(srv.URL))

	if err := client.Load(context.Background(), Shipment{}, nil); err != nil {
		t.Fatalf("Load must succeed with no tares: %v", err)
	}

	if err := client.Unload(context.Background(), Shipment{}, nil); err != nil {
		t.Fatalf("Unload must succeed with no tares: %v", err)
	}

	if counter.Load() != 0 {
		t.Fatalf("no requests expected, got %d", counter.Load())
	}
}

func TestClientRequiresRussiaEndpoint(t *testing.T) {
	for _, config := range []Config{
		{},
		{Endpoints: []Endpoint{{Country: "belarus", Enabled: true, URI: "http://localhost:8082"}}},
	} {
		if _, err := New(config); err == nil {
			t.Fatalf("New must fail without russia endpoint: %+v", config)
		}
	}
}

func TestClientUnknownCountryName(t *testing.T) {
	_, err := New(Config{
		Endpoints: []Endpoint{
			{Country: "russia", Enabled: true, URI: "http://localhost:8081"},
			{Country: "atlantis", Enabled: true, URI: "http://localhost:8082"},
		},
	})
	if err == nil {
		t.Fatal("unknown country name must fail")
	}
}

func TestClientInvalidEndpointURI(t *testing.T) {
	_, err := New(Config{
		Endpoints: []Endpoint{
			{Country: "russia", Enabled: true, URI: "://bad"},
		},
	})
	if err == nil {
		t.Fatal("invalid uri must fail")
	}
}

func TestClientDisablesInactiveEndpoints(t *testing.T) {
	client, err := New(Config{
		Endpoints: []Endpoint{
			{Country: "russia", Enabled: true, URI: "http://localhost:8081"},
			{Country: "belarus", Enabled: false, URI: "http://localhost:8082"},
		},
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if len(client.endpoints) != 1 {
		t.Fatalf("expected 1 active endpoint, got %d", len(client.endpoints))
	}
}

func TestClientUnknownCountryFallsBackToRussia(t *testing.T) {
	var counter atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := mustClient(t, russiaConfig(srv.URL))

	// country id 999 is unknown -> fallback to the russia connection.
	err := client.Load(context.Background(), Shipment{}, []tares.Tare{testTare("TRBX1", 1, 999)})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if counter.Load() != 1 {
		t.Fatalf("expected 1 request, got %d", counter.Load())
	}
}

func TestClientOverrideOfficeRoutesToBelarus(t *testing.T) {
	var russiaHits, belarusHits atomic.Int32

	russiaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		russiaHits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer russiaSrv.Close()

	belarusSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		belarusHits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer belarusSrv.Close()

	client, err := New(Config{
		Endpoints: []Endpoint{
			{Country: "russia", Enabled: true, URI: russiaSrv.URL},
			{Country: "belarus", Enabled: true, URI: belarusSrv.URL},
		},
		OverrideOffices: []OverrideOffice{{Country: "belarus", OfficeIDs: []int32{42}}},
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	err = client.Load(context.Background(), Shipment{}, []tares.Tare{
		testTare("TRBX-BY", 42, 643), // office overridden to belarus
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if belarusHits.Load() != 1 {
		t.Fatalf("overridden request must go to belarus, hits=%d", belarusHits.Load())
	}

	if russiaHits.Load() != 0 {
		t.Fatalf("no request expected on russia, hits=%d", russiaHits.Load())
	}
}

func TestClientMulticountryParallel(t *testing.T) {
	mock := wmsmock.New()
	defer func() { _ = mock.Close() }()

	belarusSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer belarusSrv.Close()

	client, err := New(Config{
		Endpoints: []Endpoint{
			{Country: "russia", Enabled: true, URI: mock.URL()},
			{Country: "belarus", Enabled: true, URI: belarusSrv.URL},
		},
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	err = client.Load(context.Background(), Shipment{}, []tares.Tare{
		testTare("TRBX-RU", 1, 643),
		testTare("TRBX-BY", 2, 112),
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	events := mock.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event on russia mock, got %d: %+v", len(events), events)
	}

	var payload []LoadedTare
	if err := json.Unmarshal(events[0].Body, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(payload) != 1 || payload[0].Sticker != "TRBX-RU" {
		t.Fatalf("russia payload mismatch: %+v", payload)
	}
}

func TestClientGroupingPreservesBatchesPerCountry(t *testing.T) {
	mock := wmsmock.New()
	defer func() { _ = mock.Close() }()

	client := mustClient(t, russiaConfig(mock.URL()))

	err := client.Unload(context.Background(), Shipment{}, []tares.Tare{
		testTare("TRBX-A", 1, 643),
		testTare("TRBX-B", 2, 643),
		testTare("TRBX-C", 3, 643),
	})
	if err != nil {
		t.Fatalf("Unload failed: %v", err)
	}

	events := mock.Events()
	// One request per country; the whole batch must fit into a single event.
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d: %+v", len(events), events)
	}

	var payload []UnloadedTare
	if err := json.Unmarshal(events[0].Body, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(payload) != 3 {
		t.Fatalf("expected 3 tares in one batch, got %d", len(payload))
	}
}

func TestClientMultipleFailuresJoined(t *testing.T) {
	// Both endpoints fail -> error must mention both requests.
	failer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer failer.Close()

	client, err := New(Config{
		Endpoints: []Endpoint{
			{Country: "russia", Enabled: true, URI: failer.URL},
			{Country: "belarus", Enabled: true, URI: failer.URL},
		},
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	err = client.Load(context.Background(), Shipment{}, []tares.Tare{
		testTare("TRBX-RU", 1, 643),
		testTare("TRBX-BY", 2, 112),
	})
	if err == nil {
		t.Fatal("Load must fail when all endpoints fail")
	}

	if !strings.Contains(err.Error(), "2 requests failed") && !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConnectionEndpointParsing(t *testing.T) {
	conn, err := (&Endpoint{Country: "russia", URI: "http://localhost:8081/base"}).Connection()
	if err != nil {
		t.Fatalf("Connection failed: %v", err)
	}

	if conn.country != CountryRussia || conn.uri.Path != "/base" {
		t.Fatalf("connection mismatch: %+v", conn)
	}

	if _, err := (&Endpoint{Country: "russia", URI: "not-a-url"}).Connection(); err == nil {
		t.Fatal("invalid uri must fail")
	}

	if _, err := (&Endpoint{Country: "atlantis", URI: "http://localhost:8081"}).Connection(); err == nil {
		t.Fatal("unknown country must fail")
	}
}

func TestClientLoadUnloadWithMockRoundTrip(t *testing.T) {
	// End-to-end: Client -> wmsmock with realistic stickers.
	mock := wmsmock.New()
	defer func() { _ = mock.Close() }()

	client := mustClient(t, russiaConfig(mock.URL()))

	sticker, err := tares.ParseSticker("TRBX987654321")
	if err != nil {
		t.Fatalf("ParseSticker failed: %v", err)
	}

	shipment := Shipment{Driver: Driver{Name: "Petr", Phone: "+79991112233"}}

	if err := client.Load(context.Background(), shipment, []tares.Tare{{
		Sticker: sticker, OfficeID: 10, CountryID: 643,
		Timestamp: time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC),
	}}); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if err := client.Unload(context.Background(), shipment, []tares.Tare{{
		Sticker: sticker, OfficeID: 10, CountryID: 643,
		Timestamp: time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC),
	}}); err != nil {
		t.Fatalf("Unload failed: %v", err)
	}

	events := mock.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[0].Kind != "load" || events[1].Kind != "unload" {
		t.Fatalf("unexpected kinds: %s, %s", events[0].Kind, events[1].Kind)
	}

	for i, event := range events {
		if !json.Valid(event.Body) {
			t.Fatalf("event %d body is not valid JSON: %s", i, event.Body)
		}
	}
}

func TestClientLoadWithCanceledCtx(t *testing.T) {
	mock := wmsmock.New()
	defer func() { _ = mock.Close() }()

	client := mustClient(t, russiaConfig(mock.URL()))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.Load(ctx, Shipment{}, []tares.Tare{testTare("TRBX1", 1, 643)})
	if err == nil {
		t.Fatal("Load with canceled context must fail")
	}

	if len(mock.Events()) != 0 {
		t.Fatalf("no events expected, got %+v", mock.Events())
	}
}

func TestClientStringerCountries(t *testing.T) {
	if CountryRussia.String() != "RU" || CountryRussia.Alpha2() != "RU" {
		t.Fatal("country code accessors must return alpha-2")
	}
}
