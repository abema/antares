package core

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
)

type Severity int

const (
	Info Severity = iota
	Warn
	Error
)

func (s Severity) String() string {
	switch s {
	case Info:
		return "INFO"
	case Warn:
		return "WARNING"
	case Error:
		return "ERROR"
	}
	return ""
}

func (s *Severity) UnmarshalText(text []byte) error {
	switch string(text) {
	case Info.String():
		*s = Info
	case Warn.String():
		*s = Warn
	case Error.String():
		*s = Error
	default:
		return errors.New("unknown severity")
	}
	return nil
}

func (s Severity) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s Severity) WorseThan(o Severity) bool {
	return s > o
}

func (s Severity) BetterThan(o Severity) bool {
	return s < o
}

func (s Severity) WorseThanOrEqual(o Severity) bool {
	return s >= o
}

func (s Severity) BetterThanOrEqual(o Severity) bool {
	return s <= o
}

func WorstSeverity(ss ...Severity) Severity {
	worst := Info
	for _, s := range ss {
		if s.WorseThan(worst) {
			worst = s
		}
	}
	return worst
}

func BestSeverity(ss ...Severity) Severity {
	best := Error
	for _, s := range ss {
		if s.BetterThan(best) {
			best = s
		}
	}
	return best
}

type Values map[string]interface{}

func (values Values) Keys() []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (values Values) String() string {
	buf := bytes.NewBuffer(nil)
	for _, key := range values.Keys() {
		if buf.Len() != 0 {
			buf.WriteString(" ")
		}
		fmt.Fprintf(buf, "%s=[%v]", key, values[key])
	}
	return string(buf.Bytes())
}

type Report struct {
	Name     string   `json:"name"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Values   Values   `json:"values"`
}

type Reports []*Report

func (reports Reports) WorstSeverity() Severity {
	var worst Severity
	for _, report := range reports {
		worst = WorstSeverity(worst, report.Severity)
	}
	return worst
}

func (reports Reports) Infos() Reports {
	infos := make([]*Report, 0, len(reports))
	for _, report := range reports {
		if report.Severity == Info {
			infos = append(infos, report)
		}
	}
	return infos
}

func (reports Reports) Warns() Reports {
	warns := make([]*Report, 0, len(reports))
	for _, report := range reports {
		if report.Severity == Warn {
			warns = append(warns, report)
		}
	}
	return warns
}

func (reports Reports) Errors() Reports {
	errors := make([]*Report, 0, len(reports))
	for _, report := range reports {
		if report.Severity == Error {
			errors = append(errors, report)
		}
	}
	return errors
}
