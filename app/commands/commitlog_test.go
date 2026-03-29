package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thegeeklab/git-sv/app"
)

func TestCommitLogFlags(t *testing.T) {
	flags := CommitLogFlags(&app.CommitLogSettings{})
	assert.NotEmpty(t, flags)
}

func TestCommitLogHandler(t *testing.T) {
	gsv := app.New()
	settings := &app.CommitLogSettings{}
	handler := CommitLogHandler(gsv, settings)
	assert.NotNil(t, handler)
}
