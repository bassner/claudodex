package main

import (
	"context"
	"os"

	"github.com/bassner/claudodex/internal/app"
	"github.com/bassner/claudodex/internal/selfupdate"
)

var version = "0.5.12"
var installMethod = "unknown"

func main() {
	code := app.Run(context.Background(), app.Config{
		Version: version,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Updater: selfupdate.New(selfupdate.Config{Version: version, InstallMethod: installMethod, Stdin: os.Stdin, Stderr: os.Stderr}),
	}, os.Args[1:])
	os.Exit(code)
}
