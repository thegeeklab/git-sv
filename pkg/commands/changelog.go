package commands

import (
	"fmt"
	"sort"

	"github.com/thegeeklab/git-sv/v2/pkg/app"
	"github.com/thegeeklab/git-sv/v2/pkg/formatter"
	"github.com/urfave/cli/v2"
)

func ChangelogFlags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:    "size",
			Value:   10, //nolint:gomnd
			Aliases: []string{"n"},
			Usage:   "get changelog from last 'n' tags",
		},
		&cli.BoolFlag{
			Name:  "all",
			Usage: "ignore size parameter, get changelog for every tag",
		},
		&cli.BoolFlag{
			Name:  "add-next-version",
			Usage: "add next version on change log (commits since last tag, but only if there is a new version to release)",
		},
		&cli.BoolFlag{
			Name:  "semantic-version-only",
			Usage: "only show tags 'SemVer-ish'",
		},
	}
}

func ChangelogHandler(
	gsv app.GitSV,
	semverProcessor app.CommitsProcessor,
	rnProcessor app.ReleaseNoteProcessor,
	formatter formatter.OutputFormatter,
) cli.ActionFunc {
	return func(c *cli.Context) error {
		tags, err := gsv.Tags()
		if err != nil {
			return err
		}

		sort.Slice(tags, func(i, j int) bool {
			return tags[i].Date.After(tags[j].Date)
		})

		var releaseNotes []app.ReleaseNote

		size := c.Int("size")
		all := c.Bool("all")
		addNextVersion := c.Bool("add-next-version")
		semanticVersionOnly := c.Bool("semantic-version-only")

		if addNextVersion {
			rnVersion, updated, date, commits, uerr := getNextVersionInfo(gsv, semverProcessor)
			if uerr != nil {
				return uerr
			}

			if updated {
				releaseNotes = append(releaseNotes, rnProcessor.Create(rnVersion, "", date, commits))
			}
		}

		for i, tag := range tags {
			if !all && i >= size {
				break
			}

			previousTag := ""
			if i+1 < len(tags) {
				previousTag = tags[i+1].Name
			}

			if semanticVersionOnly && !app.IsValidVersion(tag.Name) {
				continue
			}

			commits, err := gsv.Log(app.NewLogRange(app.TagRange, previousTag, tag.Name))
			if err != nil {
				return fmt.Errorf("error getting git log from tag: %s, message: %w", tag.Name, err)
			}

			currentVer, _ := app.ToVersion(tag.Name)
			releaseNotes = append(releaseNotes, rnProcessor.Create(currentVer, tag.Name, tag.Date, commits))
		}

		output, err := formatter.FormatChangelog(releaseNotes)
		if err != nil {
			return fmt.Errorf("could not format changelog, message: %w", err)
		}

		fmt.Println(output)

		return nil
	}
}
