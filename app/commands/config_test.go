package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thegeeklab/git-sv/app"
	"github.com/urfave/cli/v3"
)

func TestConfigDefaultHandler(t *testing.T) {
	handler := ConfigDefaultHandler()
	err := handler(context.Background(), &cli.Command{})
	assert.NoError(t, err)
}

func TestConfigShowHandler(t *testing.T) {
	cfg := app.GetDefault()
	handler := ConfigShowHandler(cfg)
	err := handler(context.Background(), &cli.Command{})
	assert.NoError(t, err)
}
