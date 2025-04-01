package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/thegeeklab/git-sv/app"
	"github.com/thegeeklab/git-sv/sv"
	"github.com/urfave/cli/v3"
)

var (
	errCanNotCreateTagFlag = errors.New("cannot define tag flag with range, start or end flags")
	errInvalidRange        = errors.New("invalid log range")
	errUnknownTag          = errors.New("unknown tag")
)

func CommitLogFlags(settings *app.CommitLogSettings) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "t",
			Aliases:     []string{"tag"},
			Usage:       "get commit log from a specific tag",
			Destination: &settings.Tag,
			Value:       "next",
		},
		&cli.StringFlag{
			Name:        "r",
			Aliases:     []string{"range"},
			Usage:       "type of range of commits, use: tag, date or hash",
			Destination: &settings.Range,
			Value:       string(app.TagRange),
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
	}
}

func CommitLogHandler(g app.GitSV, settings *app.CommitLogSettings) cli.ActionFunc {
	return func(_ *cli.Context) error {
		var (
			commits []sv.CommitLog
			err     error
		)

		tagDefault := "next"
		tagFlag := strings.TrimSpace(strings.ToLower(settings.Tag))

		if tagFlag != tagDefault &&
			(settings.Range != string(app.TagRange) || settings.Start != "" || settings.End != "") {
			return errCanNotCreateTagFlag
		}

		if tagFlag == tagDefault {
			r, rerr := logRange(g, settings.Range, settings.Start, settings.End)
			if rerr != nil {
				return rerr
			}

			commits, err = g.Log(r)
		} else {
			commits, err = getTagCommits(g, tagFlag)
		}

		if err != nil {
			return fmt.Errorf("error getting git log: %w", err)
		}

		for _, commit := range commits {
			content, err := json.Marshal(commit)
			if err != nil {
				return err
			}

			fmt.Println(string(content))
		}

		return nil
	}
}
