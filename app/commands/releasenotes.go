package commands

import (
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/thegeeklab/git-sv/v2/app"
	"github.com/thegeeklab/git-sv/v2/sv"
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

func ReleaseNotesHandler(g app.GitSV) cli.ActionFunc {
	return func(c *cli.Context) error {
		var (
			commits   []sv.CommitLog
			rnVersion *semver.Version
			tag       string
			date      time.Time
			err       error
		)

		if tag = c.String("t"); tag != "" {
			rnVersion, date, commits, err = getTagVersionInfo(g, tag)
		} else {
			// TODO: should generate release notes if version was not updated?
			rnVersion, _, date, commits, err = getNextVersionInfo(g, g.CommitProcessor)
		}

		if err != nil {
			return err
		}

		releasenote := g.ReleasenotesProcessor.Create(rnVersion, tag, date, commits)

		output, err := g.OutputFormatter.FormatReleaseNote(releasenote)
		if err != nil {
			return fmt.Errorf("could not format release notes, message: %w", err)
		}

		fmt.Println(output)

		return nil
	}
}
