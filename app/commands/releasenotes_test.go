package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thegeeklab/git-sv/app"
)

func TestReleaseNotesFlags(t *testing.T) {
	flags := ReleaseNotesFlags(&app.ReleaseNotesSettings{})
	assert.NotEmpty(t, flags)
}

func TestReleaseNotesHandler(t *testing.T) {
	gsv := app.New()
	settings := &app.ReleaseNotesSettings{}
	handler := ReleaseNotesHandler(gsv, settings)
	assert.NotNil(t, handler)
}
