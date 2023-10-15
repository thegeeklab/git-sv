package commands

import (
	"fmt"

	"github.com/thegeeklab/git-sv/v2/pkg/app"
	"github.com/urfave/cli/v2"
)

func CurrentVersionHandler(gsv app.GitSV) cli.ActionFunc {
	return func(c *cli.Context) error {
		lastTag := gsv.LastTag()

		currentVer, err := app.ToVersion(lastTag)
		if err != nil {
			return fmt.Errorf("error parsing version: %s from git tag, message: %w", lastTag, err)
		}

		fmt.Printf("%d.%d.%d\n", currentVer.Major(), currentVer.Minor(), currentVer.Patch())

		return nil
	}
}
