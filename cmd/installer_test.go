package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallerSupportsBSDStyleCoreutils(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the installer is exercised through bash")
	}

	for _, candidate := range []string{"/usr/bin/ut", "/usr/local/bin/ut"} {
		if _, err := os.Lstat(candidate); err == nil {
			t.Skipf("installer replacement test would touch existing %s", candidate)
		} else if !os.IsNotExist(err) {
			t.Fatalf("inspect %s: %v", candidate, err)
		}
	}

	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is required by ut_install.sh")
	}
	realTools := make(map[string]string)
	for _, tool := range []string{"mkdir", "install", "cp", "chmod", "rm"} {
		path, err := exec.LookPath(tool)
		if err != nil {
			t.Fatalf("locate %s: %v", tool, err)
		}
		realTools[tool] = path
	}

	repositoryRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(repositoryRoot, "ut_install.sh")
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("installer is unavailable: %v", err)
	}

	workRoot := t.TempDir()
	fixtureBin := filepath.Join(workRoot, "bin")
	workDir := filepath.Join(workRoot, "work")
	installDir := "-install"
	homeDir := filepath.Join(workRoot, "home")
	tmpDir := filepath.Join(workRoot, "tmp")
	for _, dir := range []string{fixtureBin, workDir, homeDir, tmpDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create %s: %v", dir, err)
		}
	}

	writeInstallerFixture(t, filepath.Join(fixtureBin, "curl"), `#!/usr/bin/env bash
set -euo pipefail
target=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o)
      target="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
if [ -n "$target" ]; then
  printf '%s\n' '#!/bin/sh' 'exit 0' >"$target"
else
  printf '%s\n' 'success'
fi
`)
	writeInstallerFixture(t, filepath.Join(fixtureBin, "wget"), "#!/bin/sh\nexit 1\n")
	writeInstallerFixture(t, filepath.Join(fixtureBin, "which"), "#!/bin/sh\nexit 0\n")
	writeInstallerFixture(t, filepath.Join(fixtureBin, "uname"), `#!/bin/sh
case "${1:-}" in
  -s) printf '%s\n' 'Darwin' ;;
  -m) printf '%s\n' 'arm64' ;;
  *) exit 1 ;;
esac
`)

	overrides := map[string]string{
		"HOME":                  homeDir,
		"INSTALL_DIR":           installDir,
		"PATH":                  fixtureBin + string(os.PathListSeparator) + os.Getenv("PATH"),
		"TMPDIR":                tmpDir,
		"UT_INSTALL_ASSUME_YES": "1",
	}
	for tool, path := range realTools {
		envName := "UT_INSTALL_TEST_" + strings.ToUpper(tool)
		overrides[envName] = path
		writeInstallerFixture(t, filepath.Join(fixtureBin, tool), "#!/bin/sh\nfor argument in \"$@\"; do\n  if [ \"$argument\" = \"--\" ]; then\n    printf '%s\\n' 'GNU end-of-options marker is not supported' >&2\n    exit 97\n  fi\ndone\nexec \"${"+envName+":?}\" \"$@\"\n")
	}

	command := exec.Command(bash, script)
	command.Dir = workDir
	command.Env = installerTestEnvironment(overrides)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("installer failed with BSD-style coreutils: %v\n%s", err, output)
	}
	for _, path := range []string{filepath.Join(workDir, installDir, "ut"), filepath.Join(workDir, "ut")} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected installed binary at %s: %v\n%s", path, err, output)
		}
		if info.Mode()&0o111 == 0 {
			t.Fatalf("installed binary at %s is not executable", path)
		}
	}
}

func writeInstallerFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}

func installerTestEnvironment(overrides map[string]string) []string {
	environment := make([]string, 0, len(os.Environ())+len(overrides))
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if _, replaced := overrides[key]; !replaced {
			environment = append(environment, item)
		}
	}
	for key, value := range overrides {
		environment = append(environment, key+"="+value)
	}
	return environment
}
