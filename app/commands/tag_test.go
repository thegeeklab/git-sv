package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thegeeklab/git-sv/app"
)

func TestTagHandler(t *testing.T) {
	gsv := app.New()
	settings := &app.TagSettings{}
	handler := TagHandler(gsv, settings)
	assert.NotNil(t, handler)
}
