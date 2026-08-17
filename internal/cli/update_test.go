package cli

import (
	"strings"
	"testing"
)

func TestUpdateCommandRefusesToSelfUpdate(t *testing.T) {
	for _, args := range [][]string{
		{"update"},
		{"update", "--beta"},
		{"update", "-y"},
		{"update", "--force"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Setenv("NM_HOME", t.TempDir())

			out, err := executeCmd(args...)
			if err == nil {
				t.Fatalf("%v should refuse, got output: %s", args, out)
			}
			message := out + err.Error()
			if !strings.Contains(message, "self-update is disabled in this build") {
				t.Fatalf("%v should explain the refusal, got output %q error %v", args, out, err)
			}
			if strings.Contains(message, "self-update unavailable for development builds") {
				t.Fatalf("%v must not reach the updater, got output %q error %v", args, out, err)
			}
		})
	}
}
