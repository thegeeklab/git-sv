package commands

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/thegeeklab/git-sv/v2/pkg/git"
	"github.com/urfave/cli/v2"
)

const laxFilePerm = 0o644

var (
	errReadCommitMessage = errors.New("failed to read commit message")
	errAppendFooter      = errors.New("failed to append meta-informations on footer")
)

func ValidateCommitMessageFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "path",
			Required: true,
			Usage:    "git working directory",
		},
		&cli.StringFlag{
			Name:     "file",
			Required: true,
			Usage:    "name of the file that contains the commit log message",
		},
		&cli.StringFlag{
			Name:     "source",
			Required: true,
			Usage:    "source of the commit message",
		},
	}
}

func ValidateCommitMessageHandler(gsv git.SV, messageProcessor git.MessageProcessor) cli.ActionFunc {
	return func(c *cli.Context) error {
		branch := gsv.Branch()
		detached, derr := gsv.IsDetached()

		if messageProcessor.SkipBranch(branch, derr == nil && detached) {
			log.Warn().Msg("commit message validation skipped, branch in ignore list or detached...")

			return nil
		}

		if source := c.String("source"); source == "merge" {
			log.Warn().Msgf("commit message validation skipped, ignoring source: %s...", source)

			return nil
		}

		filepath := filepath.Join(c.String("path"), c.String("file"))

		commitMessage, err := readFile(filepath)
		if err != nil {
			return fmt.Errorf("%w: %s", errReadCommitMessage, err.Error())
		}

		if err := messageProcessor.Validate(commitMessage); err != nil {
			return fmt.Errorf("%w: %s", errReadCommitMessage, err.Error())
		}

		msg, err := messageProcessor.Enhance(branch, commitMessage)
		if err != nil {
			log.Warn().Err(err).Msg("could not enhance commit message")

			return nil
		}

		if msg == "" {
			return nil
		}

		if err := appendOnFile(msg, filepath, laxFilePerm); err != nil {
			return fmt.Errorf("%w: %s", errAppendFooter, err.Error())
		}

		return nil
	}
}
