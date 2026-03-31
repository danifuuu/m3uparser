// Package m3u handles downloading and parsing M3U playlist files.
package m3u

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"

// Download fetches all M3U URLs into the m3uDir.
// If bypassHeader is true it skips the HEAD check and downloads directly.
func Download(urls []string, m3uDir string, bypassHeader bool) error {
	client := &http.Client{}

	for _, u := range urls {
		if err := downloadOne(client, u, m3uDir, bypassHeader); err != nil {
			slog.Error("failed to download M3U", "url", u, "error", err)
			// Continue with remaining URLs instead of failing entirely.
		}
	}

	return nil
}

func downloadOne(client *http.Client, url, m3uDir string, bypassHeader bool) error {
	if bypassHeader {
		slog.Info("downloading (bypass header)", "url", url)
		return doGet(client, url, m3uDir)
	}

	// Default: HEAD check first.
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return fmt.Errorf("create HEAD request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HEAD request: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HEAD returned status %d", resp.StatusCode)
	}

	cd := resp.Header.Get("Content-Disposition")
	ct := resp.Header.Get("Content-Type")
	if ct == "" || !strings.Contains(cd, "filename=") {
		slog.Warn("URL missing Content-Type or filename, skipping", "url", url)
		return nil
	}

	return doGet(client, url, m3uDir)
}

func doGet(client *http.Client, url, m3uDir string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create GET request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET returned status %d", resp.StatusCode)
	}

	filename := filepath.Base(url)
	if filename == "" || filename == "/" {
		filename = "playlist.m3u"
	}

	dst := filepath.Join(m3uDir, filename)
	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create file %s: %w", dst, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write file %s: %w", dst, err)
	}

	slog.Info("downloaded M3U", "url", url, "dest", dst)
	return nil
}

// CombineFiles merges all downloaded M3U files from m3uDir into a single file at outPath.
// It skips the first line (#EXTM3U header) of each source file.
// Returns an error if the resulting file has no entries.
func CombineFiles(m3uDir, outPath string) error {
	entries, err := os.ReadDir(m3uDir)
	if err != nil {
		return fmt.Errorf("read m3u dir: %w", err)
	}

	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create combined file: %w", err)
	}
	defer out.Close()

	fmt.Fprintln(out, "#EXTM3U")

	totalLines := 0
	for _, de := range entries {
		if de.IsDir() {
			continue
		}
		fp := filepath.Join(m3uDir, de.Name())

		data, err := os.ReadFile(fp)
		if err != nil {
			slog.Error("failed to read M3U file", "path", fp, "error", err)
			continue
		}

		lines := strings.Split(string(data), "\n")
		// Skip header line and files with < 2 content lines.
		if len(lines) > 1 {
			content := lines[1:]
			if len(content) < 2 {
				continue
			}
			fmt.Fprintln(out)
			for _, l := range content {
				fmt.Fprintln(out, l)
			}
			totalLines += len(content)
		}
	}

	if totalLines == 0 {
		return fmt.Errorf("combined M3U file %s has no entries, aborting", outPath)
	}

	slog.Info("combined M3U files", "output", outPath, "lines", totalLines)
	return nil
}
