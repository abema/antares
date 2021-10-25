package core

import (
	"hash/crc32"
	"math"
)

type FilterResult int

const (
	Pass FilterResult = iota
	Reject
)

type SegmentFilter interface {
	CheckHLS(segment *HLSSegment) FilterResult
	CheckDASH(segment *DASHSegment) FilterResult
}

type segmentFilterAnd struct {
	filters []SegmentFilter
}

func SegmentFilterAnd(filters ...SegmentFilter) SegmentFilter {
	return &segmentFilterAnd{
		filters: filters,
	}
}

func (f *segmentFilterAnd) CheckHLS(segment *HLSSegment) FilterResult {
	for _, filter := range f.filters {
		if filter.CheckHLS(segment) == Reject {
			return Reject
		}
	}
	return Pass
}

func (f *segmentFilterAnd) CheckDASH(segment *DASHSegment) FilterResult {
	for _, filter := range f.filters {
		if filter.CheckDASH(segment) == Reject {
			return Reject
		}
	}
	return Pass
}

type segmentFilterOr struct {
	filters []SegmentFilter
}

func SegmentFilterOr(filters ...SegmentFilter) SegmentFilter {
	return &segmentFilterOr{
		filters: filters,
	}
}

func (f *segmentFilterOr) CheckHLS(segment *HLSSegment) FilterResult {
	for _, filter := range f.filters {
		if filter.CheckHLS(segment) == Pass {
			return Pass
		}
	}
	return Reject
}

func (f *segmentFilterOr) CheckDASH(segment *DASHSegment) FilterResult {
	for _, filter := range f.filters {
		if filter.CheckDASH(segment) == Pass {
			return Pass
		}
	}
	return Reject
}

type allSegmentRejectionFilter struct {
}

func AllSegmentRejectionFilter() SegmentFilter {
	return &allSegmentRejectionFilter{}
}

func (f *allSegmentRejectionFilter) CheckHLS(segment *HLSSegment) FilterResult {
	return Reject
}

func (f *allSegmentRejectionFilter) CheckDASH(segment *DASHSegment) FilterResult {
	return Reject
}

type maxBandwidthSegmentFilter struct {
	bandwidth int64
}

func MaxBandwidthSegmentFilter(bandwidth int64) SegmentFilter {
	return &maxBandwidthSegmentFilter{
		bandwidth: bandwidth,
	}
}

func (f *maxBandwidthSegmentFilter) CheckHLS(segment *HLSSegment) FilterResult {
	if segment.VariantParams != nil &&
		int64(segment.VariantParams.Bandwidth) <= f.bandwidth {
		return Pass
	} else {
		return Reject
	}
}

func (f *maxBandwidthSegmentFilter) CheckDASH(segment *DASHSegment) FilterResult {
	if segment.Representation != nil &&
		segment.Representation.Bandwidth != nil &&
		*segment.Representation.Bandwidth <= f.bandwidth {
		return Pass
	} else {
		return Reject
	}
}

type minBandwidthSegmentFilter struct {
	bandwidth int64
}

func MinBandwidthSegmentFilter(bandwidth int64) SegmentFilter {
	return &minBandwidthSegmentFilter{
		bandwidth: bandwidth,
	}
}

func (f *minBandwidthSegmentFilter) CheckHLS(segment *HLSSegment) FilterResult {
	if segment.VariantParams != nil &&
		int64(segment.VariantParams.Bandwidth) >= f.bandwidth {
		return Pass
	} else {
		return Reject
	}
}

func (f *minBandwidthSegmentFilter) CheckDASH(segment *DASHSegment) FilterResult {
	if segment.Representation != nil &&
		segment.Representation.Bandwidth != nil &&
		*segment.Representation.Bandwidth >= f.bandwidth {
		return Pass
	} else {
		return Reject
	}
}

type hashSamplingSegmentFilter struct {
	rate float64
}

func HashSamplingSegmentFilter(rate float64) SegmentFilter {
	return &hashSamplingSegmentFilter{
		rate: rate,
	}
}

func (f *hashSamplingSegmentFilter) CheckHLS(segment *HLSSegment) FilterResult {
	return f.check(segment.URL)
}

func (f *hashSamplingSegmentFilter) CheckDASH(segment *DASHSegment) FilterResult {
	return f.check(segment.URL)
}

func (f *hashSamplingSegmentFilter) check(url string) FilterResult {
	hash32 := crc32.ChecksumIEEE([]byte(url))
	val := float64(hash32) / float64(math.MaxUint32)
	if val < f.rate {
		return Pass
	} else {
		return Reject
	}
}
