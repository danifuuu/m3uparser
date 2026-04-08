package strm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dani/m3uparser/internal/entry"
)

func TestWriteSeriesEntry(t *testing.T) {
	tmpDir := t.TempDir()
	tvDir := filepath.Join(tmpDir, "TV_VOD")

	entries := []*entry.Entry{
		{
			EntryType:     entry.TypeSeries,
			ShowTitle:     "Breaking Bad",
			Season:        "01",
			SeasonEpisode: "S01E01",
			StreamURL:     "http://example.com/bb-s01e01",
		},
	}

	stats, errors := WriteAll(entries, tvDir, tmpDir, tmpDir)

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	if stats.TV != 1 {
		t.Errorf("expected 1 TV entry, got %d", stats.TV)
	}

	expected := filepath.Join(tvDir, "Breaking Bad", "Season 01", "Breaking Bad S01E01.strm")
	data, err := os.ReadFile(expected)
	if err != nil {
		t.Fatalf("expected file %s to exist: %v", expected, err)
	}
	if string(data) != "http://example.com/bb-s01e01" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestWriteMovieEntry(t *testing.T) {
	tmpDir := t.TempDir()
	moviesDir := filepath.Join(tmpDir, "Movie_VOD")

	entries := []*entry.Entry{
		{
			EntryType:  entry.TypeMovie,
			MovieTitle: "The Matrix",
			MovieDate:  "1999",
			StreamURL:  "http://example.com/matrix",
		},
	}

	stats, errors := WriteAll(entries, tmpDir, moviesDir, tmpDir)

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	if stats.Movies != 1 {
		t.Errorf("expected 1 movie entry, got %d", stats.Movies)
	}

	expected := filepath.Join(moviesDir, "The Matrix (1999)", "The Matrix (1999).strm")
	data, err := os.ReadFile(expected)
	if err != nil {
		t.Fatalf("expected file %s to exist: %v", expected, err)
	}
	if string(data) != "http://example.com/matrix" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestWriteMovieEntryWithoutYear(t *testing.T) {
	tmpDir := t.TempDir()
	moviesDir := filepath.Join(tmpDir, "Movie_VOD")

	entries := []*entry.Entry{
		{
			EntryType:  entry.TypeMovie,
			MovieTitle: "Hannah Montana: Especial 20 aniversario",
			MovieDate:  "",
			StreamURL:  "http://example.com/hannah",
		},
	}

	stats, errors := WriteAll(entries, tmpDir, moviesDir, tmpDir)

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	if stats.Movies != 1 {
		t.Errorf("expected 1 movie entry, got %d", stats.Movies)
	}

	// Should NOT have empty parens — just "Title/Title.strm"
	expected := filepath.Join(moviesDir, "Hannah Montana: Especial 20 aniversario", "Hannah Montana: Especial 20 aniversario.strm")
	data, err := os.ReadFile(expected)
	if err != nil {
		t.Fatalf("expected file %s to exist: %v", expected, err)
	}
	if string(data) != "http://example.com/hannah" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestWriteLiveTVPlaylist(t *testing.T) {
	tmpDir := t.TempDir()
	liveTVFile := filepath.Join(tmpDir, "livetv.m3u")

	entries := []*entry.Entry{
		{
			EntryType:  entry.TypeLiveTV,
			ExtInfLine: `#EXTINF:-1 tvg-id="cnn" tvg-name="CNN",CNN`,
			StreamURL:  "http://example.com/live/cnn",
		},
		{
			EntryType:  entry.TypeLiveTV,
			ExtInfLine: `#EXTINF:-1 tvg-id="bbc" tvg-name="BBC",BBC`,
			StreamURL:  "http://example.com/live/bbc",
			ExtGRP:     "#EXTGRP:News",
		},
		{
			// Non-live entry, should be skipped.
			EntryType: entry.TypeMovie,
			StreamURL: "http://example.com/movie",
		},
	}

	count, err := WriteLiveTVPlaylist(entries, liveTVFile)
	if err != nil {
		t.Fatalf("WriteLiveTVPlaylist failed: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 live TV entries, got %d", count)
	}

	data, err := os.ReadFile(liveTVFile)
	if err != nil {
		t.Fatalf("read live TV file: %v", err)
	}

	content := string(data)
	if content[:7] != "#EXTM3U" {
		t.Error("live TV file should start with #EXTM3U")
	}

	if !containsStr(content, "cnn") || !containsStr(content, "bbc") {
		t.Error("live TV file should contain both channels")
	}

	if !containsStr(content, "#EXTGRP:News") {
		t.Error("live TV file should contain EXTGRP line")
	}
}

func TestWriteExcludedEntries(t *testing.T) {
	tmpDir := t.TempDir()

	entries := []*entry.Entry{
		{
			EntryType:  entry.TypeMovie,
			MovieTitle: "Some Movie",
			MovieDate:  "2020",
			StreamURL:  "http://example.com/movie",
			Excluded:   true,
		},
	}

	stats, _ := WriteAll(entries, tmpDir, tmpDir, tmpDir)

	if stats.Movies != 0 {
		t.Errorf("excluded entries should not be counted, got %d", stats.Movies)
	}
}

func TestSyncDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src")
	dest := filepath.Join(tmpDir, "dest")

	os.MkdirAll(filepath.Join(src, "subdir"), 0o755)
	os.WriteFile(filepath.Join(src, "a.strm"), []byte("url1"), 0o644)
	os.WriteFile(filepath.Join(src, "subdir", "b.strm"), []byte("url2"), 0o644)

	if err := SyncDirectories(src, dest, false); err != nil {
		t.Fatalf("SyncDirectories failed: %v", err)
	}

	// Verify files exist in dest.
	if _, err := os.Stat(filepath.Join(dest, "a.strm")); err != nil {
		t.Error("expected a.strm to be synced")
	}
	if _, err := os.Stat(filepath.Join(dest, "subdir", "b.strm")); err != nil {
		t.Error("expected subdir/b.strm to be synced")
	}
}

func TestSyncDirectoriesWithRemoval(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src")
	dest := filepath.Join(tmpDir, "dest")

	os.MkdirAll(src, 0o755)
	os.MkdirAll(dest, 0o755)
	os.WriteFile(filepath.Join(src, "keep.strm"), []byte("url"), 0o644)
	os.WriteFile(filepath.Join(dest, "keep.strm"), []byte("url"), 0o644)
	os.WriteFile(filepath.Join(dest, "remove.strm"), []byte("old"), 0o644)

	if err := SyncDirectories(src, dest, true); err != nil {
		t.Fatalf("SyncDirectories failed: %v", err)
	}

	// keep.strm should exist.
	if _, err := os.Stat(filepath.Join(dest, "keep.strm")); err != nil {
		t.Error("expected keep.strm to remain")
	}

	// remove.strm should be gone.
	if _, err := os.Stat(filepath.Join(dest, "remove.strm")); !os.IsNotExist(err) {
		t.Error("expected remove.strm to be removed")
	}
}

func TestCountStrmFiles(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "sub"), 0o755)

	os.WriteFile(filepath.Join(tmpDir, "a.strm"), []byte("1"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "b.strm"), []byte("2"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "sub", "c.strm"), []byte("3"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "d.txt"), []byte("not strm"), 0o644)

	count := CountStrmFiles(tmpDir)
	if count != 3 {
		t.Errorf("expected 3 .strm files, got %d", count)
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello/World", "Hello-World"},
		{"File: Name", "File: Name"},
		{`Star*War?s`, "StarWars"},
		{"Normal Name", "Normal Name"},
		{"  spaces  ", "spaces"},
	}

	for _, tt := range tests {
		got := sanitizePath(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizePath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
