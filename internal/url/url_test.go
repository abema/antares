package url

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveReference(t *testing.T) {
	var u string
	var err error
	u, err = ResolveReference("https://foo.bar/baz/qux.quux", "corge.grault")
	require.NoError(t, err)
	assert.Equal(t, "https://foo.bar/baz/corge.grault", u)
	u, err = ResolveReference("", "https://foo.bar/baz.qux")
	require.NoError(t, err)
	assert.Equal(t, "https://foo.bar/baz.qux", u)
	_, err = ResolveReference(":invalid URL", "https://foo.bar/baz.qux")
	assert.Error(t, err)
	_, err = ResolveReference("https://foo.bar/baz.qux", ":invalid URL")
	assert.Error(t, err)
}

func TestExtNoError(t *testing.T) {
	assert.Equal(t, ".quux", ExtNoError("https://foo.bar/baz/qux.quux"))
	assert.Equal(t, ".quux", ExtNoError("https://foo.bar/baz/qux.quux?corge=grault"))
	assert.Equal(t, ".quux", ExtNoError("/baz/qux.quux"))
	assert.Equal(t, "", ExtNoError("https://foo.bar/baz/qux"))
	assert.Equal(t, "", ExtNoError("://foo.bar/baz/qux.quux"))
}
