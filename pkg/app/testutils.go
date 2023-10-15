package app

import (
	"time"

	"github.com/Masterminds/semver/v3"
)

func TestVersion(v string) *semver.Version {
	r, _ := semver.NewVersion(v)

	return r
}

func TestCommitlog(ctype string, metadata map[string]string, author string) CommitLog {
	breaking := false
	if _, found := metadata[BreakingChangeMetadataKey]; found {
		breaking = true
	}

	return CommitLog{
		Message: CommitMessage{
			Type:             ctype,
			Description:      "subject text",
			IsBreakingChange: breaking,
			Metadata:         metadata,
		},
		AuthorName: author,
	}
}

func TestReleaseNote(
	version *semver.Version,
	tag string,
	date time.Time,
	sections []ReleaseNoteSection,
	authorsNames map[string]struct{},
) ReleaseNote {
	return ReleaseNote{
		Version:      version,
		Tag:          tag,
		Date:         date.Truncate(time.Minute),
		Sections:     sections,
		AuthorsNames: authorsNames,
	}
}

func TestNewReleaseNoteCommitsSection(name string, types []string, items []CommitLog) ReleaseNoteCommitsSection {
	return ReleaseNoteCommitsSection{
		Name:  name,
		Types: types,
		Items: items,
	}
}
