// Package config loads and validates application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Data directory is the root for all generated files.
	DataDir string

	// M3U source URLs (comma-separated in M3U_URL env var).
	M3UURLs []string

	// BypassHeader skips Content-Type/Content-Disposition checks when downloading.
	BypassHeader bool

	// Processing rules applied to group-title values.
	ScrubHeader     []string // terms whose preceding text is stripped from group-title
	ScrubDefaults   []string
	RemoveTerms     []string // terms removed from group-title
	RemoveDefaults  []string
	ReplaceTerms    map[string]string // key->value replacements
	ReplaceDefaults map[string]string
	ExcludeTerms    []string // entries matching these in group-title are excluded
	Cleaners        []string // which categories to deep-clean: movies, series, tv, unsorted

	// Output toggles.
	LiveTV   bool
	Unsorted bool

	// Directory sync: remove dest items not present in src.
	CleanSync bool

	// Jellyfin integration.
	JellyfinURL string
	APIKey      string
	RefreshLib  bool

	// Threadfin integration.
	ThreadfinUser string
	ThreadfinPass string
	ThreadfinHost string
	ThreadfinPort string

	// EPG URLs for Threadfin/Jellyfin live TV guide.
	EPGURLs []string

	// Logging level (INFO, DEBUG, WARN, ERROR).
	LogLevel string

	TelegramWebhookURL    string
	TelegramWebhookSecret string
}

// Load reads environment variables and returns a validated Config.
func Load() (*Config, error) {
	cfg := &Config{
		DataDir: envOrDefault("DATA_DIR", "/data"),

		M3UURLs:      splitCSV(os.Getenv("M3U_URL")),
		BypassHeader: parseBool(os.Getenv("BYPASS_HEADER")),

		ScrubHeader:     splitCSV(os.Getenv("SCRUB_HEADER")),
		ScrubDefaults:   splitCSV(envOrDefault("SCRUB_DEFAULTS", "HD :,SD :")),
		RemoveTerms:     splitCSV(os.Getenv("REMOVE_TERMS")),
		RemoveDefaults:  splitCSV(envOrDefault("REMOVE_DEFAULTS", "720p,WEB,h264,H264,HDTV,x264,1080p,HEVC,x265,X265")),
		ReplaceTerms:    parseKeyValuePairs(os.Getenv("REPLACE_TERMS")),
		ReplaceDefaults: parseKeyValuePairs(envOrDefault("REPLACE_DEFAULTS", "1/2=\u00BD,/=-")),
		ExcludeTerms:    splitCSV(os.Getenv("EXCLUDE_TERMS")),
		Cleaners:        splitCSV(os.Getenv("CLEANERS")),

		LiveTV:    parseBool(envOrDefault("LIVE_TV", "true")),
		Unsorted:  parseBool(os.Getenv("UNSORTED")),
		CleanSync: parseBool(os.Getenv("CLEAN_SYNC")),

		JellyfinURL: os.Getenv("JELLYFIN_URL"),
		APIKey:      os.Getenv("API_KEY"),
		RefreshLib:  parseBool(os.Getenv("REFRESH_LIB")),

		ThreadfinUser: os.Getenv("TF_USER"),
		ThreadfinPass: os.Getenv("TF_PASS"),
		ThreadfinHost: os.Getenv("TF_HOST"),
		ThreadfinPort: os.Getenv("TF_PORT"),

		EPGURLs: splitCSV(os.Getenv("EPG_URL")),

		LogLevel: envOrDefault("LOG_LEVEL", "INFO"),

		TelegramWebhookURL:    os.Getenv("TELEGRAM_WEBHOOK_URL"),
		TelegramWebhookSecret: os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
	}

	if len(cfg.M3UURLs) == 0 {
		return nil, fmt.Errorf("M3U_URL is required")
	}

	return cfg, nil
}

// Paths returns the resolved directory/file paths relative to DataDir.
func (c *Config) Paths() Paths {
	d := c.DataDir
	return Paths{
		M3UDir:        d + "/m3u",
		M3UFilePath:   d + "/m3u_file.m3u",
		LiveTVFile:    d + "/livetv.m3u",
		LiveTVDir:     d + "/VODS/Live_TV",
		TVDir:         d + "/TV_VOD",
		MoviesDir:     d + "/Movie_VOD",
		UnsortedDir:   d + "/Unsorted_VOD",
		LocalTVDir:    d + "/VODS/TV_VOD",
		LocalMovDir:   d + "/VODS/Movie_VOD",
		LocalUnsorted: d + "/VODS/Unsorted_VOD",
	}
}

// Paths holds all resolved filesystem paths.
type Paths struct {
	M3UDir        string
	M3UFilePath   string
	LiveTVFile    string
	LiveTVDir     string
	TVDir         string
	MoviesDir     string
	UnsortedDir   string
	LocalTVDir    string
	LocalMovDir   string
	LocalUnsorted string
}

// AllDirs returns every directory that must exist before processing.
func (p Paths) AllDirs() []string {
	return []string{
		p.M3UDir,
		p.LiveTVDir,
		p.TVDir,
		p.MoviesDir,
		p.UnsortedDir,
		p.LocalTVDir,
		p.LocalMovDir,
		p.LocalUnsorted,
	}
}

// CleanerEnabled returns true if the given category is enabled in CLEANERS.
func (c *Config) CleanerEnabled(category string) bool {
	for _, cl := range c.Cleaners {
		if strings.EqualFold(cl, category) {
			return true
		}
	}
	return false
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func parseBool(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "true", "yes", "1", "t":
		return true
	}
	return false
}

// splitCSV splits a potentially quoted, comma-separated string into trimmed items.
// It supports escaped commas (\,).
func splitCSV(s string) []string {
	s = stripOuterQuotes(s)
	if s == "" {
		return nil
	}

	// Split on unescaped commas.
	re := regexp.MustCompile(`(?:\\,|[^,])+`)
	matches := re.FindAllString(s, -1)

	var result []string
	for _, m := range matches {
		// Unescape \, and other backslash sequences.
		cleaned := regexp.MustCompile(`\\(.)`).ReplaceAllString(m, "$1")
		cleaned = strings.TrimSpace(cleaned)
		if cleaned != "" {
			result = append(result, cleaned)
		}
	}
	return result
}

// parseKeyValuePairs parses "key1=val1,key2=val2" into a map.
func parseKeyValuePairs(s string) map[string]string {
	s = stripOuterQuotes(s)
	if s == "" {
		return nil
	}

	result := make(map[string]string)
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if idx := strings.Index(pair, "="); idx > 0 {
			key := strings.TrimSpace(pair[:idx])
			val := strings.TrimSpace(pair[idx+1:])
			result[key] = val
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func stripOuterQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == s[len(s)-1] && (s[0] == '\'' || s[0] == '"') {
		s = s[1 : len(s)-1]
	}
	return s
}
