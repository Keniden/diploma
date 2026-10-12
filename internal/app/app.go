// Package app bootstraps the last-mile-task service: main.go delegates here.
package app

import (
	"context"
	"fmt"
	"os"

	"last-mile-task/internal/platform/application"
	"last-mile-task/internal/platform/application/builder"
)

// App is the service entry point.
type App struct{}

// NewApp builds the entry point.
func NewApp() *App { return &App{} }

// Run starts the service: builder builds dependencies, application wires and
// runs until the context is canceled.
func (a *App) Run(ctx context.Context, version string) error {
	b, err := builder.New(ctx, version)
	if err != nil {
		return err
	}

	app, err := application.New(b)
	if err != nil {
		return err
	}
	defer func() {
		if err := app.Shutdown(); err != nil {
			fmt.Fprintf(os.Stderr, "application shutdown: %v\n", err)
		}
	}()

	if err := app.Run(ctx); err != nil {
		os.Exit(1)
	}
	return nil
}