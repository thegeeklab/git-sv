package commands

import (
	"fmt"

	"github.com/thegeeklab/git-sv/v2/pkg/git"
	"github.com/urfave/cli/v2"
)

func NextVersionHandler(gsv git.SV, semverProcessor git.SemVerCommitsProcessor) cli.ActionFunc {
	return func(c *cli.Context) error {
		lastTag := gsv.LastTag()

		currentVer, err := git.ToVersion(lastTag)
		if err != nil {
			return fmt.Errorf("error parsing version: %s from git tag, message: %w", lastTag, err)
		}

		commits, err := gsv.Log(git.NewLogRange(git.TagRange, lastTag, ""))
		if err != nil {
			return fmt.Errorf("error getting git log, message: %w", err)
		}

		nextVer, _ := semverProcessor.NextVersion(currentVer, commits)

		fmt.Printf("%d.%d.%d\n", nextVer.Major(), nextVer.Minor(), nextVer.Patch())

		return nil
	}
}
