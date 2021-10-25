package url

import (
	"net/url"
	"path"
)

func ResolveReference(base string, rel string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u, err := b.Parse(rel)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func ExtNoError(u string) string {
	ext, _ := Ext(u)
	return ext
}

func Ext(u string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	return path.Ext(parsed.Path), nil
}
