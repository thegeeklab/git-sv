package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thegeeklab/git-sv/app"
)

func TestCommitNotesFlags(t *testing.T) {
	flags := CommitNotesFlags(&app.CommitNotesSettings{})
	assert.NotEmpty(t, flags)
}

func TestCommitNotesHandler(t *testing.T) {
	gsv := app.New()
	settings := &app.CommitNotesSettings{}
	handler := CommitNotesHandler(gsv, settings)
	assert.NotNil(t, handler)
}
