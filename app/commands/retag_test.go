package commands

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thegeeklab/git-sv/app"
)

// runGit runs a git command inside the current working directory and returns
// its trimmed combined output.
func runGit(t *testing.T, env []string, args ...string) string {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), "git", args...)
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}

	return strings.TrimSpace(string(out))
}

// setupRetagRepo initializes an empty git repository in a temp dir, chdirs into
// it and returns the path.
func setupRetagRepo(t *testing.T) string {
	t.Helper()

	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")

	dir := t.TempDir()
	t.Chdir(dir)

	runGit(t, nil, "init", "-b", "main")
	runGit(t, nil, "config", "user.email", "test@example.com")
	runGit(t, nil, "config", "user.name", "Test User")
	runGit(t, nil, "config", "commit.gpgsign", "false")
	runGit(t, nil, "config", "tag.gpgSign", "false")

	return dir
}

// commit creates a file, commits it with a fixed author/committer date and
// returns the resulting commit hash.
func commit(t *testing.T, dir, name, date string) string {
	t.Helper()

	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644))
	runGit(t, nil, "add", ".")

	env := []string{"GIT_AUTHOR_DATE=" + date, "GIT_COMMITTER_DATE=" + date}
	runGit(t, env, "commit", "-m", "chore: add "+name)

	return runGit(t, nil, "rev-parse", "HEAD")
}

func runRetag(t *testing.T, settings *app.RetagSettings) error {
	t.Helper()

	gsv := app.New()

	return RetagHandler(gsv, settings)(context.Background(), nil)
}

// setupRemote initializes a bare git repository in a temp dir, adds it as
// `origin` to the current test repo and returns the bare repo path.
func setupRemote(t *testing.T) string {
	t.Helper()

	remoteDir := filepath.Join(t.TempDir(), "remote.git")
	runGit(t, nil, "init", "--bare", remoteDir)
	runGit(t, nil, "remote", "add", "origin", remoteDir)

	return remoteDir
}

func TestRetagFlags(t *testing.T) {
	flags := RetagFlags(&app.RetagSettings{})
	assert.NotEmpty(t, flags)
}

func TestRetagHandler(t *testing.T) {
	gsv := app.New()
	settings := &app.RetagSettings{}
	handler := RetagHandler(gsv, settings)
	assert.NotNil(t, handler)
}

func TestRetagHandlerNoTags(t *testing.T) {
	dir := setupRetagRepo(t)
	commit(t, dir, "a.txt", "2020-01-01T00:00:00")

	err := runRetag(t, &app.RetagSettings{Local: true})
	assert.ErrorIs(t, err, errNoTagToRetag)
}

func TestRetagHandlerSpecificTagNotFound(t *testing.T) {
	dir := setupRetagRepo(t)
	commit(t, dir, "a.txt", "2020-01-01T00:00:00")
	runGit(t, nil, "tag", "1.0.0")

	err := runRetag(t, &app.RetagSettings{Local: true, Tag: "9.9.9"})

	// A specific tag that does not exist must fail instead of silently
	// creating a fresh tag at HEAD.
	assert.ErrorIs(t, err, errTagNotFound)

	tags := runGit(t, nil, "tag", "--list")
	assert.NotContains(t, tags, "9.9.9")
}

func TestRetagHandlerMostRecentByDate(t *testing.T) {
	dir := setupRetagRepo(t)

	// Highest semver tag is created on the oldest commit, while a lower version
	// is tagged more recently (e.g. a backport).
	commit(t, dir, "a.txt", "2020-01-01T00:00:00")
	runGit(t, nil, "tag", "2.0.0")

	commit(t, dir, "b.txt", "2021-01-01T00:00:00")
	runGit(t, nil, "tag", "1.5.0")

	head := commit(t, dir, "c.txt", "2022-01-01T00:00:00")

	err := runRetag(t, &app.RetagSettings{Local: true})
	require.NoError(t, err)

	// The chronologically newest tag (1.5.0) is moved to HEAD, the higher
	// semver tag (2.0.0) is left untouched.
	assert.Equal(t, head, runGit(t, nil, "rev-parse", "1.5.0"))
	assert.NotEqual(t, head, runGit(t, nil, "rev-parse", "2.0.0"))
}

func TestRetagHandlerPreservesAnnotatedType(t *testing.T) {
	dir := setupRetagRepo(t)

	commit(t, dir, "a.txt", "2020-01-01T00:00:00")

	env := []string{"GIT_COMMITTER_DATE=2020-01-01T00:00:00", "GIT_AUTHOR_DATE=2020-01-01T00:00:00"}
	runGit(t, env, "tag", "-a", "-m", "release 1.0.0", "1.0.0")

	commit(t, dir, "b.txt", "2021-01-01T00:00:00")

	// Retag without -a must not downgrade the annotated tag to a lightweight one.
	err := runRetag(t, &app.RetagSettings{Local: true})
	require.NoError(t, err)

	assert.Equal(t, "tag", runGit(t, nil, "cat-file", "-t", "1.0.0"))
}

func TestRetagHandlerPreservesVPrefix(t *testing.T) {
	dir := setupRetagRepo(t)
	commit(t, dir, "a.txt", "2020-01-01T00:00:00")
	runGit(t, nil, "tag", "v1.0.0")

	head := commit(t, dir, "b.txt", "2021-01-01T00:00:00")

	err := runRetag(t, &app.RetagSettings{Local: true, Tag: "v1.0.0"})
	require.NoError(t, err)

	// The v-prefixed tag must move to HEAD, and the default-pattern tag must
	// not be silently created.
	assert.Equal(t, head, runGit(t, nil, "rev-parse", "v1.0.0"))
	assert.Equal(t, "v1.0.0", runGit(t, nil, "tag", "--list"))
}

func TestRetagHandlerPreservesPrerelease(t *testing.T) {
	dir := setupRetagRepo(t)
	commit(t, dir, "a.txt", "2020-01-01T00:00:00")
	runGit(t, nil, "tag", "1.0.0-rc1")

	head := commit(t, dir, "b.txt", "2021-01-01T00:00:00")

	err := runRetag(t, &app.RetagSettings{Local: true, Tag: "1.0.0-rc1"})
	require.NoError(t, err)

	// The pre-release suffix must be preserved; no "1.0.0" tag must be created.
	assert.Equal(t, head, runGit(t, nil, "rev-parse", "1.0.0-rc1"))
	assert.Equal(t, "1.0.0-rc1", runGit(t, nil, "tag", "--list"))
}

func TestRetagHandlerNoopWhenTagAtHead(t *testing.T) {
	dir := setupRetagRepo(t)
	commit(t, dir, "a.txt", "2020-01-01T00:00:00")

	env := []string{"GIT_COMMITTER_DATE=2020-01-01T00:00:00", "GIT_AUTHOR_DATE=2020-01-01T00:00:00"}
	runGit(t, env, "tag", "-a", "-m", "release", "1.0.0")

	// Capture the annotated tag object identity; if the handler deletes and
	// recreates it, the tag object hash will change.
	originalTagObj := runGit(t, nil, "rev-parse", "1.0.0^{tag}")

	err := runRetag(t, &app.RetagSettings{Local: true})
	require.NoError(t, err)

	// No-op: the underlying tag object must be untouched.
	assert.Equal(t, originalTagObj, runGit(t, nil, "rev-parse", "1.0.0^{tag}"))
}

func TestRetagHandlerPushesToRemote(t *testing.T) {
	dir := setupRetagRepo(t)
	remoteDir := setupRemote(t)

	commit(t, dir, "a.txt", "2020-01-01T00:00:00")
	runGit(t, nil, "tag", "1.0.0")
	head := commit(t, dir, "b.txt", "2021-01-01T00:00:00")

	require.NoError(t, runRetag(t, &app.RetagSettings{}))

	remoteCommit := runGit(t, []string{"GIT_DIR=" + remoteDir}, "rev-parse", "1.0.0^{commit}")
	assert.Equal(t, head, remoteCommit)
}

func TestRetagHandlerReconcilesStaleRemote(t *testing.T) {
	dir := setupRetagRepo(t)
	remoteDir := setupRemote(t)

	oldHead := commit(t, dir, "a.txt", "2020-01-01T00:00:00")
	runGit(t, nil, "tag", "1.0.0")
	runGit(t, nil, "push", "origin", "1.0.0")

	// Confirm the remote is stale before retag runs.
	assert.Equal(t, oldHead, runGit(t, []string{"GIT_DIR=" + remoteDir}, "rev-parse", "1.0.0^{commit}"))

	newHead := commit(t, dir, "b.txt", "2021-01-01T00:00:00")
	runGit(t, nil, "tag", "-f", "1.0.0", newHead)

	require.NoError(t, runRetag(t, &app.RetagSettings{}))

	remoteCommit := runGit(t, []string{"GIT_DIR=" + remoteDir}, "rev-parse", "1.0.0^{commit}")
	assert.Equal(t, newHead, remoteCommit)
}

func TestRetagHandlerNoRemote(t *testing.T) {
	dir := setupRetagRepo(t)
	commit(t, dir, "a.txt", "2020-01-01T00:00:00")
	runGit(t, nil, "tag", "1.0.0")
	commit(t, dir, "b.txt", "2021-01-01T00:00:00")

	err := runRetag(t, &app.RetagSettings{})
	require.Error(t, err)
}

func TestRetagHandlerNonSemverTag(t *testing.T) {
	dir := setupRetagRepo(t)
	commit(t, dir, "a.txt", "2020-01-01T00:00:00")
	runGit(t, nil, "tag", "release-2024")
	head := commit(t, dir, "b.txt", "2021-01-01T00:00:00")

	err := runRetag(t, &app.RetagSettings{Local: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "release-2024")

	// The non-semver tag must not be moved to HEAD.
	assert.NotEqual(t, head, runGit(t, nil, "rev-parse", "release-2024"))
}

func TestRetagHandlerUpgradesLightweightToAnnotated(t *testing.T) {
	dir := setupRetagRepo(t)
	commit(t, dir, "a.txt", "2020-01-01T00:00:00")
	runGit(t, nil, "tag", "1.0.0")
	commit(t, dir, "b.txt", "2021-01-01T00:00:00")

	require.NoError(t, runRetag(t, &app.RetagSettings{Local: true, Annotate: true}))

	assert.Equal(t, "tag", runGit(t, nil, "cat-file", "-t", "1.0.0"))
}

func TestRetagHandlerUpgradesLightweightToAnnotatedAtHead(t *testing.T) {
	dir := setupRetagRepo(t)
	commit(t, dir, "a.txt", "2020-01-01T00:00:00")
	runGit(t, nil, "tag", "1.0.0")

	// The tag is already at HEAD. The -a flag must still upgrade the
	// lightweight tag to annotated rather than silently no-op.
	require.NoError(t, runRetag(t, &app.RetagSettings{Local: true, Annotate: true}))

	assert.Equal(t, "tag", runGit(t, nil, "cat-file", "-t", "1.0.0"))
}
