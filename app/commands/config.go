package commands

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/thegeeklab/git-sv/app"
	"github.com/urfave/cli/v3"
)

func ConfigDefaultHandler() cli.ActionFunc {
	return func(_ context.Context, _ *cli.Command) error {
		cfg := app.GetDefault()

		content, err := yaml.Marshal(&cfg)
		if err != nil {
			return err
		}

		fmt.Println(string(content))

		return nil
	}
}

func ConfigShowHandler(cfg *app.Config) cli.ActionFunc {
	return func(_ context.Context, _ *cli.Command) error {
		content, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}

		fmt.Println(string(content))

		return nil
	}
}
