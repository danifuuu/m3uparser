package m3u

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dani/m3uparser/internal/entry"
)

func TestCombineFiles(t *testing.T) {
	tmpDir := t.TempDir()
	m3uDir := filepath.Join(tmpDir, "m3u")
	os.MkdirAll(m3uDir, 0o755)

	// Write two M3U files.
	file1 := filepath.Join(m3uDir, "a.m3u")
	os.WriteFile(file1, []byte("#EXTM3U\n#EXTINF:-1,Channel 1\nhttp://a.com/1\n#EXTINF:-1,Channel 2\nhttp://a.com/2\n"), 0o644)

	file2 := filepath.Join(m3uDir, "b.m3u")
	os.WriteFile(file2, []byte("#EXTM3U\n#EXTINF:0,Movie 1\nhttp://b.com/1\n"), 0o644)

	outPath := filepath.Join(tmpDir, "combined.m3u")
	if err := CombineFiles(m3uDir, outPath); err != nil {
		t.Fatalf("CombineFiles failed: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read combined file: %v", err)
	}

	content := string(data)
	if content[:7] != "#EXTM3U" {
		t.Error("combined file should start with #EXTM3U")
	}

	// Should contain entries from both files.
	if !contains(content, "Channel 1") || !contains(content, "Movie 1") {
		t.Error("combined file should contain entries from both source files")
	}
}

func TestCombineFilesEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	m3uDir := filepath.Join(tmpDir, "m3u")
	os.MkdirAll(m3uDir, 0o755)

	outPath := filepath.Join(tmpDir, "combined.m3u")
	err := CombineFiles(m3uDir, outPath)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
}

func TestParseFile(t *testing.T) {
	tmpDir := t.TempDir()
	m3uFile := filepath.Join(tmpDir, "test.m3u")

	content := `#EXTM3U
#EXTINF:-1 tvg-id="cnn.us" tvg-name="CNN" group-title="News",CNN HD
http://example.com/live/cnn
#EXTINF:0 tvg-id="" tvg-name="Breaking Bad S01E01" group-title="Breaking Bad",Breaking Bad S01E01
http://example.com/vod/bb-s01e01
#EXTINF:0 tvg-id="" tvg-name="The Matrix (1999)" group-title="Movies",The Matrix (1999)
http://example.com/vod/matrix
`

	if err := os.WriteFile(m3uFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	cfg := ParseConfig{
		Cleaners: entry.CleanerFlags{},
	}

	result, err := ParseFile(m3uFile, cfg)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(result.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result.Entries))
	}

	// First entry: live TV.
	e0 := result.Entries[0]
	if e0.EntryType != entry.TypeLiveTV {
		t.Errorf("entry 0: expected TypeLiveTV, got %s", e0.EntryType)
	}
	if e0.TvgID != "cnn.us" {
		t.Errorf("entry 0: expected tvg-id 'cnn.us', got %q", e0.TvgID)
	}

	// Second entry: series.
	e1 := result.Entries[1]
	if e1.EntryType != entry.TypeSeries {
		t.Errorf("entry 1: expected TypeSeries, got %s", e1.EntryType)
	}
	if e1.StreamURL != "http://example.com/vod/bb-s01e01" {
		t.Errorf("entry 1: unexpected stream URL %q", e1.StreamURL)
	}

	// Third entry: movie.
	e2 := result.Entries[2]
	if e2.EntryType != entry.TypeMovie {
		t.Errorf("entry 2: expected TypeMovie, got %s", e2.EntryType)
	}
}

func TestParseFileWithExclusion(t *testing.T) {
	tmpDir := t.TempDir()
	m3uFile := filepath.Join(tmpDir, "test.m3u")

	content := `#EXTM3U
#EXTINF:-1 tvg-id="xxx" tvg-name="Adult" group-title="Adult XXX Content",Adult Channel
http://example.com/live/adult
#EXTINF:-1 tvg-id="cnn" tvg-name="CNN" group-title="News",CNN
http://example.com/live/cnn
`

	os.WriteFile(m3uFile, []byte(content), 0o644)

	cfg := ParseConfig{
		ExcludeTerms: []string{"XXX"},
		Cleaners:     entry.CleanerFlags{},
	}

	result, err := ParseFile(m3uFile, cfg)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(result.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result.Entries))
	}

	if !result.Entries[0].Excluded {
		t.Error("first entry should be excluded")
	}
	if result.Entries[1].Excluded {
		t.Error("second entry should not be excluded")
	}
}

func TestExtractKeyValuePairs(t *testing.T) {
	line := `#EXTINF:-1 tvg-id="cnn.us" tvg-name="CNN HD" tvg-logo="http://logo.com/cnn.png" group-title="News",CNN HD`

	kv := extractKeyValuePairs(line)

	if kv["tvg-id"] != "cnn.us" {
		t.Errorf("tvg-id = %q, want 'cnn.us'", kv["tvg-id"])
	}
	if kv["tvg-name"] != "CNN HD" {
		t.Errorf("tvg-name = %q, want 'CNN HD'", kv["tvg-name"])
	}
	if kv["tvg-logo"] != "http://logo.com/cnn.png" {
		t.Errorf("tvg-logo = %q, want 'http://logo.com/cnn.png'", kv["tvg-logo"])
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
