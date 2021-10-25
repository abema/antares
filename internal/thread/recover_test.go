package thread

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoPanic(t *testing.T) {
	t.Run("panic", func(t *testing.T) {
		err := NoPanic(func() error {
			panic("foo")
		})()
		require.Error(t, err)
		assert.True(t, strings.HasPrefix(err.Error(), "panic: foo\ngoroutine "))
	})

	t.Run("panic error", func(t *testing.T) {
		parent := errors.New("foo")
		err := NoPanic(func() error {
			panic(parent)
		})()
		require.Error(t, err)
		assert.True(t, strings.HasPrefix(err.Error(), "panic: foo\ngoroutine "))
		assert.True(t, errors.Is(err, parent))
	})

	t.Run("error", func(t *testing.T) {
		err := NoPanic(func() error {
			return errors.New("foo")
		})()
		require.Error(t, err)
		assert.Equal(t, "foo", err.Error())
	})

	t.Run("no error", func(t *testing.T) {
		err := NoPanic(func() error {
			return nil
		})()
		require.NoError(t, err)
	})
}
