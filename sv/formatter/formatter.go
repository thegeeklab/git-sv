package formatter

import (
	"bytes"
	"sort"
	"text/template"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/thegeeklab/git-sv/v2/sv"
)

type releaseNoteTemplateVariables struct {
	Release     string
	Tag         string
	Version     *semver.Version
	Date        time.Time
	Sections    []sv.ReleaseNoteSection
	AuthorNames []string
}

// OutputFormatter output formatter interface.
type OutputFormatter interface {
	FormatReleaseNote(releasenote sv.ReleaseNote) ([]byte, error)
	FormatChangelog(releasenotes []sv.ReleaseNote) ([]byte, error)
}

// BaseOutputFormatter formater for release note and changelog.
type BaseOutputFormatter struct {
	templates *template.Template
}

// NewOutputFormatter TemplateProcessor constructor.
func NewOutputFormatter(tpls *template.Template) *BaseOutputFormatter {
	return &BaseOutputFormatter{templates: tpls}
}

// FormatReleaseNote format a release note.
func (p BaseOutputFormatter) FormatReleaseNote(releasenote sv.ReleaseNote) ([]byte, error) {
	var b bytes.Buffer
	if err := p.templates.ExecuteTemplate(&b, "releasenotes-md.tpl", releaseNoteVariables(releasenote)); err != nil {
		return b.Bytes(), err
	}

	return b.Bytes(), nil
}

// FormatChangelog format a changelog.
func (p BaseOutputFormatter) FormatChangelog(releasenotes []sv.ReleaseNote) ([]byte, error) {
	templateVars := make([]releaseNoteTemplateVariables, len(releasenotes))
	for i, v := range releasenotes {
		templateVars[i] = releaseNoteVariables(v)
	}

	var b bytes.Buffer
	if err := p.templates.ExecuteTemplate(&b, "changelog-md.tpl", templateVars); err != nil {
		return b.Bytes(), err
	}

	return b.Bytes(), nil
}

func releaseNoteVariables(releasenote sv.ReleaseNote) releaseNoteTemplateVariables {
	release := releasenote.Tag
	if releasenote.Version != nil {
		release = "v" + releasenote.Version.String()
	}

	return releaseNoteTemplateVariables{
		Release:     release,
		Tag:         releasenote.Tag,
		Version:     releasenote.Version,
		Date:        releasenote.Date,
		Sections:    releasenote.Sections,
		AuthorNames: toSortedArray(releasenote.AuthorsNames),
	}
}

func toSortedArray(input map[string]struct{}) []string {
	result := make([]string, len(input))
	i := 0

	for k := range input {
		result[i] = k
		i++
	}

	sort.Strings(result)

	return result
}
