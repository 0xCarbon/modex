package diagnostics

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
)

// runTool executes a command in dir, capturing combined stdout+stderr.
// An exit error (non-zero status) is silently swallowed because diagnostic
// tools return exit code 1 when they find issues, which is the normal case.
// Other errors (command not found, permission denied) are returned as-is.
func runTool(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Non-zero exit means the tool ran and found issues; that's expected.
			return buf.String(), nil
		}
		return "", err
	}
	return buf.String(), nil
}
