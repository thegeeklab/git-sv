package commands

import (
	"fmt"

	"github.com/thegeeklab/git-sv/v2/pkg/app"
	"github.com/thegeeklab/git-sv/v2/pkg/sv"
	"github.com/urfave/cli/v2"
)

func TagHandler(g app.GitSV) cli.ActionFunc {
	return func(c *cli.Context) error {
		lastTag := g.LastTag()

		currentVer, err := sv.ToVersion(lastTag)
		if err != nil {
			return fmt.Errorf("error parsing version: %s from git tag, message: %w", lastTag, err)
		}

		commits, err := g.Log(app.NewLogRange(app.TagRange, lastTag, ""))
		if err != nil {
			return fmt.Errorf("error getting git log, message: %w", err)
		}

		nextVer, _ := g.CommitProcessor.NextVersion(currentVer, commits)
		tagname, err := g.Tag(*nextVer)

		fmt.Println(tagname)

		if err != nil {
			return fmt.Errorf("error generating tag version: %s, message: %w", nextVer.String(), err)
		}

		return nil
	}
}
