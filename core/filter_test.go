package core

import (
	"testing"

	"github.com/grafov/m3u8"
	"github.com/stretchr/testify/assert"
	"github.com/zencoder/go-dash/helpers/ptrs"
	"github.com/zencoder/go-dash/mpd"
)

func TestSegmentFilter(t *testing.T) {
	filter := SegmentFilterOr(
		SegmentFilterAnd(
			MaxBandwidthSegmentFilter(9000*1e3),
			MinBandwidthSegmentFilter(7000*1e3),
		),
		SegmentFilterAnd(
			MaxBandwidthSegmentFilter(5000*1e3),
			MinBandwidthSegmentFilter(3000*1e3),
		),
	)
	assert.Equal(t, Reject, filter.CheckHLS(&HLSSegment{
		VariantParams: &m3u8.VariantParams{Bandwidth: 10000 * 1e3},
	}))
	assert.Equal(t, Pass, filter.CheckHLS(&HLSSegment{
		VariantParams: &m3u8.VariantParams{Bandwidth: 8000 * 1e3},
	}))
	assert.Equal(t, Reject, filter.CheckHLS(&HLSSegment{
		VariantParams: &m3u8.VariantParams{Bandwidth: 6000 * 1e3},
	}))
	assert.Equal(t, Pass, filter.CheckHLS(&HLSSegment{
		VariantParams: &m3u8.VariantParams{Bandwidth: 4000 * 1e3},
	}))
	assert.Equal(t, Reject, filter.CheckHLS(&HLSSegment{
		VariantParams: &m3u8.VariantParams{Bandwidth: 2000 * 1e3},
	}))
	assert.Equal(t, Reject, filter.CheckDASH(&DASHSegment{
		Representation: &mpd.Representation{Bandwidth: ptrs.Int64ptr(10000 * 1e3)},
	}))
	assert.Equal(t, Pass, filter.CheckDASH(&DASHSegment{
		Representation: &mpd.Representation{Bandwidth: ptrs.Int64ptr(8000 * 1e3)},
	}))
	assert.Equal(t, Reject, filter.CheckDASH(&DASHSegment{
		Representation: &mpd.Representation{Bandwidth: ptrs.Int64ptr(6000 * 1e3)},
	}))
	assert.Equal(t, Pass, filter.CheckDASH(&DASHSegment{
		Representation: &mpd.Representation{Bandwidth: ptrs.Int64ptr(4000 * 1e3)},
	}))
	assert.Equal(t, Reject, filter.CheckDASH(&DASHSegment{
		Representation: &mpd.Representation{Bandwidth: ptrs.Int64ptr(2000 * 1e3)},
	}))
}
