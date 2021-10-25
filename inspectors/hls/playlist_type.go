package hls

import (
	"github.com/abema/antares/core"
	"github.com/grafov/m3u8"
)

type PlaylistTypeCondition int

const (
	PlaylistTypeAny PlaylistTypeCondition = iota
	PlaylistTypeMustOmitted
	PlaylistTypeMustEvent
	PlaylistTypeMustVOD
)

type EndlistCondition int

const (
	EndlistAny EndlistCondition = iota
	EndlistMustExist
	EndlistMustNotExist
)

type PlaylistTypeInspectorConfig struct {
	PlaylistTypeCondition
	EndlistCondition
}

// NewPlaylistTypeInspector returns PlaylistTypeInspector.
// It inspects EXT-X-PLAYLIST-TYPE tag.
func NewPlaylistTypeInspector(config *PlaylistTypeInspectorConfig) core.HLSInspector {
	return &playlistTypeInspector{
		config: config,
	}
}

type playlistTypeInspector struct {
	config *PlaylistTypeInspectorConfig
}

func (ins *playlistTypeInspector) Inspect(playlists *core.Playlists, segments core.SegmentStore) *core.Report {
	var noType bool
	var event bool
	var vod bool
	var endlist bool
	var noEndlist bool
	for _, media := range playlists.MediaPlaylists {
		switch media.MediaType {
		case 0:
			noType = true
		case m3u8.EVENT:
			event = true
		case m3u8.VOD:
			vod = true
		}
		if media.Closed {
			endlist = true
		} else {
			noEndlist = true
		}
	}

	values := make(core.Values, 2)
	if (noType && event) || (noType && vod) || (event && vod) {
		values["playlistType"] = "mixed"
	} else if noType {
		values["playlistType"] = "not exists"
	} else if event {
		values["playlistType"] = "EVENT"
	} else if vod {
		values["playlistType"] = "VOD"
	} else {
		values["playlistType"] = "n/a"
	}
	if endlist && noEndlist {
		values["endlist"] = "mixed"
	} else if endlist {
		values["endlist"] = "exists"
	} else if noEndlist {
		values["endlist"] = "not exists"
	} else {
		values["endlist"] = "n/a"
	}

	switch ins.config.PlaylistTypeCondition {
	case PlaylistTypeMustOmitted:
		if !noType || event || vod {
			return &core.Report{
				Name:     "PlaylistTypeInspector",
				Severity: core.Error,
				Message:  "PLAYLIST-TYPE must be omitted",
				Values:   values,
			}
		}
	case PlaylistTypeMustEvent:
		if noType || !event || vod {
			return &core.Report{
				Name:     "PlaylistTypeInspector",
				Severity: core.Error,
				Message:  "PLAYLIST-TYPE must be EVENT",
				Values:   values,
			}
		}
	case PlaylistTypeMustVOD:
		if noType || event || !vod {
			return &core.Report{
				Name:     "PlaylistTypeInspector",
				Severity: core.Error,
				Message:  "PLAYLIST-TYPE must be VOD",
				Values:   values,
			}
		}
	}
	switch ins.config.EndlistCondition {
	case EndlistMustExist:
		if !endlist || noEndlist {
			return &core.Report{
				Name:     "PlaylistTypeInspector",
				Severity: core.Error,
				Message:  "ENDLIST must exist",
				Values:   values,
			}
		}
	case EndlistMustNotExist:
		if endlist || !noEndlist {
			return &core.Report{
				Name:     "PlaylistTypeInspector",
				Severity: core.Error,
				Message:  "ENDLIST must not exist",
				Values:   values,
			}
		}
	}
	return &core.Report{
		Name:     "PlaylistTypeInspector",
		Severity: core.Info,
		Message:  "good",
		Values:   values,
	}
}
