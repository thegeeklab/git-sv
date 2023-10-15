package commands

import (
	"fmt"

	"github.com/thegeeklab/git-sv/v2/app"
	"github.com/thegeeklab/git-sv/v2/sv"
	"github.com/urfave/cli/v2"
)

func CommitFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "no-scope",
			Aliases: []string{"nsc"},
			Usage:   "do not prompt for commit scope",
		},
		&cli.BoolFlag{
			Name:    "no-body",
			Aliases: []string{"nbd"},
			Usage:   "do not prompt for commit body",
		},
		&cli.BoolFlag{
			Name:    "no-issue",
			Aliases: []string{"nis"},
			Usage:   "do not prompt for commit issue, will try to recover from branch if enabled",
		},
		&cli.BoolFlag{
			Name:    "no-breaking",
			Aliases: []string{"nbc"},
			Usage:   "do not prompt for breaking changes",
		},
		&cli.StringFlag{
			Name:    "type",
			Aliases: []string{"t"},
			Usage:   "define commit type",
		},
		&cli.StringFlag{
			Name:    "scope",
			Aliases: []string{"s"},
			Usage:   "define commit scope",
		},
		&cli.StringFlag{
			Name:    "description",
			Aliases: []string{"d"},
			Usage:   "define commit description",
		},
		&cli.StringFlag{
			Name:    "breaking-change",
			Aliases: []string{"b"},
			Usage:   "define commit breaking change message",
		},
	}
}

func CommitHandler(g app.GitSV) cli.ActionFunc {
	return func(c *cli.Context) error {
		noBreaking := c.Bool("no-breaking")
		noBody := c.Bool("no-body")
		noIssue := c.Bool("no-issue")
		noScope := c.Bool("no-scope")
		inputType := c.String("type")
		inputScope := c.String("scope")
		inputDescription := c.String("description")
		inputBreakingChange := c.String("breaking-change")

		ctype, err := getCommitType(g.Config, g.MessageProcessor, inputType)
		if err != nil {
			return err
		}

		scope, err := getCommitScope(g.Config, g.MessageProcessor, inputScope, noScope)
		if err != nil {
			return err
		}

		subject, err := getCommitDescription(g.MessageProcessor, inputDescription)
		if err != nil {
			return err
		}

		fullBody, err := getCommitBody(noBody)
		if err != nil {
			return err
		}

		issue, err := getCommitIssue(g.Config, g.MessageProcessor, g.Branch(), noIssue)
		if err != nil {
			return err
		}

		breakingChange, err := getCommitBreakingChange(noBreaking, inputBreakingChange)
		if err != nil {
			return err
		}

		header, body, footer := g.MessageProcessor.Format(
			sv.NewCommitMessage(ctype, scope, subject, fullBody, issue, breakingChange),
		)

		err = g.Commit(header, body, footer)
		if err != nil {
			return fmt.Errorf("error executing git commit: %w", err)
		}

		return nil
	}
}
