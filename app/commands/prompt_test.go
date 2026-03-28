package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPromptSelect_InvalidInput(t *testing.T) {
	_, err := promptSelect("label", nil, nil)
	assert.Error(t, err)

	_, err = promptSelect("label", "not-a-slice", nil)
	assert.Error(t, err)
}
