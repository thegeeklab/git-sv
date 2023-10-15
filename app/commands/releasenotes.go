package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/thegeeklab/git-sv/v2/app"
	"github.com/thegeeklab/git-sv/v2/sv"
	"github.com/urfave/cli/v2"
)

func ReleaseNotesFlags(settings *app.ReleaseNotesSettings) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "t",
			Aliases:     []string{"tag"},
			Usage:       "get release note from tag",
			Destination: &settings.Tag,
		},
		&cli.StringFlag{
			Name:        "o",
			Aliases:     []string{"output"},
			Usage:       "output file name. Omit to use standard output.",
			Destination: &settings.Out,
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

		if tag = g.Settings.ReleaseNotesSettings.Tag; tag != "" {
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
			return fmt.Errorf("could not format release notes: %w", err)
		}

		if g.Settings.ReleaseNotesSettings.Out == "" {
			os.Stdout.WriteString(fmt.Sprintf("%s\n", output))

			return nil
		}

		w, err := os.Create(g.Settings.ReleaseNotesSettings.Out)
		if err != nil {
			return fmt.Errorf("could not write release notes: %w", err)
		}
		defer w.Close()

		if _, err := w.Write(output); err != nil {
			return err
		}

		return nil
	}
}
