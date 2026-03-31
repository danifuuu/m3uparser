package m3u

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/dani/m3uparser/internal/entry"
)

var (
	reKeyValue   = regexp.MustCompile(`(\w[\w-]*?)="(.*?)"`)
	reDuration   = regexp.MustCompile(`^#EXTINF:(-?\d*)\s`)
	reResolution = regexp.MustCompile(`\b(HD|SD)\s*`)
)

// ParseResult holds the output of parsing an M3U file.
type ParseResult struct {
	Entries []*entry.Entry
	Errors  []string
}

// ParseFile reads an M3U file and returns parsed entries.
// It applies value processing (scrub, replace) and classification.
func ParseFile(
	path string,
	cfg ParseConfig,
) (*ParseResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open M3U file: %w", err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	// Increase buffer for very long lines (e.g. long URLs).
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan M3U file: %w", err)
	}

	result := &ParseResult{}
	i := 0
	total := len(lines)

	for i < total {
		line := strings.TrimSpace(lines[i])

		if !strings.HasPrefix(line, "#EXTINF") {
			i++
			continue
		}

		e, parseErr := parseLine(line, lines, &i, total, cfg)
		if parseErr != "" {
			slog.Warn("parse error", "line", line, "error", parseErr)
			result.Errors = append(result.Errors, parseErr)
			i++
			continue
		}

		// Classify and clean.
		entry.ClassifyAndClean(e, cfg.RemoveTerms, cfg.RemoveDefaults, cfg.Cleaners)

		result.Entries = append(result.Entries, e)
		i++
	}

	slog.Info("parsed M3U file", "entries", len(result.Entries), "errors", len(result.Errors))
	return result, nil
}

// ParseConfig holds the processing parameters for parsing.
type ParseConfig struct {
	ScrubHeader     []string
	ScrubDefaults   []string
	ReplaceTerms    map[string]string
	ReplaceDefaults map[string]string
	RemoveTerms     []string
	RemoveDefaults  []string
	ExcludeTerms    []string
	Cleaners        entry.CleanerFlags
}

func parseLine(line string, lines []string, i *int, total int, cfg ParseConfig) (*entry.Entry, string) {
	e := &entry.Entry{}

	// Extract key=value pairs.
	kv := extractKeyValuePairs(line)

	e.GroupTitle = kv["group-title"]
	e.TvgID = kv["tvg-id"]
	e.TvgName = kv["tvg-name"]
	e.TvgLogo = kv["tvg-logo"]
	e.ExtInfLine = line

	// Duration.
	if m := reDuration.FindStringSubmatch(line); len(m) > 1 {
		e.Duration = m[1]
	}

	// Resolution.
	if m := reResolution.FindStringSubmatch(line); len(m) > 1 {
		e.Resolution = m[1]
	}

	// Check exclusion.
	if e.GroupTitle != "" && entry.CheckExcluded(e.GroupTitle, cfg.ExcludeTerms) {
		e.Excluded = true
	}

	// Process group-title: scrub headers, apply replacements.
	if e.GroupTitle != "" {
		e.GroupTitle = entry.ProcessValue(e.GroupTitle, nil, cfg.ScrubDefaults)
		e.GroupTitle = entry.ProcessValue(e.GroupTitle, nil, cfg.ScrubHeader)
		e.GroupTitle = entry.ProcessValue(e.GroupTitle, cfg.ReplaceDefaults, nil)
		e.GroupTitle = entry.ProcessValue(e.GroupTitle, cfg.ReplaceTerms, nil)
	}

	// Check for #EXTGRP on next line.
	if *i+1 < total && strings.HasPrefix(lines[*i+1], "#EXTGRP") {
		e.ExtGRP = strings.TrimSpace(lines[*i+1])
		*i++
	}

	// Next line should be the stream URL.
	if *i+1 < total && !strings.HasPrefix(lines[*i+1], "#EXTINF") {
		e.StreamURL = strings.TrimSpace(lines[*i+1])
		*i++
	}

	return e, ""
}

// extractKeyValuePairs parses key="value" pairs from an #EXTINF line.
// It also captures trailing text after the last quoted value as part of that key.
func extractKeyValuePairs(line string) map[string]string {
	result := make(map[string]string)

	matches := reKeyValue.FindAllStringSubmatchIndex(line, -1)
	var lastKey string
	var lastEnd int

	for i, loc := range matches {
		key := strings.TrimSpace(line[loc[2]:loc[3]])
		value := strings.TrimSpace(line[loc[4]:loc[5]])
		result[key] = value

		if i > 0 && lastKey != "" {
			// Append text between previous match end and this match start to previous key.
			between := strings.TrimSpace(line[lastEnd:loc[0]])
			if between != "" {
				result[lastKey] += between
			}
		}

		lastKey = key
		lastEnd = loc[1]
	}

	// Trailing text after last key-value pair.
	if lastKey != "" && lastEnd < len(line) {
		trailing := strings.TrimSpace(line[lastEnd:])
		if trailing != "" {
			result[lastKey] += trailing
		}
	}

	return result
}
