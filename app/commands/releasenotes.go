package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/thegeeklab/git-sv/app"
	"github.com/thegeeklab/git-sv/sv"
	"github.com/urfave/cli/v2"
)

func ReleaseNotesFlags(settings *app.ReleaseNotesSettings) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "t",
			Aliases:     []string{"tag"},
			Usage:       "get release note from tag",
			Destination: &settings.Tag,
			Value:       "next",
		},
		&cli.StringFlag{
			Name:        "o",
			Aliases:     []string{"output"},
			Usage:       "output file name. Omit to use standard output.",
			Destination: &settings.Out,
		},
	}
}

func ReleaseNotesHandler(g app.GitSV, settings *app.ReleaseNotesSettings) cli.ActionFunc {
	return func(c *cli.Context) error {
		var (
			commits   []sv.CommitLog
			rnVersion *semver.Version
			tag       string
			date      time.Time
			err       error
		)

		tagFlag := strings.TrimSpace(strings.ToLower(settings.Tag))

		if tagFlag == "next" {
			// TODO: should generate release notes if version was not updated?
			rnVersion, _, date, commits, err = getNextVersionInfo(g, g.CommitProcessor)
		} else {
			rnVersion, date, commits, err = getTagVersionInfo(g, tag)
		}

		if err != nil {
			return err
		}

		releasenote := g.ReleasenotesProcessor.Create(rnVersion, tag, date, commits)

		output, err := g.OutputFormatter.FormatReleaseNote(releasenote)
		if err != nil {
			return fmt.Errorf("could not format release notes: %w", err)
		}

		if settings.Out == "" {
			os.Stdout.WriteString(fmt.Sprintf("%s\n", output))

			return nil
		}

		w, err := os.Create(settings.Out)
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
