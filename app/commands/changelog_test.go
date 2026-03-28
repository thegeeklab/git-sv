package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thegeeklab/git-sv/app"
)

func TestChangelogFlags(t *testing.T) {
	flags := ChangelogFlags(&app.ChangelogSettings{})
	assert.NotEmpty(t, flags)
}

func TestChangelogHandler(t *testing.T) {
	gsv := app.New()
	settings := &app.ChangelogSettings{}
	handler := ChangelogHandler(gsv, settings)
	assert.NotNil(t, handler)
}
