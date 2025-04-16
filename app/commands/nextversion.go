package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/thegeeklab/git-sv/app"
	"github.com/thegeeklab/git-sv/sv"
	"github.com/urfave/cli/v3"
)

func NextVersionHandler(g app.GitSV) cli.ActionFunc {
	return func(_ context.Context, _ *cli.Command) error {
		lastTag := g.LastTag()

		currentVer, err := sv.ToVersion(lastTag)
		if err != nil {
			return fmt.Errorf("error parsing version: %s from git tag: %w", lastTag, err)
		}

		commits, err := g.Log(app.NewLogRange(app.TagRange, lastTag, ""))
		if err != nil {
			return fmt.Errorf("error getting git log: %w", err)
		}

		nextVer, updated := g.CommitProcessor.NextVersion(currentVer, commits)
		if !updated {
			log.Info().Msgf("nothing to do: current version %s unchanged", currentVer)

			return nil
		}

		fmt.Printf("%d.%d.%d\n", nextVer.Major(), nextVer.Minor(), nextVer.Patch())

		return nil
	}
}
