package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thegeeklab/git-sv/app"
)

func TestFind(t *testing.T) {
	tags := []app.Tag{
		{Name: "v1.0.0"},
		{Name: "v1.1.0"},
		{Name: "v2.0.0"},
	}

	assert.Equal(t, 1, find("v1.1.0", tags))
	assert.Equal(t, -1, find("v3.0.0", tags))
}

func TestStr(t *testing.T) {
	assert.Equal(t, "a", str("a", "b"))
	assert.Equal(t, "b", str("", "b"))
}

func TestLogRange(t *testing.T) {
	gsv := app.GitSV{}

	lr, err := logRange(gsv, "tag", "v1.0.0", "v2.0.0")
	assert.NoError(t, err)
	assert.NotNil(t, lr)

	lr, err = logRange(gsv, "date", "2020-01-01", "2020-12-31")
	assert.NoError(t, err)
	assert.NotNil(t, lr)

	lr, err = logRange(gsv, "hash", "abc", "def")
	assert.NoError(t, err)
	assert.NotNil(t, lr)

	_, err = logRange(gsv, "invalid", "", "")
	assert.Error(t, err)
}
