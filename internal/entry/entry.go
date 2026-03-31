// Package entry defines the M3U entry data model and its classification.
package entry

// Type represents the classification of an M3U entry.
type Type int

const (
	TypeUnknown  Type = iota
	TypeSeries        // TV series with season/episode info
	TypeTV            // TV show with air date
	TypeMovie         // Movie with year
	TypeLiveTV        // Live TV channel (duration == -1)
	TypeUnsorted      // Does not match any other category
)

func (t Type) String() string {
	switch t {
	case TypeSeries:
		return "series"
	case TypeTV:
		return "tv"
	case TypeMovie:
		return "movie"
	case TypeLiveTV:
		return "livetv"
	case TypeUnsorted:
		return "unsorted"
	default:
		return "unknown"
	}
}

// Entry represents a parsed and classified M3U entry.
type Entry struct {
	// Raw M3U fields.
	ExtInfLine string
	Duration   string
	GroupTitle string
	TvgID      string
	TvgName    string
	TvgLogo    string
	StreamURL  string
	ExtGRP     string
	Resolution string

	// Parsed metadata (populated by classifier/cleaner).
	EntryType     Type
	ShowTitle     string // series or TV show name
	Season        string
	Episode       string
	SeasonEpisode string // raw match, e.g. "S01E02"
	AirDate       string // formatted as YYYY-MM-DD
	GuestStar     string
	MovieTitle    string
	MovieDate     string

	// Flags.
	Excluded bool
	Error    string
}
