package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredericomozzato/crypto_tracker/internal/api"
	"github.com/fredericomozzato/crypto_tracker/internal/db"
	"github.com/fredericomozzato/crypto_tracker/internal/store"
	"github.com/fredericomozzato/crypto_tracker/internal/ui"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	debug := flag.Bool("debug", false, "enable file logging")
	flag.Parse()

	cleanup := setupLogger(*debug)
	defer cleanup()

	// Root context is cancelled on SIGINT/SIGTERM, propagating cancellation to
	// all in-flight HTTP requests and DB queries. This is safe because every DB
	// write is an atomic upsert — there is no risk of partial or corrupt data on
	// abrupt shutdown. Must revisit if multi-statement transactions are added.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Open database
	dbPath, err := dataFilePath()
	if err != nil {
		slog.Error("determining data path", "error", err)
		return 1
	}

	// The raw *sql.DB handle is passed to the store and not used directly after
	// this point. The store owns the DB lifecycle (including Close via defer below).
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		slog.Error("opening database", "error", err)
		return 1
	}

	// Create store
	s := store.NewSQLiteStore(database)
	defer func() {
		if err := s.Close(); err != nil {
			slog.Error("closing store", "error", err)
		}
	}()

	// Create API client
	apiKey := os.Getenv("COINGECKO_API_KEY")
	client := api.NewHTTPClient(apiKey)

	// Warm the supported-currency cache if it is empty.
	// This costs one API call on first launch (or after a DB reset) and makes
	// the Settings tab immediately useful. A failure here is non-fatal.
	codes, err := s.GetCachedCurrencies(ctx)
	if err != nil {
		slog.Error("checking currency cache", "error", err)
	} else if len(codes) == 0 {
		if fetched, fetchErr := client.FetchSupportedCurrencies(ctx); fetchErr != nil {
			slog.Error("fetching supported currencies", "error", fetchErr)
		} else if storeErr := s.UpsertCurrencies(ctx, fetched); storeErr != nil {
			slog.Error("caching supported currencies", "error", storeErr)
		}
	}

	// Create model with dependencies
	model := ui.NewAppModel(ctx, s, client)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithContext(ctx),
	)

	if _, err := p.Run(); err != nil {
		slog.Error("program exited with error", "error", err)
		return 1
	}
	return 0
}

func setupLogger(debug bool) func() {
	if debug {
		logPath, err := logFilePath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
			return func() {}
		}
		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
			slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
			return func() {}
		}

		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
			return func() {}
		}

		slog.SetDefault(slog.New(slog.NewTextHandler(f, nil)))
		return func() {
			if err := f.Close(); err != nil {
				slog.Error("closing log file", "error", err)
			}
		}
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	return func() {}
}

func logFilePath() (string, error) {
	stateDir := os.Getenv("XDG_STATE_HOME")
	if stateDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determining home directory: %w", err)
		}
		stateDir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(stateDir, "crypto_tracker", "app.log"), nil
}

func dataFilePath() (string, error) {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determining home directory: %w", err)
		}
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "crypto_tracker", "data.db"), nil
}
