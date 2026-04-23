package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thegeeklab/git-sv/app"
	"github.com/urfave/cli/v3"
)

func TestTagFlags(t *testing.T) {
	flags := TagFlags(&app.TagSettings{})
	assert.NotEmpty(t, flags)

	forceFlag := false
	for _, flag := range flags {
		boolFlag, ok := flag.(*cli.BoolFlag)
		if !ok {
			continue
		}

		if boolFlag.Name == "force" {
			forceFlag = true
			assert.Contains(t, boolFlag.Aliases, "f")
		}
	}

	assert.True(t, forceFlag)
}

func TestTagHandler(t *testing.T) {
	gsv := app.New()
	settings := &app.TagSettings{}
	handler := TagHandler(gsv, settings)
	assert.NotNil(t, handler)
}
