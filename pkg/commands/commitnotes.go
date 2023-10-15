package commands

import (
	"fmt"
	"time"

	"github.com/thegeeklab/git-sv/v2/pkg/app"
	"github.com/thegeeklab/git-sv/v2/pkg/formatter"
	"github.com/urfave/cli/v2"
)

func CommitNotesFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name: "r", Aliases: []string{"range"},
			Usage:    "type of range of commits, use: tag, date or hash",
			Required: true,
		},
		&cli.StringFlag{
			Name:    "s",
			Aliases: []string{"start"},
			Usage:   "start range of git log revision range, if date, the value is used on since flag instead",
		},
		&cli.StringFlag{
			Name:    "e",
			Aliases: []string{"end"},
			Usage:   "end range of git log revision range, if date, the value is used on until flag instead",
		},
	}
}

func CommitNotesHandler(
	git app.GitSV, rnProcessor app.ReleaseNoteProcessor, outputFormatter formatter.OutputFormatter,
) cli.ActionFunc {
	return func(c *cli.Context) error {
		var date time.Time

		rangeFlag := c.String("r")

		lr, err := logRange(git, rangeFlag, c.String("s"), c.String("e"))
		if err != nil {
			return err
		}

		commits, err := git.Log(lr)
		if err != nil {
			return fmt.Errorf("error getting git log from range: %s, message: %w", rangeFlag, err)
		}

		if len(commits) > 0 {
			date, _ = time.Parse("2006-01-02", commits[0].Date)
		}

		output, err := outputFormatter.FormatReleaseNote(rnProcessor.Create(nil, "", date, commits))
		if err != nil {
			return fmt.Errorf("could not format release notes, message: %w", err)
		}

		fmt.Println(output)

		return nil
	}
}
