package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
)

type testRepo struct {
	commits     int  // Total number of commits to create
	tags        int  // Number of tags to create
	setupRemote bool // Whether to set up a remote repository
}

// setupGitRepo creates a temporary git repository with optional commit history
// and returns the temp directory path
func setupGitRepo(t *testing.T, tr testRepo) string {
	t.Helper()

	// Create a temporary directory using t.TempDir() which is automatically cleaned up
	tmpDir := t.TempDir()

	// Change to the temporary directory
	t.Chdir(tmpDir)
	// Initialize git repository
	runGitCommand(t, "init")
	runGitCommand(t, "config", "user.email", "test@example.com")
	runGitCommand(t, "config", "user.name", "Test User")

	// Create a test file and commit it (initial commit)
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	runGitCommand(t, "add", "test.txt")
	runGitCommand(t, "commit", "-m", "Initial commit")

	// Calculate commits per tag
	commitsPerTag := 0
	if tr.tags > 0 {
		commitsPerTag = tr.commits / tr.tags
	}

	// Create additional commits and tags
	for i := 0; i < tr.commits; i++ {
		commitType := "feat"
		if i%2 == 0 {
			commitType = "fix"
		}

		testFile := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		err := os.WriteFile(testFile, []byte(fmt.Sprintf("content %d", i)), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		runGitCommand(t, "add", testFile)
		runGitCommand(t, "commit", "-m", fmt.Sprintf("%s: add file %d", commitType, i))

		// Create a tag if needed
		if tr.tags > 0 && commitsPerTag > 0 && (i+1)%commitsPerTag == 0 {
			tagNumber := (i + 1) / commitsPerTag
			if tagNumber <= tr.tags {
				runGitCommand(t, "tag", fmt.Sprintf("v%d.0.0", tagNumber))
			}
		}
	}

	// If we need to set up a remote repository
	if tr.setupRemote {
		// Create a bare repository to act as a remote
		remoteDir := t.TempDir()
		runGitCommand(t, "init", "--bare", remoteDir)

		// Add the remote to the local repo
		runGitCommand(t, "remote", "add", "origin", remoteDir)

		// Make an initial push to the remote
		runGitCommand(t, "push", "origin", "main")
	}

	return tmpDir
}

// Helper function to run git commands
func runGitCommand(t *testing.T, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Git command failed: %v\nCommand: git %v\nOutput: %s", err, args, output)
	}
}

func date(input string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05 -0700", input)
	if err != nil {
		panic(err)
	}

	return t
}

func TestLastTag(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T)
		filter    string
		want      string
	}{
		{
			name:      "no tags",
			setupFunc: func(t *testing.T) {},
			filter:    "",
			want:      "",
		},
		{
			name: "single tag",
			setupFunc: func(t *testing.T) {
				runGitCommand(t, "tag", "v1.0.0")
			},
			filter: "",
			want:   "v1.0.0",
		},
		{
			name: "multiple tags",
			setupFunc: func(t *testing.T) {
				runGitCommand(t, "tag", "v1.0.0")
				runGitCommand(t, "tag", "v2.0.0")
			},
			filter: "",
			want:   "v2.0.0",
		},
		{
			name: "with tag filter",
			setupFunc: func(t *testing.T) {
				runGitCommand(t, "tag", "v1.0.0")
				runGitCommand(t, "tag", "v2.0.0")
			},
			filter: "v1*",
			want:   "v1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup git repository with minimal configuration
			_ = setupGitRepo(t, testRepo{})

			// Run the test-specific setup
			tt.setupFunc(t)

			// Test with the specified filter
			g := New()
			g.Config.Tag.Filter = &tt.filter

			got := g.LastTag()
			assert.Equal(t, tt.want, got, "Expected %s as the last tag", tt.want)
		})
	}
}

func TestLastTagWithMultipleVersions(t *testing.T) {
	tests := []struct {
		name   string
		tags   []string
		filter string
		want   string
	}{
		{
			name:   "non-sequential versions",
			tags:   []string{"v1.0.0", "v0.9.0", "v1.1.0", "v0.8.0"},
			filter: "",
			want:   "v1.1.0",
		},
		{
			name:   "with filter",
			tags:   []string{"v1.0.0", "v0.9.0", "v1.1.0", "v0.8.0"},
			filter: "v0*",
			want:   "v0.9.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup git repository with minimal configuration
			_ = setupGitRepo(t, testRepo{})

			// Create tags
			for _, tag := range tt.tags {
				runGitCommand(t, "tag", tag)
			}

			// Test with the specified filter
			g := New()
			g.Config.Tag.Filter = &tt.filter

			got := g.LastTag()
			assert.Equal(t, tt.want, got, "Expected %s as the last tag", tt.want)
		})
	}
}

func TestLog(t *testing.T) {
	tests := []struct {
		name     string
		logRange LogRange
		repo     testRepo
		want     int
		wantErr  bool
	}{
		{
			name:     "empty range",
			logRange: NewLogRange(HashRange, "", ""),
			repo:     testRepo{commits: 3},
			want:     4, // 3 commits + initial commit
			wantErr:  false,
		},
		{
			name:     "hash range with start and end",
			logRange: NewLogRange(HashRange, "HEAD~2", "HEAD"),
			repo:     testRepo{commits: 3},
			want:     2,
			wantErr:  false,
		},
		{
			name:     "hash range with only end",
			logRange: NewLogRange(HashRange, "", "HEAD~1"),
			repo:     testRepo{commits: 3},
			want:     3, // Initial commit + first 2 commits
			wantErr:  false,
		},
		{
			name:     "date range",
			logRange: NewLogRange(DateRange, time.Now().AddDate(0, 0, -1).Format("2006-01-02"), time.Now().Format("2006-01-02")),
			repo:     testRepo{commits: 3},
			want:     4, // All commits should be from today
			wantErr:  false,
		},
		{
			name:     "tag range",
			logRange: NewLogRange(TagRange, "v1.0.0", "v2.0.0"),
			repo:     testRepo{commits: 6, tags: 2},
			want:     3, // Commits between v1.0.0 and v2.0.0
			wantErr:  false,
		},
		{
			name:     "invalid git command",
			logRange: NewLogRange(HashRange, "invalid-ref", ""),
			repo:     testRepo{commits: 1},
			want:     0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup git repository with the specified number of commits and tags
			_ = setupGitRepo(t, tt.repo)

			// Create GitSV instance
			g := New()

			// Call the Log function
			logs, err := g.Log(tt.logRange)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err, "Log() returned an error: %v", err)
			assert.Equal(t, tt.want, len(logs), "Log() returned %d logs, want %d", len(logs), tt.want)

			// Verify log structure
			for _, log := range logs {
				// Check that required fields are present
				assert.NotEmpty(t, log.Hash, "Log() returned log with empty hash")
				assert.NotEmpty(t, log.Date, "Log() returned log with empty date")
				assert.NotEmpty(t, log.AuthorName, "Log() returned log with empty author name")
			}
		})
	}
}

func TestCommit(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		body    string
		footer  string
		repo    testRepo
		wantErr bool
	}{
		{
			name:    "valid commit",
			header:  "feat: add new feature",
			body:    "This is the body of the commit message\nIt can span multiple lines",
			footer:  "Refs: #123",
			repo:    testRepo{commits: 1},
			wantErr: false,
		},
		{
			name:    "commit with empty body",
			header:  "fix: fix a bug",
			body:    "",
			footer:  "Closes: #456",
			repo:    testRepo{commits: 1},
			wantErr: false,
		},
		{
			name:    "commit with empty footer",
			header:  "docs: update documentation",
			body:    "Update the README with new information",
			footer:  "",
			repo:    testRepo{commits: 1},
			wantErr: false,
		},
		{
			name:    "commit with all empty parts",
			header:  "",
			body:    "",
			footer:  "",
			repo:    testRepo{commits: 1},
			wantErr: true, // Git will reject a commit with an empty message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup git repository
			repo := setupGitRepo(t, tt.repo)

			// Create a new file to commit
			testFile := filepath.Join(repo, "commit_test.txt")
			err := os.WriteFile(testFile, []byte("test content for commit"), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
			runGitCommand(t, "add", "commit_test.txt")

			// Create GitSV instance
			g := New()

			// Call the Commit function
			err = g.Commit(tt.header, tt.body, tt.footer)

			// Check error
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify the commit was created with the correct message
				cmd := exec.Command("git", "log", "-1", "--pretty=%B")
				output, err := cmd.CombinedOutput()
				assert.NoError(t, err)

				// The commit message should contain the header, body, and footer
				commitMsg := string(output)
				assert.Contains(t, commitMsg, tt.header)

				if tt.body != "" {
					assert.Contains(t, commitMsg, tt.body)
				}

				if tt.footer != "" {
					assert.Contains(t, commitMsg, tt.footer)
				}
			}
		})
	}
}

func TestTags(t *testing.T) {
	tests := []struct {
		name       string
		repo       testRepo
		tagFilter  string
		extraSetup func(t *testing.T)
		want       int
		wantErr    bool
	}{
		{
			name:      "no tags",
			repo:      testRepo{commits: 3},
			tagFilter: "",
			want:      0,
			wantErr:   false,
		},
		{
			name:      "multiple tags",
			repo:      testRepo{commits: 5, tags: 3},
			tagFilter: "",
			want:      3,
			wantErr:   false,
		},
		{
			name:      "with tag filter",
			repo:      testRepo{},
			tagFilter: "v1*",
			extraSetup: func(t *testing.T) {
				runGitCommand(t, "tag", "v1.0.0")
				runGitCommand(t, "tag", "v1.1.0")
				runGitCommand(t, "tag", "v2.0.0")
			},
			want:    2, // Only v1.0.0 and v1.1.0 should match
			wantErr: false,
		},
		{
			name:      "with annotated tags",
			repo:      testRepo{},
			tagFilter: "",
			extraSetup: func(t *testing.T) {
				// Create annotated tags
				runGitCommand(t, "tag", "-a", "v1.0.0", "-m", "Version 1.0.0")
				runGitCommand(t, "tag", "-a", "v1.1.0", "-m", "Version 1.1.0")
			},
			want:    2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup git repository
			_ = setupGitRepo(t, tt.repo)

			// Run any extra setup if provided
			if tt.extraSetup != nil {
				tt.extraSetup(t)
			}

			// Create GitSV instance
			g := New()
			g.Config.Tag.Filter = &tt.tagFilter

			// Call the Tags function
			tags, err := g.Tags()

			// Check error
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, len(tags), "Expected %d tags, got %d", tt.want, len(tags))

			// Verify tag structure
			for _, tag := range tags {
				assert.NotEmpty(t, tag.Name, "Tag name should not be empty")
			}

			// For the tag filter test, verify that only matching tags are returned
			if tt.tagFilter == "v1*" {
				for _, tag := range tags {
					assert.True(t, strings.HasPrefix(tag.Name, "v1"),
						"Tag %s should match filter %s", tag.Name, tt.tagFilter)
				}
			}
		})
	}
}

func TestBranch(t *testing.T) {
	tests := []struct {
		name       string
		repo       testRepo
		extraSetup func(t *testing.T)
		want       string
	}{
		{
			name: "default branch",
			repo: testRepo{commits: 1},
			want: "main",
		},
		{
			name: "new branch",
			repo: testRepo{commits: 1},
			extraSetup: func(t *testing.T) {
				runGitCommand(t, "checkout", "-b", "feature/new-feature")
			},
			want: "feature/new-feature",
		},
		{
			name: "detached HEAD",
			repo: testRepo{commits: 3},
			extraSetup: func(t *testing.T) {
				// Checkout a specific commit to create a detached HEAD state
				runGitCommand(t, "checkout", "HEAD~1")
			},
			want: "", // Branch() should return empty string for detached HEAD
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup git repository
			_ = setupGitRepo(t, tt.repo)

			// Run any extra setup if provided
			if tt.extraSetup != nil {
				tt.extraSetup(t)
			}

			// Create GitSV instance
			g := New()

			// Call the Branch function
			got := g.Branch()

			// Check result
			assert.Equal(t, tt.want, got, "Expected branch name %q, got %q", tt.want, got)
		})
	}
}

func TestIsDetached(t *testing.T) {
	tests := []struct {
		name       string
		repo       testRepo
		extraSetup func(t *testing.T)
		want       bool
		wantErr    bool
	}{
		{
			name:    "on branch (not detached)",
			repo:    testRepo{commits: 1},
			want:    false,
			wantErr: false,
		},
		{
			name: "on new branch (not detached)",
			repo: testRepo{commits: 1},
			extraSetup: func(t *testing.T) {
				runGitCommand(t, "checkout", "-b", "feature/new-feature")
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "detached HEAD",
			repo: testRepo{commits: 3},
			extraSetup: func(t *testing.T) {
				// Checkout a specific commit to create a detached HEAD state
				runGitCommand(t, "checkout", "HEAD~1")
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "detached HEAD at tag",
			repo: testRepo{commits: 3, tags: 1},
			extraSetup: func(t *testing.T) {
				// Checkout a tag to create a detached HEAD state
				runGitCommand(t, "checkout", "v1.0.0")
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup git repository
			_ = setupGitRepo(t, tt.repo)

			// Run any extra setup if provided
			if tt.extraSetup != nil {
				tt.extraSetup(t)
			}

			// Create GitSV instance
			g := New()

			// Call the IsDetached function
			got, err := g.IsDetached()

			// Check error
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got, "Expected IsDetached() to return %v, got %v", tt.want, got)
		})
	}
}

func TestTag(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		annotate   bool
		local      bool
		repo       testRepo
		extraSetup func(t *testing.T)
		want       string
		wantErr    bool
	}{
		{
			name:     "simple tag",
			version:  "1.0.0",
			annotate: false,
			local:    true,
			repo:     testRepo{commits: 1},
			want:     "1.0.0",
			wantErr:  false,
		},
		{
			name:     "annotated tag",
			version:  "2.1.0",
			annotate: true,
			local:    true,
			repo:     testRepo{commits: 1},
			want:     "2.1.0",
			wantErr:  false,
		},
		{
			name:     "tag already exists",
			version:  "1.0.0",
			annotate: false,
			local:    true,
			repo:     testRepo{commits: 1},
			extraSetup: func(t *testing.T) {
				// Create a tag that will conflict
				runGitCommand(t, "tag", "1.0.0")
			},
			want:    "1.0.0",
			wantErr: true, // Should fail because tag already exists
		},
		{
			name:     "push to remote",
			version:  "1.0.0",
			annotate: false,
			local:    false, // Try to push
			repo:     testRepo{commits: 1, setupRemote: true},
			want:     "1.0.0",
			wantErr:  false, // Should succeed because we set up a remote
		},
		{
			name:     "push attempt without remote",
			version:  "1.0.0",
			annotate: false,
			local:    false, // Try to push
			repo:     testRepo{commits: 1},
			want:     "1.0.0",
			wantErr:  true, // Should fail because there's no remote
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup git repository
			_ = setupGitRepo(t, tt.repo)

			// Run any extra setup if provided
			if tt.extraSetup != nil {
				tt.extraSetup(t)
			}

			// Create GitSV instance
			g := New()

			// If no custom pattern was set in extraSetup, use default
			if g.Config.Tag.Pattern == nil {
				pattern := "%d.%d.%d"
				g.Config.Tag.Pattern = &pattern
			}

			// Parse version
			semverVersion, err := semver.NewVersion(tt.version)
			assert.NoError(t, err, "Failed to parse version")

			// Call the Tag function
			tag, err := g.Tag(*semverVersion, tt.annotate, tt.local)

			// Check error
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, tag, "Expected tag %q, got %q", tt.want, tag)

			// Verify the tag was created
			cmd := exec.Command("git", "tag", "-l", tag)
			output, err := cmd.CombinedOutput()
			assert.NoError(t, err)
			assert.Contains(t, string(output), tag, "Tag %q should exist", tag)

			// If annotated, verify it's an annotated tag
			if tt.annotate {
				cmd := exec.Command("git", "tag", "-l", "-n", tag)
				output, err := cmd.CombinedOutput()
				assert.NoError(t, err)

				// Annotated tags should include the message
				expectedMsg := fmt.Sprintf("Version %d.%d.%d",
					semverVersion.Major(), semverVersion.Minor(), semverVersion.Patch())
				assert.Contains(t, string(output), expectedMsg,
					"Annotated tag should contain message %q", expectedMsg)
			}

			// If we tried to push, verify the tag was pushed to the remote
			if !tt.local && tt.repo.setupRemote {
				cmd := exec.Command("git", "ls-remote", "--tags", "origin", tag)
				output, err := cmd.CombinedOutput()
				assert.NoError(t, err)
				assert.Contains(t, string(output), tag, "Tag %q should exist in remote", tag)
			}
		})
	}
}
