package main

import (
	"context"
	"os"

	"github.com/bassner/claudodex/internal/app"
)

var version = "0.1.20"

func main() {
	code := app.Run(context.Background(), app.Config{
		Version: version,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}, os.Args[1:])
	os.Exit(code)
}
