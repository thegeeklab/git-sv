# git-sv

Semantic versioning tool for git based on conventional commits.

[![Build Status](https://ci.thegeeklab.de/api/badges/thegeeklab/git-sv/status.svg)](https://ci.thegeeklab.de/repos/thegeeklab/git-sv)
[![Go Report Card](https://goreportcard.com/badge/github.com/thegeeklab/git-sv)](https://goreportcard.com/report/github.com/thegeeklab/git-sv)
[![GitHub contributors](https://img.shields.io/github/contributors/thegeeklab/git-sv)](https://github.com/thegeeklab/git-sv/graphs/contributors)
[![License: MIT](https://img.shields.io/github/license/thegeeklab/git-sv)](https://github.com/thegeeklab/git-sv/blob/main/LICENSE)

## Requirements

- Git 2.17+

## Installation

Prebuilt multi-arch binaries are available for Linux and macOS.

```Shell
curl -SsfL https://github.com/thegeeklab/git-sv/releases/latest/download/git-sv-linux-amd64 -o /usr/local/bin/git-sv
chmod +x /usr/local/bin/git-sv
```

## Build

Build the binary from source with the following command:

```Shell
make build
```

## Configuration

The configuration is loaded from a YAML file in the following order (last wins):

- built-in default
- `.gitsv/config.yml` in repository root

To check the default configuration, run:

```Shell
git sv cfg default
```

```Yaml
versioning:
  update-major: [] # Commit types used to bump major.
  update-minor: [feat] # Commit types used to bump minor.
  update-patch: [build, ci, chore, fix, perf, refactor, test] # Commit types used to bump patch.
  # When type is not present on update rules and is unknown (not mapped on commit message types);
  # if ignore-unknown=false bump patch, if ignore-unknown=true do not bump version.
  ignore-unknown: false

tag:
  pattern: "%d.%d.%d" # Pattern used to create git tag.
  filter: "" # Enables you to filter for considerable tags using git pattern syntax.

release-notes:
  sections: # Array with each section of release note. Check template section for more information.
    - name: Features # Name used on section.
      section-type: commits # Type of the section, supported types: commits, breaking-changes.
      commit-types: [feat] # Commit types for commit section-type, one commit type cannot be in more than one section.
    - name: Bug Fixes
      section-type: commits
      commit-types: [fix]
    - name: Breaking Changes
      section-type: breaking-changes

branches: # Git branches config.
  prefix: ([a-z]+\/)? # Prefix used on branch name, it should be a regex group.
  suffix: (-.*)? # Suffix used on branch name, it should be a regex group.
  disable-issue: false # Set true if there is no need to recover issue id from branch name.
  skip: [master, main, developer] # List of branch names ignored on commit message validation.
  skip-detached: false # Set true if a detached branch should be ignored on commit message validation.

commit-message:
  # Supported commit types.
  types: [
      build,
      ci,
      chore,
      docs,
      feat,
      fix,
      perf,
      refactor,
      revert,
      style,
      test,
    ]
  header-selector: "" # You can put in a regex here to select only a certain part of the commit message. Please define a regex group 'header'.
  scope:
    # Define supported scopes, if blank, scope will not be validated, if not, only scope listed will be valid.
    # Don't forget to add "" on your list if you need to define scopes and keep it optional.
    values: []
  footer:
    issue: # Use "issue: {}" if you wish to disable issue footer.
      key: jira # Name used to define an issue on footer metadata.
      key-synonyms: [Jira, JIRA] # Supported variations for footer metadata.
      use-hash: false # If false, use :<space> separator. If true, use <space># separator.
      add-value-prefix: "" # Add a prefix to issue value.
  issue:
    regex: "[A-Z]+-[0-9]+" # Regex for issue id.
```

### Templates

**git-sv** uses _go templates_ to format the output for `release-notes` and `changelog`, to see how the default template is configured check [template directory](https://github.com/thegeeklab/git-sv/tree/main/templates/assets). It's possible to overwrite the default configuration by adding `.gitsv/templates` on your repository.

```Shell
.gitsv
└── templates
    ├── changelog-md.tpl
    └── releasenotes-md.tpl
```

Everything inside `.gitsv/templates` will be loaded, so it's possible to add more files to be used as needed.

#### Variables

To execute the template the `releasenotes-md.tpl` will receive a single `ReleaseNote` and `changelog-md.tpl` will receive a list of `ReleaseNote` as variables.

Each `ReleaseNoteSection` will be configured according with `release-notes.section` from configuration file. The order for each section will be maintained and the `SectionType` is defined according with `section-type` attribute as described on the table below.

| section-type     | ReleaseNoteSection               |
| ---------------- | -------------------------------- |
| commits          | ReleaseNoteCommitsSection        |
| breaking-changes | ReleaseNoteBreakingChangeSection |

> :warning: currently only `commits` and `breaking-changes` are supported as `section-types`, using a different value for this field will make the section to be removed from the template variables.

## Usage

Use `--help` or `-h` to get usage information, don't forget that some commands have unique options too:

```Shell
$ git-sv --help
NAME:
   git-sv - Semantic version for git.

USAGE:
   git-sv [global options] command [command options] [arguments...]

VERSION:
   20e64f8

COMMANDS:
   config, cfg                   cli configuration
   current-version, cv           get last released version from git
   next-version, nv              generate the next version based on git commit messages
   commit-log, cl                list all commit logs according to range as json
   commit-notes, cn              generate a commit notes according to range
   release-notes, rn             generate release notes
   changelog, cgl                generate changelog
   tag, tg                       generate tag with version based on git commit messages
   commit, cmt                   execute git commit with conventional commit message helper
   validate-commit-message, vcm  use as prepare-commit-message hook to validate and enhance commit message
   help, h                       Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

If `git-sv` is configured on your path, you can also use it like a git command.

```Shell
git sv
git sv current-version
git sv next-version
```

### Ranges

Commands like `commit-log` and `commit-notes` has a range option. Supported range types are: `tag`, `date` and `hash`.

By default, it's used [--date=short](https://git-scm.com/docs/git-log#Documentation/git-log.txt---dateltformatgt) at `git log`, all dates returned from it will be in `YYYY-MM-DD` format.

Range `tag` will use `git for-each-ref refs/tags` to get the last tag available if `start` is empty, the others types won't use the existing tags. It's recommended to always use a start limit in an old repository with a lot of commits.

Range `date` use git log `--since` and `--until`. It's possible to use all supported formats from [git log](https://git-scm.com/docs/git-log#Documentation/git-log.txt---sinceltdategt). If `end` is in `YYYY-MM-DD` format, `sv` will add a day on git log command to make the end date inclusive.

Range `tag` and `hash` are used on git log [revision range](https://git-scm.com/docs/git-log#Documentation/git-log.txt-ltrevisionrangegt). If `end` is empty, `HEAD` will be used instead.

```Shell
# get commit log as json using a inclusive range
git-sv commit-log --range hash --start 7ea9306~1 --end c444318

# return all commits after last tag
git-sv commit-log --range tag
```

## Contributors

Special thanks to all [contributors](https://github.com/thegeeklab/git-sv/graphs/contributors). If you would like to contribute, please see the [instructions](https://github.com/thegeeklab/git-sv/blob/main/CONTRIBUTING.md).

This project is a fork of [sv4git](https://github.com/bvieira/sv4git) from Beatriz Vieira. Thanks for your work.

## License

This project is licensed under the MIT License - see the [LICENSE](https://github.com/thegeeklab/git-sv/blob/main/LICENSE) file for details.
