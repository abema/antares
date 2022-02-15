package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/abema/antares/adapters"
	"github.com/abema/antares/core"
	"github.com/abema/antares/inspectors/dash"
	"github.com/abema/antares/inspectors/hls"
	"github.com/abema/antares/internal/url"
)

var opts struct {
	IntervalMs uint
	IsHLS      bool
	IsDASH     bool
	HLS        struct {
		PlaylistType string
		Endlist      bool
		NoEndlist    bool
	}
	DASH struct {
		MandatoryMimeTypes string
		ValidMimeTypes     string
		MPDType            string
		MaxHeight          int64
		MinHeight          int64
		ValidPARs          string
		MaxVideoBandwidth  int64
		MinVideoBandwidth  int64
		MaxAudioBandwidth  int64
		MinAudioBandwidth  int64
	}
	Export struct {
		Enable bool
		Meta   bool
		Dir    string
	}
	Segment struct {
		Disable      bool
		MaxBandwidth uint
		MinBandwidth uint
	}
	Log struct {
		JSON     bool
		Severity string
	}
	HTTP struct {
		Header string
	}
}
var flagSet *flag.FlagSet

func main() {
	defaultExportDir := "" + time.Now().Format("export-20060102-150405")
	flagSet = flag.NewFlagSet("antares", flag.ExitOnError)
	flagSet.UintVar(&opts.IntervalMs, "interval", 0, "fixed manifest polling interval (milliseconds).")
	flagSet.BoolVar(&opts.IsHLS, "hls", false, "This flag indicates URL argument is HLS.")
	flagSet.BoolVar(&opts.IsDASH, "dash", false, "This flag indicates URL argument is DASH.")
	flagSet.BoolVar(&opts.Export.Enable, "export", false, "Export raw data as local files.")
	flagSet.StringVar(&opts.HLS.PlaylistType, "hls.playlistType", "", "PLAYLIST-TYPE tag status (omitted|event|vod)")
	flagSet.BoolVar(&opts.HLS.Endlist, "hls.endlist", false, "If true, playlist must have ENDLIST tag.")
	flagSet.BoolVar(&opts.HLS.NoEndlist, "hls.noEndlist", false, "If true, playlist must have no ENDLIST tag.")
	flagSet.StringVar(&opts.DASH.MandatoryMimeTypes, "dash.mandatoryMimeTypes", "", "comma-separated list of mandatory mimeType attribute values.")
	flagSet.StringVar(&opts.DASH.ValidMimeTypes, "dash.validMimeTypes", "", "comma-separated list of valid mimeType attribute values.")
	flagSet.StringVar(&opts.DASH.MPDType, "dash.mpdType", "", "expected MPD@type attribute value. (static|dynamic)")
	flagSet.Int64Var(&opts.DASH.MaxHeight, "dash.maxHeight", 0, "maximum value of height.")
	flagSet.Int64Var(&opts.DASH.MinHeight, "dash.minHeight", 0, "minimum value of height.")
	flagSet.StringVar(&opts.DASH.ValidPARs, "dash.validPARs", "", "picture aspect ratio which calculated by width, height and sar. (ex: \"16:9,4:3\")")
	flagSet.Int64Var(&opts.DASH.MaxVideoBandwidth, "dash.maxVideoBandwidth", 0, "maximum value of video bandwidth.")
	flagSet.Int64Var(&opts.DASH.MinVideoBandwidth, "dash.minVideoBandwidth", 0, "minimum value of video bandwidth.")
	flagSet.Int64Var(&opts.DASH.MaxAudioBandwidth, "dash.maxAudioBandwidth", 0, "maximum value of audio bandwidth.")
	flagSet.Int64Var(&opts.DASH.MinAudioBandwidth, "dash.minAudioBandwidth", 0, "minimum value of audio bandwidth.")
	flagSet.BoolVar(&opts.Export.Meta, "export.meta", false, "Export metadatas with raw files.")
	flagSet.StringVar(&opts.Export.Dir, "export.dir", defaultExportDir, "an export directory")
	flagSet.BoolVar(&opts.Segment.Disable, "segment.disable", false, "Disable segment download.")
	flagSet.UintVar(&opts.Segment.MaxBandwidth, "segment.maxBandwidth", 0, "max-bandwidth segment filter")
	flagSet.UintVar(&opts.Segment.MinBandwidth, "segment.minBandwidth", 0, "min-bandwidth segment filter")
	flagSet.BoolVar(&opts.Log.JSON, "log.json", false, "JSON log format")
	flagSet.StringVar(&opts.Log.Severity, "log.severity", "info", "log severity (info|warn|error)")
	flagSet.StringVar(&opts.HTTP.Header, "http.head", "", "file name of custom request header.")
	flagSet.Parse(os.Args[1:])

	if len(flagSet.Args()) != 1 {
		invalidArguments("URL must be specified")
	}

	terminated := make(chan struct{})

	u := flagSet.Args()[0]
	streamType := getStreamType(u)
	config := core.NewConfig(u, streamType)
	if opts.IntervalMs != 0 {
		config.DefaultInterval = time.Millisecond * time.Duration(opts.IntervalMs)
	} else {
		config.PrioritizeSuggestedInterval = true
	}
	config.SegmentFilter = segmentFilter()
	config.TerminateIfVOD = true
	if opts.Export.Enable {
		config.OnDownload = adapters.LocalFileExporter(opts.Export.Dir, opts.Export.Meta)
	}
	config.OnReport = buildOnReportHandler()
	config.OnTerminate = func() {
		terminated <- struct{}{}
	}
	switch streamType {
	case core.StreamTypeHLS:
		config.HLS = hlsConfig()
	case core.StreamTypeDASH:
		config.DASH = dashConfig()
	}
	config.RequestHeader = buildRequestHeader()
	m := core.NewMonitor(config)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-terminated:
	case sig := <-sigCh:
		log.Print("SIGNAL:", sig)
		m.Terminate()
		<-terminated
	}
	log.Print("terminated")
}

func getStreamType(u string) core.StreamType {
	if opts.IsHLS {
		return core.StreamTypeHLS
	} else if opts.IsDASH {
		return core.StreamTypeDASH
	} else {
		switch url.ExtNoError(u) {
		case ".m3u8":
			return core.StreamTypeHLS
		case ".mpd":
			return core.StreamTypeDASH
		default:
			invalidArguments("if the extension is neither .m3u8 or .mpd, you must use -hls or -dash option")
		}
	}
	return 0
}

func segmentFilter() core.SegmentFilter {
	filters := make([]core.SegmentFilter, 0)
	if opts.Segment.Disable {
		filters = append(filters, core.AllSegmentRejectionFilter())
	} else {
		if opts.Segment.MaxBandwidth != 0 {
			filters = append(filters, core.MaxBandwidthSegmentFilter(int64(opts.Segment.MaxBandwidth)))
		}
		if opts.Segment.MinBandwidth != 0 {
			filters = append(filters, core.MaxBandwidthSegmentFilter(int64(opts.Segment.MinBandwidth)))
		}
	}
	return core.SegmentFilterAnd(filters...)
}

func hlsConfig() *core.HLSConfig {
	inspectors := []core.HLSInspector{
		hls.NewSpeedInspector(),
		hls.NewVariantsSyncInspector(),
	}
	if opts.HLS.PlaylistType != "" || opts.HLS.NoEndlist {
		config := new(hls.PlaylistTypeInspectorConfig)
		switch opts.HLS.PlaylistType {
		case "omitted":
			config.PlaylistTypeCondition = hls.PlaylistTypeMustOmitted
		case "event":
			config.PlaylistTypeCondition = hls.PlaylistTypeMustEvent
		case "vod":
			config.PlaylistTypeCondition = hls.PlaylistTypeMustVOD
		default:
			invalidArguments("unknown playlist type: %s", opts.HLS.PlaylistType)
		}
		if opts.HLS.Endlist && opts.HLS.NoEndlist {
			invalidArguments("only one of -hls.endlist or -hls.noEndlist can be true")
		} else if opts.HLS.Endlist {
			config.EndlistCondition = hls.EndlistMustExist
		} else if opts.HLS.NoEndlist {
			config.EndlistCondition = hls.EndlistMustNotExist
		}
		inspectors = append(inspectors, hls.NewPlaylistTypeInspector(config))
	}
	return &core.HLSConfig{
		Inspectors: inspectors,
	}
}

func dashConfig() *core.DASHConfig {
	inspectors := []core.DASHInspector{
		dash.NewSpeedInspector(),
	}
	if opts.DASH.MandatoryMimeTypes != "" || opts.DASH.ValidMimeTypes != "" {
		inspectors = append(inspectors, dash.NewAdaptationSetInspector(&dash.AdaptationSetInspectorConfig{
			MandatoryMimeTypes: strings.Split(opts.DASH.MandatoryMimeTypes, ","),
			ValidMimeTypes:     strings.Split(opts.DASH.ValidMimeTypes, ","),
		}))
	}
	if opts.DASH.MPDType != "" {
		inspectors = append(inspectors, dash.NewMPDTypeInspector(opts.DASH.MPDType))
	}
	if opts.DASH.MaxHeight != 0 || opts.DASH.MinHeight != 0 || opts.DASH.ValidPARs != "" ||
		opts.DASH.MaxVideoBandwidth != 0 || opts.DASH.MinVideoBandwidth != 0 ||
		opts.DASH.MaxAudioBandwidth != 0 || opts.DASH.MinAudioBandwidth != 0 {
		inspectors = append(inspectors, dash.NewRepresentationInspector(&dash.RepresentationInspectorConfig{
			ErrorMaxHeight:         opts.DASH.MaxHeight,
			ErrorMinHeight:         opts.DASH.MinHeight,
			ValidPARs:              buildAspectRatios(opts.DASH.ValidPARs),
			ErrorMaxVideoBandwidth: opts.DASH.MaxVideoBandwidth,
			ErrorMinVideoBandwidth: opts.DASH.MinVideoBandwidth,
			ErrorMaxAudioBandwidth: opts.DASH.MaxAudioBandwidth,
			ErrorMinAudioBandwidth: opts.DASH.MinAudioBandwidth,
		}))
	}
	return &core.DASHConfig{
		Inspectors: inspectors,
	}
}

func buildAspectRatios(ars string) []dash.AspectRatio {
	aspectRatios := make([]dash.AspectRatio, 0)
	for _, ar := range strings.Split(ars, ",") {
		aspectRatio, err := dash.ParseAspectRatio(ar)
		if err != nil {
			invalidArguments("invalid aspect ratio format: %s", ars)
		}
		aspectRatios = append(aspectRatios, aspectRatio)
	}
	return aspectRatios
}

func buildOnReportHandler() core.OnReportHandler {
	var severity core.Severity
	switch opts.Log.Severity {
	case "info":
		severity = core.Info
	case "warn":
		severity = core.Warn
	case "error":
		severity = core.Error
	default:
		invalidArguments("invalid log severity: %s", opts.Log.Severity)
	}
	return adapters.ReportLogger(&adapters.ReportLogConfig{
		Flag:     log.LstdFlags,
		Summary:  true,
		JSON:     opts.Log.JSON,
		Severity: severity,
	}, os.Stdout)
}

func buildRequestHeader() http.Header {
	if opts.HTTP.Header == "" {
		return http.Header{}
	}
	file, err := os.Open(opts.HTTP.Header)
	if err != nil {
		invalidArguments("file not found: %s", opts.HTTP.Header)
	}
	defer file.Close()
	r := bufio.NewReader(file)
	header := make(http.Header)
	for {
		line, _, err := r.ReadLine()
		if err == io.EOF {
			return header
		}
		if err != nil {
			panic(err)
		}
		s := strings.SplitN(string(line), ":", 2)
		if len(s) == 2 {
			header.Add(s[0], strings.TrimLeft(s[1], " "))
		}
	}
}

func printUsage() {
	println("USAGE: antares monitor [OPTIONS] URL")
	println()
	println("OPTIONS:")
	flagSet.PrintDefaults()
}

func invalidArguments(format string, args ...interface{}) {
	println("ERROR: invalid arguments:", fmt.Sprintf(format, args...))
	println()
	println("HELP: antares -h")
	os.Exit(1)
}
