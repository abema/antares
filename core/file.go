package core

import (
	"net/http"
	"time"
)

type Meta struct {
	URL              string        `json:"url"`
	Via              string        `json:"via"`
	RequestHeader    http.Header   `json:"requestHeader"`
	ResponseHeader   http.Header   `json:"responseHeader"`
	Status           string        `json:"status"`     // e.g. "200 OK"
	StatusCode       int           `json:"statusCode"` // e.g. 200
	Proto            string        `json:"proto"`      // e.g. "HTTP/1.0"
	ProtoMajor       int           `json:"protoMajor"` // e.g. 1
	ProtoMinor       int           `json:"protoMinor"` // e.g. 0
	ContentLength    int64         `json:"contentLength"`
	TransferEncoding []string      `json:"transferEncoding"`
	Uncompressed     bool          `json:"uncompressed"`
	RequestTimestamp time.Time     `json:"requestTimestamp"`
	DownloadTime     time.Duration `json:"downloadTime"`
}

type File struct {
	Meta
	Body []byte
}
