package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeverity(t *testing.T) {
	assert.Equal(t, "INFO", Info.String())
	assert.Equal(t, "WARNING", Warn.String())
	assert.Equal(t, "ERROR", Error.String())

	assert.False(t, Info.WorseThan(Info))
	assert.False(t, Info.WorseThan(Warn))
	assert.False(t, Info.WorseThan(Error))
	assert.True(t, Warn.WorseThan(Info))
	assert.False(t, Warn.WorseThan(Warn))
	assert.False(t, Warn.WorseThan(Error))
	assert.True(t, Error.WorseThan(Info))
	assert.True(t, Error.WorseThan(Warn))
	assert.False(t, Error.WorseThan(Error))

	assert.False(t, Info.BetterThan(Info))
	assert.True(t, Info.BetterThan(Warn))
	assert.True(t, Info.BetterThan(Error))
	assert.False(t, Warn.BetterThan(Info))
	assert.False(t, Warn.BetterThan(Warn))
	assert.True(t, Warn.BetterThan(Error))
	assert.False(t, Error.BetterThan(Info))
	assert.False(t, Error.BetterThan(Warn))
	assert.False(t, Error.BetterThan(Error))

	assert.True(t, Info.WorseThanOrEqual(Info))
	assert.False(t, Info.WorseThanOrEqual(Warn))
	assert.False(t, Info.WorseThanOrEqual(Error))
	assert.True(t, Warn.WorseThanOrEqual(Info))
	assert.True(t, Warn.WorseThanOrEqual(Warn))
	assert.False(t, Warn.WorseThanOrEqual(Error))
	assert.True(t, Error.WorseThanOrEqual(Info))
	assert.True(t, Error.WorseThanOrEqual(Warn))
	assert.True(t, Error.WorseThanOrEqual(Error))

	assert.True(t, Info.BetterThanOrEqual(Info))
	assert.True(t, Info.BetterThanOrEqual(Warn))
	assert.True(t, Info.BetterThanOrEqual(Error))
	assert.False(t, Warn.BetterThanOrEqual(Info))
	assert.True(t, Warn.BetterThanOrEqual(Warn))
	assert.True(t, Warn.BetterThanOrEqual(Error))
	assert.False(t, Error.BetterThanOrEqual(Info))
	assert.False(t, Error.BetterThanOrEqual(Warn))
	assert.True(t, Error.BetterThanOrEqual(Error))

	assert.Equal(t, Info, WorstSeverity(Info, Info, Info))
	assert.Equal(t, Warn, WorstSeverity(Info, Warn, Info))
	assert.Equal(t, Error, WorstSeverity(Error, Warn, Info))

	assert.Equal(t, Info, BestSeverity(Error, Warn, Info))
	assert.Equal(t, Warn, BestSeverity(Error, Warn, Warn))
	assert.Equal(t, Error, BestSeverity(Error, Error, Error))
}

func TestValues(t *testing.T) {
	values := Values{
		"int":    123,
		"string": "abc",
		"array":  []string{"foo", "bar"},
	}
	assert.Equal(t, []string{"array", "int", "string"}, values.Keys())
	assert.Equal(t, "array=[[foo bar]] int=[123] string=[abc]", values.String())
}

func TestReport(t *testing.T) {
	reports := Reports{
		{Severity: Error},
		{Severity: Info},
		{Severity: Warn},
		{Severity: Error},
		{Severity: Info},
		{Severity: Error},
	}
	assert.Equal(t, Error, reports.WorstSeverity())
	assert.Len(t, reports.Infos(), 2)
	assert.Len(t, reports.Warns(), 1)
	assert.Len(t, reports.Errors(), 3)
}
