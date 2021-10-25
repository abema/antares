package file

import (
	"os"
	"path"
)

func Open(name string) (*os.File, error) {
	return OpenFile(name, os.O_RDONLY, 0)
}

func Create(name string) (*os.File, error) {
	return OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

func Append(name string) (*os.File, error) {
	return OpenFile(name, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
}

func OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	logDir := path.Dir(name)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}
	return os.OpenFile(name, flag, perm)
}
