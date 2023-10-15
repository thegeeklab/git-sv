package commands

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/thegeeklab/git-sv/v2/pkg/app"
	"github.com/urfave/cli/v2"
)

var (
	errCanNotCreateTagFlag = errors.New("cannot define tag flag with range, start or end flags")
	errInvalidRange        = errors.New("invalid log range")
	errUnknownTag          = errors.New("unknown tag")
)

func CommitLogFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "t",
			Aliases: []string{"tag"},
			Usage:   "get commit log from a specific tag",
		},
		&cli.StringFlag{
			Name:    "r",
			Aliases: []string{"range"},
			Usage:   "type of range of commits, use: tag, date or hash",
			Value:   string(app.TagRange),
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

func CommitLogHandler(gsv app.GitSV) cli.ActionFunc {
	return func(c *cli.Context) error {
		var (
			commits []app.CommitLog
			err     error
		)

		tagFlag := c.String("t")
		rangeFlag := c.String("r")
		startFlag := c.String("s")
		endFlag := c.String("e")

		if tagFlag != "" && (rangeFlag != string(app.TagRange) || startFlag != "" || endFlag != "") {
			return errCanNotCreateTagFlag
		}

		if tagFlag != "" {
			commits, err = getTagCommits(gsv, tagFlag)
		} else {
			r, rerr := logRange(gsv, rangeFlag, startFlag, endFlag)
			if rerr != nil {
				return rerr
			}
			commits, err = gsv.Log(r)
		}

		if err != nil {
			return fmt.Errorf("error getting git log, message: %w", err)
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
