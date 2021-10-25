package strings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsIn(t *testing.T) {
	set := []string{"foo", "bar", "baz"}
	assert.True(t, ContainsIn("foo", set))
	assert.True(t, ContainsIn("bar", set))
	assert.True(t, ContainsIn("baz", set))
	assert.False(t, ContainsIn("qux", set))
}
