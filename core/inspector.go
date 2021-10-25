package core

type HLSInspector interface {
	Inspect(playlists *Playlists, segments SegmentStore) *Report
}

type DASHInspector interface {
	Inspect(manifest *Manifest, segments SegmentStore) *Report
}
