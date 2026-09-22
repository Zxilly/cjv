package process_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func helperCommand(t *testing.T, ctx context.Context, mode string) *exec.Cmd {
	t.Helper()
	executable, err := os.Executable()
	require.NoError(t, err)
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestProcessHelper$")
	cmd.Env = append(os.Environ(), "CJV_TEST_PROCESS_HELPER="+mode)
	return cmd
}

func TestRunPreservesCommandConfiguration(t *testing.T) {
	cmd := helperCommand(t, t.Context(), "streams")
	cmd.Env = append(cmd.Env, "CJV_TEST_PROCESS_VALUE=child environment")
	cmd.Stdin = strings.NewReader("child input")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	require.NoError(t, process.Run(cmd))
	assert.Equal(t, "child input", stdout.String())
	assert.Equal(t, "child environment", stderr.String())
}

func TestRunPreservesStartError(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), filepath.Join(t.TempDir(), "missing"))
	err := process.Run(cmd)
	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist) || errors.Is(err, exec.ErrNotFound),
		"expected the operating system's missing executable error, got %v", err)
	var exitErr *cjverr.ExitCodeError
	assert.False(t, errors.As(err, &exitErr))
}

func TestRunPreservesExitCode(t *testing.T) {
	err := process.Run(helperCommand(t, t.Context(), "exit"))
	var exitErr *cjverr.ExitCodeError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 37, exitErr.Code)
}

func TestRunCancelsStartedChild(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := helperCommand(t, ctx, "wait")
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	finished := make(chan error, 1)
	go func() { finished <- process.Run(cmd) }()

	line, err := bufio.NewReader(stdout).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "ready\n", line)
	cancel()

	select {
	case err := <-finished:
		var exitErr *cjverr.ExitCodeError
		require.ErrorAs(t, err, &exitErr)
		assert.NotZero(t, exitErr.Code)
	case <-time.After(10 * time.Second):
		t.Fatal("canceled child did not exit")
	}
}

func TestProcessHelper(t *testing.T) {
	switch os.Getenv("CJV_TEST_PROCESS_HELPER") {
	case "":
		return
	case "streams":
		_, _ = io.Copy(os.Stdout, os.Stdin)
		fmt.Fprint(os.Stderr, os.Getenv("CJV_TEST_PROCESS_VALUE"))
	case "exit":
		os.Exit(37)
	case "wait":
		fmt.Fprintln(os.Stdout, "ready")
		for {
			time.Sleep(time.Hour)
		}
	default:
		t.Fatal("unknown helper mode")
	}
	os.Exit(0)
}
