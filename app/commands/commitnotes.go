package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/thegeeklab/git-sv/v2/app"
	"github.com/urfave/cli/v2"
)

func CommitNotesFlags(settings *app.CommitNotesSettings) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name: "r", Aliases: []string{"range"},
			Usage:       "type of range of commits, use: tag, date or hash",
			Required:    true,
			Destination: &settings.Range,
		},
		&cli.StringFlag{
			Name:        "s",
			Aliases:     []string{"start"},
			Usage:       "start range of git log revision range, if date, the value is used on since flag instead",
			Destination: &settings.Start,
		},
		&cli.StringFlag{
			Name:        "e",
			Aliases:     []string{"end"},
			Usage:       "end range of git log revision range, if date, the value is used on until flag instead",
			Destination: &settings.End,
		},
		&cli.StringFlag{
			Name:        "o",
			Aliases:     []string{"output"},
			Usage:       "output file name. Omit to use standard output.",
			Destination: &settings.Out,
		},
	}
}

func CommitNotesHandler(g app.GitSV) cli.ActionFunc {
	return func(c *cli.Context) error {
		var date time.Time

		rangeFlag := g.Settings.CommitNotesSettings.Range

		lr, err := logRange(g, rangeFlag, g.Settings.CommitNotesSettings.Start, g.Settings.CommitNotesSettings.End)
		if err != nil {
			return err
		}

		commits, err := g.Log(lr)
		if err != nil {
			return fmt.Errorf("error getting git log from range: %s: %w", rangeFlag, err)
		}

		if len(commits) > 0 {
			date, _ = time.Parse("2006-01-02", commits[0].Date)
		}

		output, err := g.OutputFormatter.FormatReleaseNote(g.ReleasenotesProcessor.Create(nil, "", date, commits))
		if err != nil {
			return fmt.Errorf("could not format commit notes: %w", err)
		}

		if g.Settings.CommitNotesSettings.Out == "" {
			os.Stdout.WriteString(fmt.Sprintf("%s\n", output))

			return nil
		}

		w, err := os.Create(g.Settings.CommitNotesSettings.Out)
		if err != nil {
			return fmt.Errorf("could not write commit notes: %w", err)
		}
		defer w.Close()

		if _, err := w.Write(output); err != nil {
			return err
		}

		return nil
	}
}
