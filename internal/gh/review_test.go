package gh

import (
	"testing"

	"github.com/milis92/git-simple-flow/internal/runner"
)

func TestGetPRChecksAllowsPendingExitCodeWithJSONOutput(t *testing.T) {
	installFakeGH(t, `#!/bin/sh
if [ "$1" = "pr" ] && [ "$2" = "checks" ]; then
  echo '[{"name":"build","state":"PENDING","bucket":"pending"}]'
  exit 8
fi
echo "unexpected gh command: $*" >&2
exit 1
`)

	client := New(runner.NewRunner(false, false))
	checks, err := client.GetPRChecks()
	if err != nil {
		t.Fatalf("GetPRChecks() error = %v, want pending checks parsed from gh exit code 8", err)
	}
	if len(checks) != 1 {
		t.Fatalf("len(checks) = %d, want 1", len(checks))
	}
	if checks[0].Name != "build" || checks[0].Bucket != "pending" {
		t.Fatalf("checks[0] = %+v, want pending build check", checks[0])
	}
}
