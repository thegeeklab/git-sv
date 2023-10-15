package commands

import (
	"fmt"

	"github.com/thegeeklab/git-sv/v2/pkg/config"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

func ConfigDefaultHandler() cli.ActionFunc {
	return func(c *cli.Context) error {
		cfg := config.GetDefault()

		content, err := yaml.Marshal(&cfg)
		if err != nil {
			return err
		}

		fmt.Println(string(content))

		return nil
	}
}

func ConfigShowHandler(cfg *config.Config) cli.ActionFunc {
	return func(c *cli.Context) error {
		content, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}

		fmt.Println(string(content))

		return nil
	}
}
