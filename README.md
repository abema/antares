# antares

[![Go Reference](https://pkg.go.dev/badge/github.com/abema/antares.svg)](https://pkg.go.dev/github.com/abema/antares)
[![CircleCI](https://circleci.com/gh/abema/antares.svg?style=svg)](https://circleci.com/gh/abema/antares)
[![Coverage Status](https://coveralls.io/repos/github/abema/antares/badge.svg?branch=main)](https://coveralls.io/github/abema/antares?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/abema/antares)](https://goreportcard.com/report/github.com/abema/antares)

Antares is monitoring system for HLS and MPEG-DASH.
This program is written by golang.

## Description

Antares monitors any HLS/MPEG-DASH streams and outputs inspection reports.
You can use prepared command or Go interfaces.

## Command

### Install

```sh
go install github.com/abema/antares
```

### Example

```sh
antares \
	-hls.noEndlist \
	-hls.playlistType omitted \
	-export \
	-export.meta \
	-segment.maxBandwidth 500000 \
	"http://localhost/index.m3u8"
```

### Help

```sh
antares -h
```

## Integrate with your Go application

### Monitor and Manager

You can use `core.Monitor` to monitor your live stream as follows:

```go
config := core.NewConfig("http://localhost/index.m3u8", core.StreamTypeHLS)
config.HLS.Inspectors = []core.HLSInspector{
	hls.NewSpeedInspector(),
	hls.NewVariantsSyncInspector(),
}
core.NewMonitor(config)
```

`manager.Manager` manages multiple monitors and provides batch update interface.

```go
manager := manager.NewManager(&manager.Config{})
for range time.Tick(time.Minute) {
	configs := make(map[string]*core.Config)
	for _, stream := range listMyCurrentStreams() {
		config := core.NewConfig(stream.URL, stream.StreamType)
		  :
		configs[stream.ID] = config
	}
	added, removed := manager.Batch(configs)
	log.Println("added", added)
	log.Println("removed:", removed)
}
```

### Inspectors

Inspector inspects manifest and segment files.
For example, `SpeedInspector` checks whether addition speed of segment is appropriate as compared to real time.
Some inspectors are implemented in `inspectors/hls` package and `inspectors/dash` package for each aims.
Implementing `hls.Inspector` or `dash.Inspector` interface, you can add your any inspectors to Monitor.

### Handlers and Adapters

You can set handlers to handle downloaded files, inspection reports, and etc.
And `adapters` package has some useful handlers.

```go
config.OnReport = core.MergeOnReportHandlers(
	adapters.ReportLogger(&adapters.ReportLogConfig{JSON: true}, os.Stdout),
	adapters.Alarm(&adapters.AlarmConfig{
		OnAlarm                     : func(reports core.Reports) { /* start alarm */ },
		OnRecover                   : func(reports core.Reports) { /* stop alarm */ },
		Window                      : 10,
		AlarmIfErrorGreaterThanEqual: 2,
		RecoverIfErrorLessThanEqual : 0,
	}),
	func(reports core.Reports) { /* send metrics */ },
)
```

## Manifest format support

### HLS

- [x] Live
- [x] Event
- [x] On-demand
- [ ] Byte range
- [ ] LHLS
- [ ] Decryption
- [ ] I-frame-only playlists

### DASH

- [x] Live
- [x] Static
- [x] SegmentTimeline
- [ ] Open-Ended SegmentTimeline (S@r = -1)
- [ ] SegmentList
- [ ] Only SegmentTemplate (Without SegmentTimeline/SegmentList)
- [x] Multi-Period
- [x] Location
- [ ] Decryption

Identifiers for URL templates:

- [x] $$
- [x] $RepresentationID$
- [x] $Number$
- [x] $Bandwidth$
- [x] $Time$
- [ ] $SubNumber$
- [ ] IEEE 1003.1 Format Tag

## License

[MIT](https://github.com/abema/antares/blob/main/LICENSE)
