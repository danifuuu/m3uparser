// Package strm handles writing .strm files and syncing output directories.
package strm

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/dani/m3uparser/internal/entry"
)

// Stats tracks counts of created .strm files.
type Stats struct {
	TV       int
	Movies   int
	Unsorted int
	LiveTV   int
	Errors   int
}

// WriteAll processes all entries and writes .strm files to the appropriate directories.
func WriteAll(entries []*entry.Entry, tvDir, moviesDir, unsortedDir string) (*Stats, []string) {
	stats := &Stats{}
	var errors []string

	for _, e := range entries {
		if e.Excluded {
			continue
		}

		path, err := writeEntry(e, tvDir, moviesDir, unsortedDir)
		if err != nil {
			msg := fmt.Sprintf("error writing entry %q: %v", e.GroupTitle, err)
			slog.Error(msg)
			errors = append(errors, msg)
			stats.Errors++
			continue
		}

		if path == "" {
			continue // livetv entries are handled separately
		}

		switch e.EntryType {
		case entry.TypeSeries, entry.TypeTV:
			stats.TV++
		case entry.TypeMovie:
			stats.Movies++
		case entry.TypeUnsorted:
			stats.Unsorted++
		}
	}

	return stats, errors
}

func writeEntry(e *entry.Entry, tvDir, moviesDir, unsortedDir string) (string, error) {
	switch e.EntryType {
	case entry.TypeSeries:
		return writeSeries(e, tvDir)
	case entry.TypeTV:
		return writeTV(e, tvDir)
	case entry.TypeMovie:
		return writeMovie(e, moviesDir)
	case entry.TypeUnsorted:
		return writeUnsorted(e, unsortedDir)
	case entry.TypeLiveTV:
		return "", nil // handled by WriteLiveTVPlaylist
	default:
		return "", nil
	}
}

func writeSeries(e *entry.Entry, tvDir string) (string, error) {
	showTitle := sanitizePath(e.ShowTitle)
	season := e.Season
	if season == "" {
		season = "0"
	}
	dir := filepath.Join(tvDir, showTitle, fmt.Sprintf("Season %s", season))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}

	filename := fmt.Sprintf("%s %s.strm", showTitle, e.SeasonEpisode)
	path := filepath.Join(dir, sanitizePath(filename))
	return path, writeFile(path, e.StreamURL)
}

func writeTV(e *entry.Entry, tvDir string) (string, error) {
	showTitle := sanitizePath(e.ShowTitle)
	dir := filepath.Join(tvDir, showTitle)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}

	parts := []string{showTitle}
	if e.GuestStar != "" {
		parts = append(parts, e.GuestStar)
	}
	parts = append(parts, e.AirDate)

	filename := fmt.Sprintf("%s.strm", strings.Join(parts, " "))
	path := filepath.Join(dir, sanitizePath(filename))
	return path, writeFile(path, e.StreamURL)
}

func writeMovie(e *entry.Entry, moviesDir string) (string, error) {
	movieTitle := sanitizePath(e.MovieTitle)
	movieDate := e.MovieDate
	dirName := fmt.Sprintf("%s (%s)", movieTitle, movieDate)
	dir := filepath.Join(moviesDir, dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}

	filename := fmt.Sprintf("%s (%s).strm", movieTitle, movieDate)
	path := filepath.Join(dir, sanitizePath(filename))
	return path, writeFile(path, e.StreamURL)
}

func writeUnsorted(e *entry.Entry, unsortedDir string) (string, error) {
	groupTitle := sanitizePath(e.GroupTitle)
	if groupTitle == "" {
		groupTitle = "Unknown"
	}
	dir := filepath.Join(unsortedDir, groupTitle)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}

	filename := fmt.Sprintf("%s.strm", groupTitle)
	path := filepath.Join(dir, sanitizePath(filename))
	return path, writeFile(path, e.StreamURL)
}

// WriteLiveTVPlaylist writes the live TV entries as an M3U playlist.
func WriteLiveTVPlaylist(entries []*entry.Entry, liveTVFile string) (int, error) {
	f, err := os.Create(liveTVFile)
	if err != nil {
		return 0, fmt.Errorf("create livetv file: %w", err)
	}
	defer f.Close()

	fmt.Fprintln(f, "#EXTM3U")

	count := 0
	for _, e := range entries {
		if e.EntryType != entry.TypeLiveTV || e.Excluded {
			continue
		}

		fmt.Fprintln(f, e.ExtInfLine)
		if e.ExtGRP != "" {
			fmt.Fprintln(f, e.ExtGRP)
		}
		fmt.Fprintln(f, e.StreamURL)
		count++
	}

	slog.Info("wrote live TV playlist", "channels", count, "file", liveTVFile)
	return count, nil
}

// SyncDirectories copies files from src to dest, optionally removing dest items not in src.
func SyncDirectories(src, dest string, removeExtra bool) error {
	srcEntries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read src dir %s: %w", src, err)
	}

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("mkdir dest %s: %w", dest, err)
	}

	for _, de := range srcEntries {
		srcPath := filepath.Join(src, de.Name())
		destPath := filepath.Join(dest, de.Name())

		if de.IsDir() {
			if err := SyncDirectories(srcPath, destPath, removeExtra); err != nil {
				return err
			}
			continue
		}

		if err := copyFileIfChanged(srcPath, destPath); err != nil {
			return err
		}
	}

	if !removeExtra {
		return nil
	}

	// Remove items in dest that are not in src.
	destEntries, err := os.ReadDir(dest)
	if err != nil {
		return nil // dest may not exist yet
	}

	srcNames := make(map[string]bool, len(srcEntries))
	for _, de := range srcEntries {
		srcNames[de.Name()] = true
	}

	for _, de := range destEntries {
		if srcNames[de.Name()] {
			continue
		}
		p := filepath.Join(dest, de.Name())
		if de.IsDir() {
			if err := os.RemoveAll(p); err != nil {
				slog.Warn("failed to remove extra dir", "path", p, "error", err)
			} else {
				slog.Info("removed extra directory", "path", p)
			}
		} else {
			if err := os.Remove(p); err != nil {
				slog.Warn("failed to remove extra file", "path", p, "error", err)
			} else {
				slog.Info("removed extra file", "path", p)
			}
		}
	}

	return nil
}

// MoveFile moves a file to the destination directory.
func MoveFile(src, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", destDir, err)
	}

	destPath := filepath.Join(destDir, filepath.Base(src))

	// Remove existing file at destination.
	if _, err := os.Stat(destPath); err == nil {
		os.Remove(destPath)
	}

	if err := os.Rename(src, destPath); err != nil {
		return fmt.Errorf("move %s -> %s: %w", src, destPath, err)
	}

	slog.Info("moved file", "from", src, "to", destPath)
	return nil
}

// CountStrmFiles counts .strm files recursively in a directory.
func CountStrmFiles(dir string) int {
	count := 0
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".strm") {
			count++
		}
		return nil
	})
	return count
}

// Cleanup removes intermediate files and directories after processing.
func Cleanup(paths CleanupPaths) {
	for _, f := range paths.Files {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			slog.Warn("failed to remove file", "path", f, "error", err)
		} else if err == nil {
			slog.Debug("removed file", "path", f)
		}
	}

	for _, d := range paths.Dirs {
		if err := os.RemoveAll(d); err != nil {
			slog.Warn("failed to remove dir", "path", d, "error", err)
		} else {
			slog.Debug("removed directory", "path", d)
		}
	}

	// Clean m3u dir contents but keep the directory.
	if paths.M3UDir != "" {
		entries, _ := os.ReadDir(paths.M3UDir)
		for _, de := range entries {
			p := filepath.Join(paths.M3UDir, de.Name())
			os.RemoveAll(p)
		}
	}
}

// CleanupPaths defines what to remove during cleanup.
type CleanupPaths struct {
	Files  []string
	Dirs   []string
	M3UDir string
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func copyFileIfChanged(src, dest string) error {
	srcData, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}

	destData, _ := os.ReadFile(dest)
	if string(srcData) == string(destData) {
		return nil // no change
	}

	destF, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	defer destF.Close()

	srcF, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer srcF.Close()

	if _, err := io.Copy(destF, srcF); err != nil {
		return fmt.Errorf("copy %s -> %s: %w", src, dest, err)
	}

	action := "added"
	if len(destData) > 0 {
		action = "updated"
	}
	slog.Info("synced file", "action", action, "path", dest)
	return nil
}

// sanitizePath replaces characters that are problematic in file/dir names.
func sanitizePath(s string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "",
	)
	return strings.TrimSpace(replacer.Replace(s))
}
