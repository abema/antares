package adapters

import (
	"encoding/json"
	"log"
	"net/url"
	"path"
	"strings"

	"github.com/abema/antares/core"
	"github.com/abema/antares/internal/file"
)

func OnDownloadPathSuffixFilter(handler core.OnDownloadHandler, suffixes ...string) core.OnDownloadHandler {
	return func(httpFile *core.File) {
		u, err := url.Parse(httpFile.URL)
		if err != nil {
			log.Printf("ERROR: invalid URL: %s: %s", u, err)
			return
		}
		for _, suffix := range suffixes {
			if strings.HasSuffix(u.Path, suffix) {
				handler(httpFile)
				return
			}
		}
	}
}

func LocalFileExporter(baseDir string, enableMeta bool) core.OnDownloadHandler {
	e := &localFileExporter{
		BaseDir:    baseDir,
		EnableMeta: enableMeta,
	}
	return func(httpFile *core.File) {
		if err := e.onDownload(httpFile); err != nil {
			log.Printf("ERROR: failed to export to file: %s: %s", httpFile.URL, err)
		}
	}
}

type localFileExporter struct {
	BaseDir    string
	EnableMeta bool
}

func (e *localFileExporter) onDownload(httpFile *core.File) error {
	p, err := e.resolvePath(httpFile)
	if err != nil {
		return err
	}
	if err := e.export(p, httpFile); err != nil {
		return err
	}
	if e.EnableMeta {
		if err := e.exportMeta(p+"-meta.json", httpFile); err != nil {
			return err
		}
	}
	return nil
}

func (e *localFileExporter) resolvePath(httpFile *core.File) (string, error) {
	static := e.isStatic(httpFile)
	u, err := url.Parse(httpFile.URL)
	if err != nil {
		return "", err
	}
	p := path.Join(e.BaseDir, u.Host, u.Path)
	if !static {
		ext := path.Ext(p)
		p = p[:len(p)-len(ext)] + httpFile.RequestTimestamp.Format("-20060102-150405.000") + ext
	}
	return p, nil
}

func (e *localFileExporter) isStatic(httpFile *core.File) bool {
	switch httpFile.ResponseHeader.Get("Content-Type") {
	case "application/x-mpegURL", "application/dash+xml":
		return false
	case "video/mp4", "video/MP2T":
		return true
	}
	u, err := url.Parse(httpFile.URL)
	if err != nil {
		return false
	}
	return strings.HasSuffix(u.Path, ".mp4") ||
		strings.HasSuffix(u.Path, ".m4s") ||
		strings.HasSuffix(u.Path, ".m4v") ||
		strings.HasSuffix(u.Path, ".m4a") ||
		strings.HasSuffix(u.Path, ".ts")
}

func (e *localFileExporter) export(path string, httpFile *core.File) error {
	f, err := file.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(httpFile.Body); err != nil {
		return err
	}
	return nil
}

func (e *localFileExporter) exportMeta(path string, httpFile *core.File) error {
	f, err := file.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(httpFile.Meta); err != nil {
		return err
	}
	return nil
}
