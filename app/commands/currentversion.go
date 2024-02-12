package commands

import (
	"fmt"

	"github.com/thegeeklab/git-sv/app"
	"github.com/thegeeklab/git-sv/sv"
	"github.com/urfave/cli/v2"
)

func CurrentVersionHandler(gsv app.GitSV) cli.ActionFunc {
	return func(_ *cli.Context) error {
		lastTag := gsv.LastTag()

		currentVer, err := sv.ToVersion(lastTag)
		if err != nil {
			return fmt.Errorf("error parsing version: %s from git tag: %w", lastTag, err)
		}

		fmt.Printf("%d.%d.%d\n", currentVer.Major(), currentVer.Minor(), currentVer.Patch())

		return nil
	}
}
