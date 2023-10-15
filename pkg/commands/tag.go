package commands

import (
	"fmt"

	"github.com/thegeeklab/git-sv/v2/pkg/app"
	"github.com/urfave/cli/v2"
)

func TagHandler(gsv app.GitSV, semverProcessor app.CommitsProcessor) cli.ActionFunc {
	return func(c *cli.Context) error {
		lastTag := gsv.LastTag()

		currentVer, err := app.ToVersion(lastTag)
		if err != nil {
			return fmt.Errorf("error parsing version: %s from git tag, message: %w", lastTag, err)
		}

		commits, err := gsv.Log(app.NewLogRange(app.TagRange, lastTag, ""))
		if err != nil {
			return fmt.Errorf("error getting git log, message: %w", err)
		}

		nextVer, _ := semverProcessor.NextVersion(currentVer, commits)
		tagname, err := gsv.Tag(*nextVer)

		fmt.Println(tagname)

		if err != nil {
			return fmt.Errorf("error generating tag version: %s, message: %w", nextVer.String(), err)
		}

		return nil
	}
}
