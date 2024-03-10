package main

import (
	"log/slog"
	"os"
	"snapsync/cmd"
)

func main() {
	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelDebug)
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	})))
	cmd.Execute()
}
