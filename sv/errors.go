package sv

import "errors"

var (
	errUnknownGitError      = errors.New("git command failed")
	errInvalidCommitMessage = errors.New("commit message not valid")
	errIssueIDNotFound      = errors.New("could not find issue id using configured regex")
	errInvalidIssueRegex    = errors.New("could not compile issue regex")
	errInvalidHeaderRegex   = errors.New("invalid regex on header-selector")
)
