package commands

import (
	"fmt"
	"os"
	"sort"

	"github.com/thegeeklab/git-sv/app"
	"github.com/thegeeklab/git-sv/sv"
	"github.com/urfave/cli/v2"
)

func ChangelogFlags(settings *app.ChangelogSettings) []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:        "size",
			Value:       10, //nolint:gomnd
			Aliases:     []string{"n"},
			Destination: &settings.Size,
			Usage:       "get changelog from last 'n' tags",
		},
		&cli.BoolFlag{
			Name:        "all",
			Usage:       "ignore size parameter, get changelog for every tag",
			Destination: &settings.All,
		},
		&cli.BoolFlag{
			Name:        "add-next",
			Usage:       "add next version on change log (commits since last tag, only if there is a new release)",
			Destination: &settings.AddNext,
		},
		&cli.BoolFlag{
			Name:        "strict",
			Usage:       "only show tags 'SemVer-ish'",
			Destination: &settings.Strict,
		},
		&cli.StringFlag{
			Name:        "o",
			Aliases:     []string{"output"},
			Usage:       "output file name. Omit to use standard output.",
			Destination: &settings.Out,
		},
	}
}

//nolint:gocognit
func ChangelogHandler(g app.GitSV, settings *app.ChangelogSettings) cli.ActionFunc {
	return func(c *cli.Context) error {
		tags, err := g.Tags()
		if err != nil {
			return err
		}

		sort.Slice(tags, func(i, j int) bool {
			return tags[i].Date.After(tags[j].Date)
		})

		var releaseNotes []sv.ReleaseNote

		if settings.AddNext {
			rnVersion, updated, date, commits, uerr := getNextVersionInfo(g, g.CommitProcessor)
			if uerr != nil {
				return uerr
			}

			if updated {
				releaseNotes = append(releaseNotes, g.ReleasenotesProcessor.Create(rnVersion, "", date, commits))
			}
		}

		for i, tag := range tags {
			if !settings.All && i >= settings.Size {
				break
			}

			previousTag := ""
			if i+1 < len(tags) {
				previousTag = tags[i+1].Name
			}

			if settings.Strict && !sv.IsValidVersion(tag.Name) {
				continue
			}

			commits, err := g.Log(app.NewLogRange(app.TagRange, previousTag, tag.Name))
			if err != nil {
				return fmt.Errorf("error getting git log from tag: %s: %w", tag.Name, err)
			}

			currentVer, _ := sv.ToVersion(tag.Name)
			releaseNotes = append(releaseNotes, g.ReleasenotesProcessor.Create(currentVer, tag.Name, tag.Date, commits))
		}

		output, err := g.OutputFormatter.FormatChangelog(releaseNotes)
		if err != nil {
			return fmt.Errorf("could not format changelog: %w", err)
		}

		if settings.Out == "" {
			os.Stdout.WriteString(fmt.Sprintf("%s\n", output))

			return nil
		}

		w, err := os.Create(settings.Out)
		if err != nil {
			return fmt.Errorf("could not write changelog: %w", err)
		}
		defer w.Close()

		if _, err := w.Write(output); err != nil {
			return err
		}

		return nil
	}
}
