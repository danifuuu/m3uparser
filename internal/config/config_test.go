package config

import (
	"os"
	"testing"
)

func TestLoadRequiresM3UURL(t *testing.T) {
	// Clear any existing M3U_URL.
	os.Unsetenv("M3U_URL")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when M3U_URL is empty")
	}
}

func TestLoadWithValidURL(t *testing.T) {
	t.Setenv("M3U_URL", "http://example.com/playlist.m3u")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.M3UURLs) != 1 || cfg.M3UURLs[0] != "http://example.com/playlist.m3u" {
		t.Errorf("expected single URL, got %v", cfg.M3UURLs)
	}
}

func TestLoadMultipleURLs(t *testing.T) {
	t.Setenv("M3U_URL", "http://a.com/1.m3u,http://b.com/2.m3u")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.M3UURLs) != 2 {
		t.Fatalf("expected 2 URLs, got %d", len(cfg.M3UURLs))
	}
	if cfg.M3UURLs[0] != "http://a.com/1.m3u" || cfg.M3UURLs[1] != "http://b.com/2.m3u" {
		t.Errorf("unexpected URLs: %v", cfg.M3UURLs)
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"yes", true},
		{"1", true},
		{"t", true},
		{"false", false},
		{"False", false},
		{"", false},
		{"no", false},
		{"0", false},
	}

	for _, tt := range tests {
		got := parseBool(tt.input)
		if got != tt.expected {
			t.Errorf("parseBool(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"a,b,c", []string{"a", "b", "c"}},
		{" a , b , c ", []string{"a", "b", "c"}},
		{`"a,b,c"`, []string{"a", "b", "c"}},
		{`a\,b,c`, []string{"a,b", "c"}},
	}

	for _, tt := range tests {
		got := splitCSV(tt.input)
		if len(got) != len(tt.expected) {
			t.Errorf("splitCSV(%q) = %v (len %d), want %v (len %d)", tt.input, got, len(got), tt.expected, len(tt.expected))
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("splitCSV(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.expected[i])
			}
		}
	}
}

func TestParseKeyValuePairs(t *testing.T) {
	tests := []struct {
		input    string
		expected map[string]string
	}{
		{"", nil},
		{"a=1,b=2", map[string]string{"a": "1", "b": "2"}},
		{"1/2=\u00BD,/=-", map[string]string{"1/2": "\u00BD", "/": "-"}},
	}

	for _, tt := range tests {
		got := parseKeyValuePairs(tt.input)
		if len(got) != len(tt.expected) {
			t.Errorf("parseKeyValuePairs(%q) = %v, want %v", tt.input, got, tt.expected)
			continue
		}
		for k, v := range tt.expected {
			if got[k] != v {
				t.Errorf("parseKeyValuePairs(%q)[%q] = %q, want %q", tt.input, k, got[k], v)
			}
		}
	}
}

func TestCleanerEnabled(t *testing.T) {
	cfg := &Config{
		Cleaners: []string{"movies", "tv"},
	}

	if !cfg.CleanerEnabled("movies") {
		t.Error("expected movies to be enabled")
	}
	if !cfg.CleanerEnabled("tv") {
		t.Error("expected tv to be enabled")
	}
	if cfg.CleanerEnabled("series") {
		t.Error("expected series to be disabled")
	}
}

func TestPathsAllDirs(t *testing.T) {
	cfg := &Config{DataDir: "/data"}
	paths := cfg.Paths()

	dirs := paths.AllDirs()
	if len(dirs) != 8 {
		t.Errorf("expected 8 directories, got %d", len(dirs))
	}

	// Verify all paths start with /data.
	for _, d := range dirs {
		if d[:5] != "/data" {
			t.Errorf("expected path to start with /data, got %s", d)
		}
	}
}
