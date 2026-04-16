// m3uparser parses M3U playlist files and creates .strm file libraries
// for use with Jellyfin, Threadfin, and other media servers.
//
// Designed to run as a Kubernetes CronJob (run once, exit).
package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/dani/m3uparser/internal/config"
	"github.com/dani/m3uparser/internal/entry"
	"github.com/dani/m3uparser/internal/jellyfin"
	"github.com/dani/m3uparser/internal/m3u"
	"github.com/dani/m3uparser/internal/notify"
	"github.com/dani/m3uparser/internal/strm"
	"github.com/dani/m3uparser/internal/threadfin"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
	slog.Info("job complete")
}

func run() error {
	// Load configuration.
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Configure structured logging.
	setupLogging(cfg.LogLevel)

	paths := cfg.Paths()

	// Step 1: Create required directories.
	slog.Info("initializing directories")
	for _, dir := range paths.AllDirs() {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	// Step 2: Download M3U files.
	slog.Info("downloading M3U files", "urls", len(cfg.M3UURLs))
	if err := m3u.Download(cfg.M3UURLs, paths.M3UDir, cfg.BypassHeader); err != nil {
		return err
	}

	// Step 3: Combine downloaded M3U files into one.
	slog.Info("combining M3U files")
	if err := m3u.CombineFiles(paths.M3UDir, paths.M3UFilePath); err != nil {
		slog.Error("M3U combine failed, cleaning up", "error", err)
		cleanup(cfg, paths)
		return err
	}

	// Step 4: Parse the combined M3U file.
	slog.Info("parsing M3U file")
	parseCfg := m3u.ParseConfig{
		ScrubHeader:     cfg.ScrubHeader,
		ScrubDefaults:   cfg.ScrubDefaults,
		ReplaceTerms:    cfg.ReplaceTerms,
		ReplaceDefaults: cfg.ReplaceDefaults,
		RemoveTerms:     cfg.RemoveTerms,
		RemoveDefaults:  cfg.RemoveDefaults,
		ExcludeTerms:    cfg.ExcludeTerms,
		Cleaners: entry.CleanerFlags{
			Movies:   cfg.CleanerEnabled("movies"),
			Series:   cfg.CleanerEnabled("series"),
			TV:       cfg.CleanerEnabled("tv"),
			Unsorted: cfg.CleanerEnabled("unsorted"),
		},
	}

	result, err := m3u.ParseFile(paths.M3UFilePath, parseCfg)
	if err != nil {
		return err
	}

	// Step 5: Write .strm files.
	slog.Info("writing .strm files")
	stats, writeErrors := strm.WriteAll(result.Entries, paths.TVDir, paths.MoviesDir, paths.UnsortedDir)
	result.Errors = append(result.Errors, writeErrors...)

	// Step 6: Write live TV playlist.
	liveTVCount := 0
	if cfg.LiveTV {
		slog.Info("writing live TV playlist")
		var err error
		liveTVCount, err = strm.WriteLiveTVPlaylist(result.Entries, paths.LiveTVFile)
		if err != nil {
			slog.Error("failed to write live TV playlist", "error", err)
		}
	}

	// Step 7: Sync to local directories.
	// A DiffCollector is wired in when TELEGRAM_WEBHOOK_URL is set; otherwise nil (no-op).
	slog.Info("syncing directories")
	var diff *notify.DiffCollector
	if cfg.TelegramWebhookURL != "" {
		diff = &notify.DiffCollector{}
	}

	syncDir(paths.MoviesDir, paths.LocalMovDir, cfg.CleanSync, diff, "movie")
	syncDir(paths.TVDir, paths.LocalTVDir, cfg.CleanSync, diff, "tv")
	if cfg.Unsorted {
		syncDir(paths.UnsortedDir, paths.LocalUnsorted, cfg.CleanSync, diff, "unsorted")
	}
	if cfg.LiveTV {
		if err := strm.MoveFile(paths.LiveTVFile, paths.LiveTVDir); err != nil {
			slog.Error("failed to move live TV file", "error", err)
		}
	}

	// Step 8: Report results.
	movieCount := strm.CountStrmFiles(paths.MoviesDir)
	tvCount := strm.CountStrmFiles(paths.TVDir)
	unsortedCount := strm.CountStrmFiles(paths.UnsortedDir)

	slog.Info("processing complete",
		"movies", movieCount,
		"episodes", tvCount,
		"unsorted", unsortedCount,
		"livetv_channels", liveTVCount,
		"errors", len(result.Errors),
	)

	if len(result.Errors) > 0 {
		slog.Warn("errors encountered during processing", "count", len(result.Errors))
		for _, e := range result.Errors {
			slog.Warn("processing error", "detail", e)
		}
	}

	// Step 9: Cleanup intermediate files.
	cleanup(cfg, paths)

	// Step 10: Post-processing integrations.
	runIntegrations(cfg, stats, diff, len(result.Errors))

	return nil
}

func syncDir(src, dest string, removeExtra bool, diff *notify.DiffCollector, mediaType string) {
	if err := strm.SyncDirectories(src, dest, removeExtra, diff, mediaType); err != nil {
		slog.Error("directory sync failed", "src", src, "dest", dest, "error", err)
	}
}

func cleanup(cfg *config.Config, paths config.Paths) {
	cp := strm.CleanupPaths{
		Files:  []string{paths.M3UFilePath},
		Dirs:   []string{paths.MoviesDir, paths.TVDir, paths.UnsortedDir},
		M3UDir: paths.M3UDir,
	}

	if !cfg.LiveTV {
		cp.Files = append(cp.Files, paths.LiveTVFile)
		cp.Dirs = append(cp.Dirs, paths.LiveTVDir)
	}
	if !cfg.Unsorted {
		cp.Dirs = append(cp.Dirs, paths.LocalUnsorted)
	}

	strm.Cleanup(cp)
}

func runIntegrations(cfg *config.Config, stats *strm.Stats, diff *notify.DiffCollector, errorCount int) {
	_ = stats // available for future use

	// Jellyfin: refresh library and/or guide.
	if jf := jellyfin.New(cfg.JellyfinURL, cfg.APIKey); jf != nil {
		slog.Info("running jellyfin integration")
		if err := jf.Ping(8, 7*time.Second); err != nil {
			slog.Error("jellyfin unreachable, skipping", "error", err)
		} else {
			if cfg.LiveTV {
				if err := jf.RefreshGuide(); err != nil {
					slog.Error("jellyfin guide refresh failed", "error", err)
				}
			}
			if cfg.RefreshLib {
				if err := jf.RefreshLibrary(); err != nil {
					slog.Error("jellyfin library refresh failed", "error", err)
				}
			}
		}
	}

	// Threadfin: update M3U/EPG data.
	if tf := threadfin.New(cfg.ThreadfinHost, cfg.ThreadfinPort, cfg.ThreadfinUser, cfg.ThreadfinPass); tf != nil {
		slog.Info("running threadfin integration")
		if err := tf.RunFullUpdate(); err != nil {
			slog.Error("threadfin update failed", "error", err)
		}
	}

	// Telegram: send media diff notification.
	if cfg.TelegramWebhookURL != "" {
		payload := diff.Build(errorCount)
		nc := notify.New(cfg.TelegramWebhookURL, cfg.TelegramWebhookSecret)
		if err := nc.Send(payload); err != nil {
			slog.Error("telegram notification failed", "error", err)
		} else {
			slog.Info("telegram notification sent",
				"added", len(payload.Added),
				"removed", len(payload.Removed),
			)
		}
	}
}

func setupLogging(level string) {
	var logLevel slog.Level
	switch level {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "WARN":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))
}
