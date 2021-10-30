package adapters

import (
	"encoding/json"
	"io"
	"log"
	"time"

	"github.com/abema/antares/core"
	"github.com/abema/antares/internal/file"
)

type AlarmConfig struct {
	OnAlarm                       core.OnReportHandler
	OnRecover                     core.OnReportHandler
	Window                        int
	AlarmIfErrorGreaterThanEqual  int
	RecoverIfInfoGreaterThanEqual int
}

func Alarm(config *AlarmConfig) core.OnReportHandler {
	history := make([]core.Severity, 0)
	var alarm bool
	return func(reports core.Reports) {
		history = append(history, reports.WorstSeverity())
		if len(history) > config.Window {
			history = history[1:]
		}
		var infoCnt int
		var errCnt int
		for _, hist := range history {
			switch hist {
			case core.Info:
				infoCnt++
			case core.Error:
				errCnt++
			}
		}
		if errCnt >= config.AlarmIfErrorGreaterThanEqual {
			if !alarm {
				config.OnAlarm(reports)
				alarm = true
			}
		} else if infoCnt >= config.RecoverIfInfoGreaterThanEqual {
			if alarm {
				config.OnRecover(reports)
				alarm = false
			}
		}
	}
}

type ReportLogConfig struct {
	// Flag is log flag defined standard log package.
	// When JSON option is true, this option is ignored.
	Flag int
	// Summary represents whether to output summary line.
	// When JSON option is true, this option is ignored.
	Summary  bool
	JSON     bool
	Severity core.Severity
}

func ReportLogger(config *ReportLogConfig, w io.Writer) core.OnReportHandler {
	return func(reports core.Reports) {
		writeReport(config, w, reports)
	}
}

func FileReportLogger(config *ReportLogConfig, name string) core.OnReportHandler {
	return func(reports core.Reports) {
		file, err := file.Append(name)
		if err != nil {
			log.Printf("failed to open log file: %s: %s", name, err)
			return
		}
		defer file.Close()
		writeReport(config, file, reports)
	}
}

func writeReport(config *ReportLogConfig, w io.Writer, reports core.Reports) {
	if config.JSON {
		writeReportJSON(config, w, reports)
	} else {
		writeReportDefault(config, w, reports)
	}
}

func writeReportDefault(config *ReportLogConfig, w io.Writer, reports core.Reports) {
	logger := log.New(w, "", config.Flag)
	if config.Summary {
		severity := reports.WorstSeverity()
		if config.Severity <= severity {
			logger.Printf("%s: Summary info=%d warn=%d error=%d", severity, len(reports.Infos()), len(reports.Warns()), len(reports.Errors()))
		}
	}
	if config.Severity.BetterThanOrEqual(core.Error) {
		for _, err := range reports.Errors() {
			logger.Printf("ERROR: %s: %s: %s", err.Name, err.Message, err.Values)
		}
	}
	if config.Severity.BetterThanOrEqual(core.Warn) {
		for _, warn := range reports.Warns() {
			logger.Printf("WARNING: %s: %s: %s", warn.Name, warn.Message, warn.Values)
		}
	}
	if config.Severity.BetterThanOrEqual(core.Info) {
		for _, info := range reports.Infos() {
			logger.Printf("INFO: %s: %s: %s", info.Name, info.Message, info.Values)
		}
	}
}

func writeReportJSON(config *ReportLogConfig, w io.Writer, reports core.Reports) {
	severity := reports.WorstSeverity()
	if config.Severity <= severity {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"reports":  reports,
			"severity": severity.String(),
			"time":     time.Now().Format(time.RFC3339),
		})
	}
}
