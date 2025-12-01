package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"

	adventofcode "github.com/seabird-chat/seabird-adventofcode-plugin"
)

func Env(logger *slog.Logger, key string) string {
	ret, ok := os.LookupEnv(key)

	if !ok {
		logger.With(slog.Any("var", key)).Error("Required environment variable not found")
		os.Exit(1)
	}

	return ret
}

func EnvDefault(key string, def string) string {
	if ret, ok := os.LookupEnv(key); ok {
		return ret
	}
	return def
}

func main() {
	var logger *slog.Logger

	if isatty.IsTerminal(os.Stdout.Fd()) {
		logger = slog.New(tint.NewHandler(os.Stdout, &tint.Options{
			AddSource:  true,
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
		}))
	} else {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelInfo,
		}))
	}

	slog.SetDefault(logger)

	config := adventofcode.Config{
		SeabirdHost:    Env(logger, "SEABIRD_HOST"),
		SeabirdToken:   Env(logger, "SEABIRD_TOKEN"),
		AOCSession:     Env(logger, "AOC_SESSION"),
		AOCLeaderboard: Env(logger, "AOC_LEADERBOARD"),
		AOCChannel:     Env(logger, "AOC_CHANNEL"),
		TimestampFile:  EnvDefault("TIMESTAMP_FILE", "./aoc_timestamp.txt"),
	}

	plugin, err := adventofcode.NewPlugin(logger, config)
	if err != nil {
		logger.With(slog.Any("error", err)).Error("failed to load backend")
		os.Exit(2)
	}

	err = plugin.Run(context.Background())
	if err != nil {
		logger.With(slog.Any("error", err)).Error("failed to run backend")
		os.Exit(3)
	}
}
