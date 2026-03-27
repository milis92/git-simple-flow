package cmd

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/runner"
	"github.com/milis92/git-simple-flow/internal/ui"
)

func TestConfigEditDryRunDoesNotWriteConfig(t *testing.T) {
	scriptPath, err := exec.LookPath("script")
	if err != nil {
		t.Skip("script utility not available")
	}

	binary := buildReviewBinary(t)
	repoDir := initConfigEditRepo(t)
	configPath := filepath.Join(repoDir, ".sfconfig.yml")

	initial := "main_branch: main\n"
	if err := os.WriteFile(configPath, []byte(initial), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Answer the interactive wizard with values that would visibly change the file
	// if config edit incorrectly ignores --dry-run.
	inputs := strings.Join([]string{
		"2",
		"feat/",
		"fix/",
		"rel-",
		"3",
		"2",
		"y",
		"y",
		"",
	}, "\n")

	cmd := exec.Command(scriptPath, "-qec", binary+" --dry-run config edit", "/dev/null")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "TERM=dumb")
	cmd.Stdin = strings.NewReader(inputs)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("config edit command failed: %v\n%s", err, out)
	}

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(got) != initial {
		t.Fatalf("config file changed during --dry-run\noutput:\n%s\nwant:\n%s\ngot:\n%s", out, initial, got)
	}
}

func TestConfigEditYesSkipsWizard(t *testing.T) {
	scriptPath, err := exec.LookPath("script")
	if err != nil {
		t.Skip("script utility not available")
	}

	binary := buildReviewBinary(t)
	repoDir := initConfigEditRepo(t)
	configPath := filepath.Join(repoDir, ".sfconfig.yml")
	if err := os.WriteFile(configPath, []byte("main_branch: main\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, scriptPath, "-qec", binary+" --yes config edit", "/dev/null")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "TERM=dumb")

	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("config edit --yes hung waiting for wizard input\noutput:\n%s", out)
	}
	if err != nil {
		t.Fatalf("config edit --yes failed: %v\n%s", err, out)
	}
}

func TestConfigEditYesAllowsNonTTY(t *testing.T) {
	binary := buildReviewBinary(t)
	repoDir := initConfigEditRepo(t)
	configPath := filepath.Join(repoDir, ".sfconfig.yml")

	initial := "main_branch: main\n"
	if err := os.WriteFile(configPath, []byte(initial), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := exec.Command(binary, "--yes", "config", "edit")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "TERM=dumb")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("config edit --yes should not require a TTY when prompts are skipped: %v\n%s", err, out)
	}

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(got) != initial {
		t.Fatalf("config edit --yes changed config unexpectedly\noutput:\n%s\nwant:\n%s\ngot:\n%s", out, initial, got)
	}
}

func TestConfigEditYesCreatesUsableConfigWhenRepoFileMissing(t *testing.T) {
	repoDir := initConfigEditRepo(t)

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(origWD); chdirErr != nil {
			t.Fatalf("restore cwd: %v", chdirErr)
		}
	}()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	oldDryRun, oldVerbose := dryRun, verbose
	oldNoInteractive, oldAutoConfirm := noInteractive, autoConfirm
	dryRun = false
	verbose = false
	noInteractive = true
	autoConfirm = true
	defer func() {
		dryRun = oldDryRun
		verbose = oldVerbose
		noInteractive = oldNoInteractive
		autoConfirm = oldAutoConfirm
	}()

	if err := configEditCmd.RunE(configEditCmd, nil); err != nil {
		t.Fatalf("configEditCmd.RunE() error = %v", err)
	}

	configPath := filepath.Join(repoDir, ".sfconfig.yml")
	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(got), "main_branch:") {
		t.Fatalf("config edit should create a usable config file when none exists, got:\n%s", got)
	}
}

func TestConfigEditRequiresRepository(t *testing.T) {
	workDir := t.TempDir()

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(origWD); chdirErr != nil {
			t.Fatalf("restore cwd: %v", chdirErr)
		}
	}()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	oldDryRun, oldVerbose := dryRun, verbose
	oldNoInteractive, oldAutoConfirm := noInteractive, autoConfirm
	dryRun = false
	verbose = false
	noInteractive = true
	autoConfirm = true
	defer func() {
		dryRun = oldDryRun
		verbose = oldVerbose
		noInteractive = oldNoInteractive
		autoConfirm = oldAutoConfirm
	}()

	err = configEditCmd.RunE(configEditCmd, nil)
	if err == nil {
		t.Fatal("configEditCmd.RunE() error = nil, want repository validation failure")
	}

	configPath := filepath.Join(workDir, ".sfconfig.yml")
	if _, statErr := os.Stat(configPath); !os.IsNotExist(statErr) {
		t.Fatalf("config edit should not create %s outside a git repository, stat err = %v", configPath, statErr)
	}
}

func TestInitCmdRequiresRepository(t *testing.T) {
	workDir := t.TempDir()

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(origWD); chdirErr != nil {
			t.Fatalf("restore cwd: %v", chdirErr)
		}
	}()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	oldDryRun, oldVerbose := dryRun, verbose
	oldNoInteractive, oldAutoConfirm := noInteractive, autoConfirm
	dryRun = false
	verbose = false
	noInteractive = true
	autoConfirm = false
	defer func() {
		dryRun = oldDryRun
		verbose = oldVerbose
		noInteractive = oldNoInteractive
		autoConfirm = oldAutoConfirm
	}()

	if err := initCmd.Flags().Set("force", "false"); err != nil {
		t.Fatalf("Set(force) error = %v", err)
	}

	err = initCmd.RunE(initCmd, nil)
	if err == nil {
		t.Fatal("initCmd.RunE() error = nil, want repository validation failure")
	}

	configPath := filepath.Join(workDir, ".sfconfig.yml")
	if _, statErr := os.Stat(configPath); !os.IsNotExist(statErr) {
		t.Fatalf("init should not create %s outside a git repository, stat err = %v", configPath, statErr)
	}
}

func TestDetectBranchesDryRunUsesRealBranches(t *testing.T) {
	repoDir := initConfigEditRepo(t)
	runConfigGit(t, repoDir, "checkout", "-b", "trunk")

	var out bytes.Buffer
	r := runner.NewRunner(true, false)
	g := git.New(r, repoDir)
	u := ui.New()
	u.Out = &out

	branches := detectBranches(g, u)

	foundTrunk := false
	for _, branch := range branches {
		if branch == "trunk" {
			foundTrunk = true
			break
		}
	}
	if !foundTrunk {
		t.Fatalf("detectBranches() = %v, want real repo branch list to include %q during dry-run", branches, "trunk")
	}
	if strings.Contains(out.String(), "using defaults") {
		t.Fatalf("detectBranches() output = %q, should not fall back to defaults during dry-run", out.String())
	}
}

func TestConfigCommandShowsValidatedEffectiveConfig(t *testing.T) {
	binary := buildReviewBinary(t)
	repoDir := initConfigEditRepo(t)

	configPath := filepath.Join(repoDir, ".sfconfig.yml")
	if err := os.WriteFile(configPath, []byte("main_branch: \"   \"\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := exec.Command(binary, "config")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "TERM=dumb")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("config command failed: %v\n%s", err, out)
	}

	var mainBranchLine string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "main_branch") {
			mainBranchLine = line
			break
		}
	}
	if mainBranchLine == "" {
		t.Fatalf("config output did not contain main_branch line:\n%s", out)
	}

	fields := strings.Fields(mainBranchLine)
	if len(fields) < 3 || fields[1] != "main" || fields[2] != defaultConfigSource {
		t.Fatalf("main_branch line = %q, want validated effective value %q from %s", mainBranchLine, "main", defaultConfigSource)
	}
}

func buildReviewBinary(t *testing.T) string {
	t.Helper()

	moduleRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("Abs(..) error = %v", err)
	}

	binary := filepath.Join(t.TempDir(), "git-sf")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = moduleRoot
	cmd.Env = append(os.Environ(), "GOCACHE=/tmp/gocache", "GOPATH=/tmp/gopath")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}

	return binary
}

func initConfigEditRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	runConfigGit(t, repoDir, "init", "-b", "main")
	runConfigGit(t, repoDir, "config", "user.email", "test@example.com")
	runConfigGit(t, repoDir, "config", "user.name", "Test User")

	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("test\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	runConfigGit(t, repoDir, "add", "README.md")
	runConfigGit(t, repoDir, "commit", "-m", "init")

	return repoDir
}
