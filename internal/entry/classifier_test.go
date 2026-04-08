package entry

import (
	"testing"
)

func TestClassifySeries(t *testing.T) {
	e := &Entry{
		GroupTitle: "Breaking Bad S01E01 Pilot",
		Duration:   "0",
		StreamURL:  "http://example.com/stream",
	}

	ClassifyAndClean(e, nil, nil, CleanerFlags{})

	if e.EntryType != TypeSeries {
		t.Errorf("expected TypeSeries, got %s", e.EntryType)
	}
	if e.Season != "01" {
		t.Errorf("expected season 01, got %q", e.Season)
	}
	if e.Episode != "01" {
		t.Errorf("expected episode 01, got %q", e.Episode)
	}
	if e.SeasonEpisode != "S01E01" {
		t.Errorf("expected SeasonEpisode S01E01, got %q", e.SeasonEpisode)
	}
	if e.ShowTitle != "Breaking Bad" {
		t.Errorf("expected ShowTitle 'Breaking Bad', got %q", e.ShowTitle)
	}
}

func TestClassifySeriesAltFormat(t *testing.T) {
	e := &Entry{
		GroupTitle: "Friends 3x14 The One With Phoebes Ex",
		Duration:   "0",
		StreamURL:  "http://example.com/stream",
	}

	ClassifyAndClean(e, nil, nil, CleanerFlags{})

	if e.EntryType != TypeSeries {
		t.Errorf("expected TypeSeries, got %s", e.EntryType)
	}
	if e.Season != "3" {
		t.Errorf("expected season 3, got %q", e.Season)
	}
	if e.Episode != "14" {
		t.Errorf("expected episode 14, got %q", e.Episode)
	}
}

func TestClassifyMovie(t *testing.T) {
	e := &Entry{
		GroupTitle: "The Matrix 1999",
		Duration:   "0",
		StreamURL:  "http://example.com/stream",
	}

	ClassifyAndClean(e, nil, nil, CleanerFlags{})

	if e.EntryType != TypeMovie {
		t.Errorf("expected TypeMovie, got %s", e.EntryType)
	}
	if e.MovieTitle != "The Matrix" {
		t.Errorf("expected MovieTitle 'The Matrix', got %q", e.MovieTitle)
	}
	if e.MovieDate != "1999" {
		t.Errorf("expected MovieDate 1999, got %q", e.MovieDate)
	}
}

func TestClassifyMovieWithParens(t *testing.T) {
	e := &Entry{
		GroupTitle: "Inception (2010)",
		Duration:   "0",
		StreamURL:  "http://example.com/stream",
	}

	ClassifyAndClean(e, nil, nil, CleanerFlags{})

	if e.EntryType != TypeMovie {
		t.Errorf("expected TypeMovie, got %s", e.EntryType)
	}
	if e.MovieDate != "(2010)" {
		t.Errorf("expected MovieDate (2010), got %q", e.MovieDate)
	}
}

func TestClassifyLiveTV(t *testing.T) {
	e := &Entry{
		GroupTitle: "CNN HD",
		Duration:   "-1",
		StreamURL:  "http://example.com/live/cnn",
		TvgID:      "cnn.us",
		TvgName:    "CNN",
	}

	ClassifyAndClean(e, nil, nil, CleanerFlags{})

	if e.EntryType != TypeLiveTV {
		t.Errorf("expected TypeLiveTV, got %s", e.EntryType)
	}
}

func TestClassifyLiveTVDerivesID(t *testing.T) {
	e := &Entry{
		GroupTitle: "ESPN",
		Duration:   "-1",
		StreamURL:  "http://example.com/live/espn/123",
		TvgID:      "",
		TvgName:    "ESPN HD",
		ExtInfLine: `#EXTINF:-1 tvg-id="" tvg-name="ESPN HD",ESPN`,
	}

	ClassifyAndClean(e, nil, nil, CleanerFlags{})

	if e.EntryType != TypeLiveTV {
		t.Errorf("expected TypeLiveTV, got %s", e.EntryType)
	}
	if e.TvgID != "ESPN.HD.123" {
		t.Errorf("expected derived TvgID 'ESPN.HD.123', got %q", e.TvgID)
	}
	expected := `#EXTINF:-1 tvg-id="ESPN.HD.123" tvg-name="ESPN HD",ESPN`
	if e.ExtInfLine != expected {
		t.Errorf("expected ExtInfLine %q, got %q", expected, e.ExtInfLine)
	}
}

func TestClassifyUnsorted(t *testing.T) {
	e := &Entry{
		GroupTitle: "Random Channel",
		Duration:   "0",
		StreamURL:  "http://example.com/stream",
	}

	ClassifyAndClean(e, nil, nil, CleanerFlags{})

	if e.EntryType != TypeUnsorted {
		t.Errorf("expected TypeUnsorted, got %s", e.EntryType)
	}
}

func TestClassifyWithRemoveTerms(t *testing.T) {
	e := &Entry{
		GroupTitle: "The Office S02E03 720p HDTV",
		Duration:   "0",
		StreamURL:  "http://example.com/stream",
	}

	ClassifyAndClean(e, []string{"720p", "HDTV"}, nil, CleanerFlags{Series: true})

	if e.EntryType != TypeSeries {
		t.Errorf("expected TypeSeries, got %s", e.EntryType)
	}
	if e.ShowTitle != "The Office" {
		t.Errorf("expected ShowTitle 'The Office', got %q", e.ShowTitle)
	}
}

func TestClassifyTVShowWithAirDate(t *testing.T) {
	e := &Entry{
		GroupTitle: "The Tonight Show 2024 03 15 Guest Name",
		Duration:   "0",
		StreamURL:  "http://example.com/stream",
	}

	ClassifyAndClean(e, nil, nil, CleanerFlags{})

	if e.EntryType != TypeTV {
		t.Errorf("expected TypeTV, got %s", e.EntryType)
	}
	if e.AirDate != "2024-03-15" {
		t.Errorf("expected AirDate '2024-03-15', got %q", e.AirDate)
	}
	if e.ShowTitle != "The Tonight Show" {
		t.Errorf("expected ShowTitle 'The Tonight Show', got %q", e.ShowTitle)
	}
}

func TestClassifyMovieByURL(t *testing.T) {
	// Movie without a year in the title, but URL contains /movie/
	e := &Entry{
		GroupTitle: "Hannah Montana: Especial 20 aniversario",
		Duration:   "-1",
		StreamURL:  "https://example.com/movie/user/pass/7c32cec4.mkv",
	}

	ClassifyAndClean(e, nil, nil, CleanerFlags{})

	if e.EntryType != TypeMovie {
		t.Errorf("expected TypeMovie, got %s", e.EntryType)
	}
	if e.MovieTitle != "Hannah Montana: Especial 20 aniversario" {
		t.Errorf("expected MovieTitle 'Hannah Montana: Especial 20 aniversario', got %q", e.MovieTitle)
	}
	if e.MovieDate != "" {
		t.Errorf("expected empty MovieDate, got %q", e.MovieDate)
	}
}

func TestClassifyMovieByURLWithCleaners(t *testing.T) {
	// Movie URL without year, with remove terms applied
	e := &Entry{
		GroupTitle: "Peaky Blinders: El hombre inmortal 720p",
		Duration:   "-1",
		StreamURL:  "https://example.com/movie/user/pass/6543649c.mkv",
	}

	ClassifyAndClean(e, []string{"720p"}, nil, CleanerFlags{Movies: true})

	if e.EntryType != TypeMovie {
		t.Errorf("expected TypeMovie, got %s", e.EntryType)
	}
	if e.MovieTitle != "Peaky Blinders: El hombre inmortal" {
		t.Errorf("expected MovieTitle 'Peaky Blinders: El hombre inmortal', got %q", e.MovieTitle)
	}
}

func TestClassifySeriesByURL(t *testing.T) {
	// Series URL but no season/episode in title → falls to unsorted
	e := &Entry{
		GroupTitle: "Some Series Name",
		Duration:   "-1",
		StreamURL:  "https://example.com/series/user/pass/12345.ts",
	}

	ClassifyAndClean(e, nil, nil, CleanerFlags{})

	if e.EntryType != TypeUnsorted {
		t.Errorf("expected TypeUnsorted for series URL without season/episode, got %s", e.EntryType)
	}
}

func TestClassifyLiveByURL(t *testing.T) {
	// Live URL without duration=-1
	e := &Entry{
		GroupTitle: "CNN HD",
		Duration:   "0",
		StreamURL:  "https://example.com/live/user/pass/456.ts",
		TvgName:    "CNN HD",
	}

	ClassifyAndClean(e, nil, nil, CleanerFlags{})

	if e.EntryType != TypeLiveTV {
		t.Errorf("expected TypeLiveTV, got %s", e.EntryType)
	}
}

func TestClassifyMovieWithYearStillUsesTitle(t *testing.T) {
	// Movie with year in title — title-based classification should win
	// even if URL also contains /movie/
	e := &Entry{
		GroupTitle: "Inception (2010)",
		Duration:   "-1",
		StreamURL:  "https://example.com/movie/user/pass/abc123.mkv",
	}

	ClassifyAndClean(e, nil, nil, CleanerFlags{})

	if e.EntryType != TypeMovie {
		t.Errorf("expected TypeMovie, got %s", e.EntryType)
	}
	if e.MovieDate != "(2010)" {
		t.Errorf("expected MovieDate '(2010)', got %q", e.MovieDate)
	}
	if e.MovieTitle != "Inception" {
		t.Errorf("expected MovieTitle 'Inception', got %q", e.MovieTitle)
	}
}

func TestClassifyLiveTVDurationFallback(t *testing.T) {
	// Non-Xtream URL with duration=-1 should still classify as LiveTV
	e := &Entry{
		GroupTitle: "Some Channel",
		Duration:   "-1",
		StreamURL:  "http://example.com/stream/channel123",
	}

	ClassifyAndClean(e, nil, nil, CleanerFlags{})

	if e.EntryType != TypeLiveTV {
		t.Errorf("expected TypeLiveTV for duration=-1 fallback, got %s", e.EntryType)
	}
}
