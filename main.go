package main

import (
	"context"
	"fmt"
	"os"

	"last-mile-task/internal/app"
)

var sha string

func main() {
	if err := app.NewApp().Run(context.Background(), sha); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}