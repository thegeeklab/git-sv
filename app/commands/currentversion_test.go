package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thegeeklab/git-sv/app"
)

func TestCurrentVersionHandler(t *testing.T) {
	gsv := app.New()
	handler := CurrentVersionHandler(gsv)
	assert.NotNil(t, handler)
}
