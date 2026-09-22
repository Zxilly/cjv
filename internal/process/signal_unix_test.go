//go:build !windows

package process_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func signalCommand(t *testing.T, ctx context.Context, mode string) *exec.Cmd {
	t.Helper()
	executable, err := os.Executable()
	require.NoError(t, err)
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestSignalHelper$")
	cmd.Env = append(os.Environ(), "CJV_TEST_SIGNAL_HELPER="+mode)
	return cmd
}

func TestRunForwardsTermination(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := signalCommand(t, ctx, "supervisor-term")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) })

	line, err := bufio.NewReader(stdout).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "ready\n", line)
	// Signal only the supervisor. The child must receive the forwarded signal
	// and its exit status must survive both process.Run and the supervisor.
	require.NoError(t, cmd.Process.Signal(syscall.SIGTERM))
	var exitErr *exec.ExitError
	require.ErrorAs(t, cmd.Wait(), &exitErr)
	assert.Equal(t, 42, exitErr.ExitCode())
}

func TestRunDoesNotForwardInterrupt(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := signalCommand(t, ctx, "supervisor-interrupt")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) })

	line, err := bufio.NewReader(stdout).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "ready\n", line)
	// Ctrl+C already reaches the child from the terminal. Sending SIGINT to
	// cjv alone must not deliver another interrupt to that child.
	require.NoError(t, cmd.Process.Signal(os.Interrupt))
	_, err = fmt.Fprintln(stdin, "finish")
	require.NoError(t, err)
	require.NoError(t, cmd.Wait())
}

func TestRunEscalatesTermination(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	cmd := signalCommand(t, ctx, "supervisor-ignore")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) })

	line, err := bufio.NewReader(stdout).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "ready\n", line)
	started := time.Now()
	require.NoError(t, cmd.Process.Signal(syscall.SIGTERM))
	var exitErr *exec.ExitError
	require.ErrorAs(t, cmd.Wait(), &exitErr)
	// The supervisor maps the killed child's ExitCodeError(-1) through
	// os.Exit, yielding 255. Its context must not be what killed it.
	assert.Equal(t, 255, exitErr.ExitCode())
	assert.NoError(t, ctx.Err())
	assert.GreaterOrEqual(t, time.Since(started), 10*time.Second)
}

func TestSignalHelper(t *testing.T) {
	mode := os.Getenv("CJV_TEST_SIGNAL_HELPER")
	if mode == "" {
		return
	}
	if childMode, supervisor := strings.CutPrefix(mode, "supervisor-"); supervisor {
		cmd := signalCommand(t, t.Context(), childMode)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := process.Run(cmd); err != nil {
			var exitErr *cjverr.ExitCodeError
			if errors.As(err, &exitErr) {
				os.Exit(exitErr.Code)
			}
			t.Fatal(err)
		}
		os.Exit(0)
	}

	if mode == "ignore" {
		signal.Ignore(syscall.SIGTERM)
		fmt.Fprintln(os.Stdout, "ready")
		for {
			time.Sleep(time.Hour)
		}
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, os.Interrupt)
	fmt.Fprintln(os.Stdout, "ready")
	switch mode {
	case "term":
		if <-signals == syscall.SIGTERM {
			os.Exit(42)
		}
		os.Exit(43)
	case "interrupt":
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		select {
		case <-signals:
			os.Exit(44)
		case <-time.After(100 * time.Millisecond):
			os.Exit(0)
		}
	default:
		t.Fatal("unknown signal helper mode")
	}
}
