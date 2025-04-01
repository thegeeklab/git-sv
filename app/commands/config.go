package commands

import (
	"fmt"

	"github.com/thegeeklab/git-sv/app"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

func ConfigDefaultHandler() cli.ActionFunc {
	return func(_ *cli.Context) error {
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
	return func(_ *cli.Context) error {
		content, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}

		fmt.Println(string(content))

		return nil
	}
}
