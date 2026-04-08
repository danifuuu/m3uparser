package entry

import (
	"fmt"
	"regexp"
	"strings"
)

// Regex patterns for classification.
// Go's regexp uses RE2 which does not support lookaheads/lookbehinds,
// so we use FindStringIndex and substring extraction instead.
var (
	// Season/episode: S01E02, 1x02, S01 E02
	reSeasonEpisode = regexp.MustCompile(`(?i)\b[sS]\d{1,3}[eE]\d{1,3}\b|\b\d{1,3}[xX]\d{1,3}\b|\b[sS]\d{1,3}\s[eE]\d{1,3}\b`)

	// Air date: "2024 01 15" or "15 01 2024" style
	reAirDate = regexp.MustCompile(`\b(?:(?:19|20)\d{2}\s\d{2}\s\d{2}|\d{2}\s\d{2}\s(?:19|20)\d{2})\b`)

	// Movie year: 19xx, 20xx, (19xx), (20xx)
	// Parenthesized years are matched first so they include the parens.
	reMovieYear = regexp.MustCompile(`(?:\((?:19[3-9]\d|20\d{2})\)|\b(?:19\d{2}|20\d{2})\b)`)

	// URL path patterns for Xtream Codes API-style IPTV services.
	// These match /movie/, /series/, or /live/ anywhere in the URL path.
	reURLMovie  = regexp.MustCompile(`/movie/`)
	reURLSeries = regexp.MustCompile(`/series/`)
	reURLLive   = regexp.MustCompile(`/live/`)

	// Extract season/episode from SxxExx format
	reSeasonFromSE  = regexp.MustCompile(`(?i)[sS](\d{1,3})[eE]\d{1,3}`)
	reEpisodeFromSE = regexp.MustCompile(`(?i)[sS]\d{1,3}[eE](\d{1,3})`)

	// Extract season/episode from NxN format
	reSeasonFromX  = regexp.MustCompile(`(?i)(\d{1,3})[xX]\d{1,3}`)
	reEpisodeFromX = regexp.MustCompile(`(?i)\d{1,3}[xX](\d{1,3})`)

	// Extract season/episode from "S01 E02" format
	reSeasonFromSpaced  = regexp.MustCompile(`(?i)[sS](\d{1,3})\s[eE]\d{1,3}`)
	reEpisodeFromSpaced = regexp.MustCompile(`(?i)[sS]\d{1,3}\s[eE](\d{1,3})`)
)

// CleanerFlags controls which entry categories have term removal applied.
type CleanerFlags struct {
	Movies   bool
	Series   bool
	TV       bool
	Unsorted bool
}

// extractSeasonEpisode extracts the season and episode numbers from a
// matched season/episode string like "S01E02", "3x14", or "S01 E02".
func extractSeasonEpisode(se string) (season, episode string) {
	// Try SxxExx format
	if m := reSeasonFromSE.FindStringSubmatch(se); len(m) > 1 {
		season = m[1]
	}
	if m := reEpisodeFromSE.FindStringSubmatch(se); len(m) > 1 {
		episode = m[1]
	}
	if season != "" && episode != "" {
		return
	}

	// Try NxN format
	if m := reSeasonFromX.FindStringSubmatch(se); len(m) > 1 {
		season = m[1]
	}
	if m := reEpisodeFromX.FindStringSubmatch(se); len(m) > 1 {
		episode = m[1]
	}
	if season != "" && episode != "" {
		return
	}

	// Try spaced format "S01 E02"
	if m := reSeasonFromSpaced.FindStringSubmatch(se); len(m) > 1 {
		season = m[1]
	}
	if m := reEpisodeFromSpaced.FindStringSubmatch(se); len(m) > 1 {
		episode = m[1]
	}
	return
}

// findLastMovieYear finds the last occurrence of a year pattern (19xx, 20xx,
// (19xx), (20xx)) in value and returns the match and its start index, or
// ("", -1) if not found.
func findLastMovieYear(value string) (match string, startIdx int) {
	matches := reMovieYear.FindAllStringIndex(value, -1)
	if len(matches) == 0 {
		return "", -1
	}
	last := matches[len(matches)-1]
	return value[last[0]:last[1]], last[0]
}

// ClassifyAndClean populates the entry's type, parsed metadata, and cleaned group-title.
// It applies the same logic as the Python clean_group_title function.
func ClassifyAndClean(e *Entry, removeTerms, removeDefaults []string, cleaners CleanerFlags) {
	value := e.GroupTitle

	defer func() {
		if r := recover(); r != nil {
			e.Error = fmt.Sprintf("panic during classification: %v", r)
		}
	}()

	// 1. Check for season/episode pattern -> Series
	if loc := reSeasonEpisode.FindStringIndex(value); loc != nil {
		match := value[loc[0]:loc[1]]
		e.SeasonEpisode = strings.TrimSpace(match)
		e.EntryType = TypeSeries

		// Title is everything before the season/episode match
		titleBefore := strings.TrimSpace(value[:loc[0]])
		if titleBefore != "" {
			showTitle := titleBefore
			if cleaners.Series {
				showTitle = RemoveAllTerms(showTitle, removeTerms, removeDefaults)
			}
			e.ShowTitle = showTitle
		}

		// Extract season and episode numbers
		season, episode := extractSeasonEpisode(e.SeasonEpisode)
		e.Season = season
		e.Episode = episode

		// Remove the season/episode from the remaining value
		value = strings.TrimSpace(value[:loc[0]]) + " " + strings.TrimSpace(value[loc[1]:])
		value = strings.TrimSpace(value)
		// Also remove the show title from what's left
		if e.ShowTitle != "" {
			value = strings.Replace(value, titleBefore, "", 1)
			value = strings.TrimSpace(value)
		}
	}

	// 2. Check for air date pattern -> TV show
	if e.EntryType == TypeUnknown {
		if airLoc := reAirDate.FindStringIndex(value); airLoc != nil {
			// Title is everything before the air date
			titleBefore := strings.TrimSpace(value[:airLoc[0]])
			if titleBefore != "" {
				e.ShowTitle = titleBefore
			}

			rawDate := strings.TrimSpace(value[airLoc[0]:airLoc[1]])
			e.AirDate = strings.Join(strings.Fields(rawDate), "-")
			e.EntryType = TypeTV

			// Everything after the air date is the guest star / remainder
			remainder := strings.TrimSpace(value[airLoc[1]:])
			if cleaners.TV {
				remainder = RemoveAllTerms(remainder, removeTerms, removeDefaults)
			}
			e.GuestStar = strings.TrimSpace(remainder)
			value = ""
		}
	}

	// 3. Check for movie (has year, no TV classification)
	if e.EntryType == TypeUnknown {
		yearStr, yearIdx := findLastMovieYear(value)
		if yearIdx >= 0 {
			// Title is everything before the last year occurrence
			movieTitle := strings.TrimSpace(value[:yearIdx])
			if movieTitle != "" {
				if cleaners.Movies {
					movieTitle = RemoveAllTerms(movieTitle, removeTerms, removeDefaults)
				}
				e.MovieTitle = movieTitle
				e.MovieDate = strings.TrimSpace(yearStr)
				e.EntryType = TypeMovie

				// Remove title and date from value
				value = strings.TrimSpace(value[yearIdx+len(yearStr):])
			}
		}
	}

	// 4. URL-based classification fallback: Xtream Codes API URLs contain
	// /movie/, /series/, or /live/ path segments. This catches entries that
	// lack title-based classification signals (e.g., movies without years).
	if e.EntryType == TypeUnknown && e.StreamURL != "" {
		if reURLMovie.MatchString(e.StreamURL) {
			movieTitle := strings.TrimSpace(value)
			if movieTitle == "" {
				movieTitle = strings.TrimSpace(e.GroupTitle)
			}
			if cleaners.Movies {
				movieTitle = RemoveAllTerms(movieTitle, removeTerms, removeDefaults)
			}
			e.MovieTitle = movieTitle
			e.EntryType = TypeMovie
			value = ""
		} else if reURLSeries.MatchString(e.StreamURL) {
			// Series URL but no season/episode in title — classify as unsorted
			// under the show title so it doesn't get lost.
			e.EntryType = TypeUnsorted
		} else if reURLLive.MatchString(e.StreamURL) {
			e.EntryType = TypeLiveTV
			ensureTvgID(e)
		}
	}

	// 5. Live TV: unclassified with duration == -1 and no URL hint
	if e.EntryType == TypeUnknown && e.Duration == "-1" {
		e.EntryType = TypeLiveTV
		ensureTvgID(e)
	}

	// 6. Unsorted: everything else
	if e.EntryType == TypeUnknown {
		e.EntryType = TypeUnsorted
	}

	e.GroupTitle = strings.TrimSpace(value)
}

// ensureTvgID derives a tvg-id from tvg-name + stream_url suffix if it's empty.
func ensureTvgID(e *Entry) {
	if e.TvgID != "" {
		return
	}

	derived := e.TvgName
	for _, ch := range []string{" ", ":", "(", ")", "/"} {
		derived = strings.ReplaceAll(derived, ch, ".")
	}
	derived = strings.ReplaceAll(derived, "..", ".")
	derived = strings.Trim(derived, ".")

	if len(e.StreamURL) >= 3 {
		derived += "." + e.StreamURL[len(e.StreamURL)-3:]
	}

	e.TvgID = derived

	if e.ExtInfLine != "" {
		re := regexp.MustCompile(`tvg-id="[^"]*"`)
		e.ExtInfLine = re.ReplaceAllString(e.ExtInfLine, fmt.Sprintf(`tvg-id="%s"`, derived))
	}
}
