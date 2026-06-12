package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thegeeklab/git-sv/app"
)

func TestRetagFlags(t *testing.T) {
	flags := RetagFlags(&app.RetagSettings{})
	assert.NotEmpty(t, flags)
}

func TestRetagHandler(t *testing.T) {
	gsv := app.New()
	settings := &app.RetagSettings{}
	handler := RetagHandler(gsv, settings)
	assert.NotNil(t, handler)
}
