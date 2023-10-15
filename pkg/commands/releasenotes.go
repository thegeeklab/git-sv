package commands

import (
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/thegeeklab/git-sv/v2/pkg/formatter"
	"github.com/thegeeklab/git-sv/v2/pkg/git"
	"github.com/urfave/cli/v2"
)

func ReleaseNotesFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "t",
			Aliases: []string{"tag"},
			Usage:   "get release note from tag",
		},
	}
}

func ReleaseNotesHandler(
	gsv git.SV,
	semverProcessor git.CommitsProcessor,
	rnProcessor git.ReleaseNoteProcessor,
	outputFormatter formatter.OutputFormatter,
) cli.ActionFunc {
	return func(c *cli.Context) error {
		var (
			commits   []git.CommitLog
			rnVersion *semver.Version
			tag       string
			date      time.Time
			err       error
		)

		if tag = c.String("t"); tag != "" {
			rnVersion, date, commits, err = getTagVersionInfo(gsv, tag)
		} else {
			// TODO: should generate release notes if version was not updated?
			rnVersion, _, date, commits, err = getNextVersionInfo(gsv, semverProcessor)
		}

		if err != nil {
			return err
		}

		releasenote := rnProcessor.Create(rnVersion, tag, date, commits)

		output, err := outputFormatter.FormatReleaseNote(releasenote)
		if err != nil {
			return fmt.Errorf("could not format release notes, message: %w", err)
		}

		fmt.Println(output)

		return nil
	}
}
