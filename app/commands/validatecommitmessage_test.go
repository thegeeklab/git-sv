package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thegeeklab/git-sv/app"
)

func TestValidateCommitMessageFlags(t *testing.T) {
	flags := ValidateCommitMessageFlags()
	assert.NotEmpty(t, flags)
}

func TestValidateCommitMessageHandler(t *testing.T) {
	gsv := app.New()
	handler := ValidateCommitMessageHandler(gsv)
	assert.NotNil(t, handler)
}
