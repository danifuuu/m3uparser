// DiffCollector accumulates added/removed media items across multiple sync operations.
package notify

import (
	"path/filepath"
	"strings"
)

// DiffCollector gathers media changes during a sync run.
type DiffCollector struct {
	added   []MediaItem
	removed []MediaItem
}

// RecordAdded records a newly written .strm file path.
func (d *DiffCollector) RecordAdded(path, mediaType string) {
	title := titleFromPath(path)
	d.added = append(d.added, MediaItem{Title: title, Type: mediaType})
}

// RecordRemoved records a removed .strm file or directory path.
func (d *DiffCollector) RecordRemoved(path, mediaType string) {
	title := titleFromPath(path)
	d.removed = append(d.removed, MediaItem{Title: title, Type: mediaType})
}

// Build returns the accumulated Payload.
func (d *DiffCollector) Build(errors int) Payload {
	return Payload{
		Added:   d.added,
		Removed: d.removed,
		Errors:  errors,
	}
}

// titleFromPath extracts a human-readable title from a .strm file path.
// e.g. "/data/VODS/TV_VOD/Game of Thrones/Season 8/Game of Thrones S08E01.strm"
// becomes "Game of Thrones S08E01"
func titleFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, ".strm")
}
