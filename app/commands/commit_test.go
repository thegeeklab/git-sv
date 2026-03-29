package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thegeeklab/git-sv/app"
)

func TestCommitFlags(t *testing.T) {
	flags := CommitFlags()
	assert.NotEmpty(t, flags)
}

func TestCommitHandler(t *testing.T) {
	gsv := app.New()
	handler := CommitHandler(gsv)
	assert.NotNil(t, handler)
}
