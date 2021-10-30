package adapters

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/abema/antares/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnReportFrequencyFilter(t *testing.T) {
	var a int
	var r int
	handler := Alarm(&AlarmConfig{
		OnAlarm: func(reports core.Reports) {
			a++
		},
		OnRecover: func(reports core.Reports) {
			r++
		},
		Window:                        5,
		AlarmIfErrorGreaterThanEqual:  3,
		RecoverIfInfoGreaterThanEqual: 4,
	})
	handler(core.Reports{{Name: "test", Severity: core.Error}})
	handler(core.Reports{{Name: "test", Severity: core.Info}})
	handler(core.Reports{{Name: "test", Severity: core.Info}})
	handler(core.Reports{{Name: "test", Severity: core.Error}})
	handler(core.Reports{{Name: "test", Severity: core.Info}})
	handler(core.Reports{{Name: "test", Severity: core.Error}})
	require.Equal(t, 0, a)
	handler(core.Reports{{Name: "test", Severity: core.Error}})
	require.Equal(t, 1, a)
	handler(core.Reports{{Name: "test", Severity: core.Warn}})
	handler(core.Reports{{Name: "test", Severity: core.Info}})
	handler(core.Reports{{Name: "test", Severity: core.Info}})
	handler(core.Reports{{Name: "test", Severity: core.Info}})
	require.Equal(t, 0, r)
	require.Equal(t, 1, a)
	handler(core.Reports{{Name: "test", Severity: core.Info}})
	require.Equal(t, 1, r)
	require.Equal(t, 1, a)
	handler(core.Reports{{Name: "test", Severity: core.Info}})
	require.Equal(t, 1, r)
	require.Equal(t, 1, a)
}

func TestReportLogger(t *testing.T) {
	w := bytes.NewBuffer(nil)
	ReportLogger(&ReportLogConfig{
		JSON: true,
	}, w)(core.Reports{
		{
			Name: "r1", Severity: core.Info, Message: "Report 1", Values: core.Values{
				"int": 1, "string": "foo",
			},
		}, {
			Name: "r2", Severity: core.Warn, Message: "Report 2", Values: core.Values{
				"int": 2, "string": "bar",
			},
		}, {
			Name: "r3", Severity: core.Error, Message: "Report 3", Values: core.Values{
				"int": 3, "string": "baz",
			},
		},
	})
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Bytes(), &out))
	assert.Len(t, out["reports"], 3)
	r1 := out["reports"].([]interface{})[0].(map[string]interface{})
	assert.Equal(t, map[string]interface{}{
		"name":     "r1",
		"severity": "INFO",
		"message":  "Report 1",
		"values":   map[string]interface{}{"int": float64(1), "string": "foo"},
	}, r1)
	assert.Equal(t, "ERROR", out["severity"])
}

func TestFileReportLogger(t *testing.T) {
	dir, err := ioutil.TempDir("", "antares-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	name := path.Join(dir, "test.log")
	FileReportLogger(&ReportLogConfig{
		Summary: true,
	}, name)(core.Reports{
		{
			Name: "r1", Severity: core.Info, Message: "Report 1", Values: core.Values{
				"int": 1, "string": "foo",
			},
		}, {
			Name: "r2", Severity: core.Warn, Message: "Report 2", Values: core.Values{
				"int": 2, "string": "bar",
			},
		},
	})
	f, err := os.Open(name)
	require.NoError(t, err)
	b, err := ioutil.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, "WARNING: Summary info=1 warn=1 error=0\n"+
		"WARNING: r2: Report 2: int=2 string=bar\n"+
		"INFO: r1: Report 1: int=1 string=foo\n", string(b))
}
