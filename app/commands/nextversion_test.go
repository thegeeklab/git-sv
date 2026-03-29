package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thegeeklab/git-sv/app"
)

func TestNextVersionHandler(t *testing.T) {
	gsv := app.New()
	handler := NextVersionHandler(gsv)
	assert.NotNil(t, handler)
}
