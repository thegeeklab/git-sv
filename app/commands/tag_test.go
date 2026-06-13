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

func setupTestRepo(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	runGitCmd(t, "init", "-b", "main")
	runGitCmd(t, "config", "user.email", "test@example.com")
	runGitCmd(t, "config", "user.name", "Test")
	runGitCmd(t, "config", "commit.gpgsign", "false")

	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("initial"), 0o644)
	require.NoError(t, err)

	runGitCmd(t, "add", "test.txt")
	runGitCmd(t, "commit", "-m", "feat: initial commit")
	runGitCmd(t, "tag", "1.0.0")

	err = os.WriteFile(testFile, []byte("updated"), 0o644)
	require.NoError(t, err)

	runGitCmd(t, "add", "test.txt")
	runGitCmd(t, "commit", "-m", "feat: add feature")
}

func runGitCmd(t *testing.T, args ...string) {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), "git", args...)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, output)
}

func gitTags(t *testing.T) []string {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), "git", "tag", "-l")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err)

	return strings.Fields(string(output))
}

func TestTagFlags(t *testing.T) {
	flags := TagFlags(&app.TagSettings{})
	assert.NotEmpty(t, flags)
}

func TestTagHandlerCreatesNextVersion(t *testing.T) {
	setupTestRepo(t)

	gsv := app.New()
	settings := &app.TagSettings{
		Force: false,
		Local: true,
	}

	handler := TagHandler(gsv, settings)
	err := handler(context.Background(), nil)

	assert.NoError(t, err)

	tags := gitTags(t)
	assert.Contains(t, tags, "1.1.0", "without --force, the next version 1.1.0 should be created")
}

func TestTagHandlerForceRecreatesCurrentTag(t *testing.T) {
	setupTestRepo(t)

	gsv := app.New()
	settings := &app.TagSettings{
		Force: true,
		Local: true,
	}

	handler := TagHandler(gsv, settings)
	err := handler(context.Background(), nil)

	assert.NoError(t, err)

	tags := gitTags(t)
	assert.Contains(t, tags, "1.0.0", "with --force, the current tag 1.0.0 should still exist")
	assert.NotContains(t, tags, "1.1.0", "with --force, the next version 1.1.0 should NOT be created")
}

func TestTagHandlerNoNewCommitsNothingToDo(t *testing.T) {
	setupTestRepo(t)

	runGitCmd(t, "tag", "1.1.0")

	gsv := app.New()
	settings := &app.TagSettings{
		Force: false,
		Local: true,
	}

	handler := TagHandler(gsv, settings)
	err := handler(context.Background(), nil)

	assert.NoError(t, err)

	tags := gitTags(t)
	assert.Len(t, tags, 2, "no new commits means no new tag should be created")
}

func TestTagHandlerForceNoTagExists(t *testing.T) {
	setupTestRepo(t)

	runGitCmd(t, "tag", "-d", "1.0.0")

	gsv := app.New()
	settings := &app.TagSettings{
		Force: true,
		Local: true,
	}

	handler := TagHandler(gsv, settings)
	err := handler(context.Background(), nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no existing tag to recreate")
}
