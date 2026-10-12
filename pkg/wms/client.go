package wms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/google/uuid"

	"last-mile-task/pkg/wms/tares"
)

type Client struct {
	client *http.Client

	endpoints       map[CountryCode]*Connection
	overrideOffices map[int32]CountryCode
}

func New(config Config) (*Client, error) {
	client := &Client{
		client: &http.Client{
			Timeout: config.Timeout,
		},

		endpoints:       make(map[CountryCode]*Connection),
		overrideOffices: make(map[int32]CountryCode),
	}

	for _, endpoint := range config.Endpoints {
		if !endpoint.Enabled {
			slog.Info(
				"wms: endpoint disabled",
				slog.String("uri", endpoint.URI),
				slog.String("country", endpoint.Country),
			)

			continue
		}

		conn, err := endpoint.Connection()
		if err != nil {
			return nil, err
		}

		client.endpoints[conn.country] = conn
	}

	if _, ok := client.endpoints[CountryRussia]; !ok {
		return nil, fmt.Errorf("wms: validate: endpoint for russia must be specified")
	}

	for _, override := range config.OverrideOffices {
		code := countryByName(override.Country)
		if code == CountryUnknown {
			return nil, fmt.Errorf("wms: country code by name: unknown country %s", override.Country)
		}

		for _, officeID := range override.OfficeIDs {
			client.overrideOffices[officeID] = code
		}
	}

	return client, nil
}

func (c *Client) Load(ctx context.Context, shipment Shipment, ts []tares.Tare) error {
	loadedTares := make([]LoadedTare, 0, len(ts))

	for _, t := range ts {
		loadedTares = append(loadedTares, LoadedTare{
			Sticker:         t.Sticker.String(),
			SourceOfficeID:  t.OfficeID,
			CountryID:       t.CountryID,
			LoadedTimestamp: t.FormatTimestamp(),
			DriverName:      shipment.Driver.Name,
			DriverPhone:     shipment.Driver.Phone,
		})
	}

	return c.load(ctx, loadedTares)
}

func (c *Client) Unload(ctx context.Context, shipment Shipment, ts []tares.Tare) error {
	unloadedTares := make([]UnloadedTare, 0, len(ts))

	for _, t := range ts {
		unloadedTares = append(unloadedTares, UnloadedTare{
			Sticker:           t.Sticker.String(),
			SourceOfficeID:    t.OfficeID,
			CountryID:         t.CountryID,
			UnloadedTimestamp: t.FormatTimestamp(),
		})
	}

	return c.unload(ctx, unloadedTares)
}

func (c *Client) load(ctx context.Context, loadedTares []LoadedTare) error {
	const path = "load"

	if len(loadedTares) == 0 {
		slog.Info("wms: no tares to load")

		return nil
	}

	groups := make([]func() error, 0)

	for country, items := range GroupTaresByCountry(loadedTares, c.overrideOffices) {
		groups = append(groups, func() error {
			body, err := json.Marshal(items)
			if err != nil {
				return fmt.Errorf("wms: marshal request body: %w", err)
			}

			return c.do(ctx, c.connection(country), path, body)
		})
	}

	return runParallel(groups...)
}

func (c *Client) unload(ctx context.Context, unloadedTares []UnloadedTare) error {
	const path = "unload"

	if len(unloadedTares) == 0 {
		slog.Info("wms: no tares to unload")

		return nil
	}

	groups := make([]func() error, 0)

	for country, items := range GroupTaresByCountry(unloadedTares, c.overrideOffices) {
		groups = append(groups, func() error {
			body, err := json.Marshal(items)
			if err != nil {
				return fmt.Errorf("wms: marshal request body: %w", err)
			}

			return c.do(ctx, c.connection(country), path, body)
		})
	}

	return runParallel(groups...)
}

func (c *Client) connection(country CountryCode) *Connection {
	conn, ok := c.endpoints[country]
	if !ok {
		conn = c.endpoints[CountryRussia]

		slog.Warn(
			"wms: unknown connection country, fallback to russia",
			slog.String("country", country.Alpha2()),
		)
	}

	return conn
}

func (c *Client) prepareRequest(req *http.Request, auth Auth) {
	req.SetBasicAuth(auth.Username, auth.Password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.NewString())
}

func (c *Client) do(
	ctx context.Context,
	conn *Connection,
	path string,
	body []byte,
) error {
	uri := conn.uri.JoinPath(path).String()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		uri,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("wms: build request: %w", err)
	}

	c.prepareRequest(req, conn.auth)

	slog.Info(
		"wms: tares request",
		slog.String("country", conn.country.String()),
		slog.String("uri", uri),
		slog.String("action", path),
	)

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("wms: %s tares request: %w", path, err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			slog.Error("wms: close response body", slog.Any("error", err))
		}
	}()

	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNoContent {
		return nil
	}

	return fmt.Errorf(
		"wms: %s tares: failed to complete request for %s - status %d",
		path,
		conn.country.String(),
		res.StatusCode,
	)
}

// TODO(LASTMILE-042): the real repo uses golang.org/x/sync/errgroup.WithContext
// (cancels sibling requests on the first error). errgroup is an indirect
// dependency of this skeleton only, so a sync-based runner without context
// cancellation is used to avoid touching go.mod. Requests for different
// countries still run concurrently.
func runParallel(fns ...func() error) error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	for _, fn := range fns {
		wg.Add(1)

		go func(fn func() error) {
			defer wg.Done()

			if err := fn(); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(fn)
	}

	wg.Wait()

	if len(errs) > 1 {
		return fmt.Errorf("wms: %d requests failed: %w", len(errs), errors.Join(errs...))
	}

	if len(errs) == 1 {
		return errs[0]
	}

	return nil
}
