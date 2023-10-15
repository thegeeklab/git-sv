package templates

import (
	"embed"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/rs/zerolog/log"
	"github.com/thegeeklab/git-sv/v2/sv"
)

//go:embed assets
var templateFs embed.FS

// New loads the template to make it parseable.
func New(configDir string) *template.Template {
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg("error while retrieving working directory")
	}

	tplsDir := filepath.Join(workDir, configDir, "templates")

	tpls, err := template.New("templates").Funcs(Funcs()).ParseFS(templateFs, "**/*.tpl")
	if err != nil {
		log.Warn().
			Err(err).
			Msg("Failed to parse builtin templates")
	}

	custom, _ := filepath.Glob(filepath.Join(tplsDir, "*.tpl"))
	if len(custom) == 0 {
		return tpls
	}

	for _, v := range custom {
		tpls, err = template.New("templates").Funcs(Funcs()).ParseFiles(v)
		if err != nil {
			log.Warn().
				Err(err).
				Str("filename", v).
				Msg("Failed to parse custom template")
		}
	}

	return tpls
}

// Funcs provides some general usefule template helpers.
func Funcs() template.FuncMap {
	functs := sprig.FuncMap()

	functs["date"] = zeroDate
	// functs["getsection"] = getSection

	return functs
}

func zeroDate(fmt string, date time.Time) string {
	if date.IsZero() {
		return ""
	}

	return date.Format(fmt)
}

func getSection(name string, sections []sv.ReleaseNoteSection) sv.ReleaseNoteSection { //nolint:ireturn
	for _, section := range sections {
		if section.SectionName() == name {
			return section
		}
	}

	return nil
}
