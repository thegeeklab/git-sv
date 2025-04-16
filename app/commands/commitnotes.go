package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/thegeeklab/git-sv/app"
	"github.com/urfave/cli/v3"
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

func CommitNotesHandler(g app.GitSV, settings *app.CommitNotesSettings) cli.ActionFunc {
	return func(_ context.Context, _ *cli.Command) error {
		var date time.Time

		lr, err := logRange(g, settings.Range, settings.Start, settings.End)
		if err != nil {
			return err
		}

		commits, err := g.Log(lr)
		if err != nil {
			return fmt.Errorf("error getting git log from range: %s: %w", settings.Range, err)
		}

		if len(commits) > 0 {
			date, _ = time.Parse("2006-01-02", commits[0].Date)
		}

		output, err := g.OutputFormatter.FormatReleaseNote(g.ReleasenotesProcessor.Create(nil, "", date, commits))
		if err != nil {
			return fmt.Errorf("could not format commit notes: %w", err)
		}

		if settings.End == "" {
			fmt.Fprintf(os.Stdout, "%s\n", output)

			return nil
		}

		w, err := os.Create(settings.End)
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
