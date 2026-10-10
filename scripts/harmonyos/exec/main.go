// cjv-hmos-exec implements go test -exec through an HDC-connected emulator.
package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }

// HDC shell can return success even when the guest program fails. Only a final,
// per-invocation marker establishes the guest exit status.
func result(output []byte, marker string) ([]byte, int, error) {
	output = bytes.ReplaceAll(output, []byte("\r\n"), []byte("\n"))
	i := bytes.LastIndex(output, []byte("\n"+marker))
	if i < 0 {
		return output, 1, fmt.Errorf("HDC output is missing the guest exit status")
	}
	status := strings.TrimSuffix(string(output[i+1+len(marker):]), "\n")
	code, err := strconv.Atoi(status)
	if err != nil || code < 0 || code > 255 {
		return output, 1, fmt.Errorf("invalid guest exit status %q", status)
	}
	return output[:i], code, nil
}

func guestDir(source, cwd, root string) (string, error) {
	rel, err := filepath.Rel(source, cwd)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("working directory must be inside HMOS_TEST_SOURCE")
	}
	return path.Join(root, "src", filepath.ToSlash(rel)), nil
}

func run(args []string) (int, error) {
	if len(args) == 0 {
		return 1, fmt.Errorf("usage: cjv-hmos-exec binary [args...]")
	}
	hdc, target := os.Getenv("HDC"), os.Getenv("HDC_TARGET")
	root, source := os.Getenv("HMOS_TEST_ROOT"), os.Getenv("HMOS_TEST_SOURCE")
	if hdc == "" || target == "" || source == "" || !strings.HasPrefix(root, "/data/local/tmp/") || path.Clean(root) != root {
		return 1, fmt.Errorf("set HDC, HDC_TARGET, HMOS_TEST_SOURCE and HMOS_TEST_ROOT (below /data/local/tmp)")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return 1, err
	}
	dir, err := guestDir(source, cwd, root)
	if err != nil {
		return 1, err
	}
	id := rand.Text()
	runDir := path.Join(root, "runs", id)
	marker := "CJV_EXIT_" + id + "="
	call := func(args ...string) ([]byte, error) {
		return exec.Command(hdc, append([]string{"-t", target}, args...)...).CombinedOutput()
	}
	checked := func(command string) error {
		out, err := call("shell", "( "+command+" ); code=$?; printf '\\n"+marker+"%s\\n' \"$code\"")
		if err != nil {
			return fmt.Errorf("HDC: %w: %s", err, out)
		}
		body, code, err := result(out, marker)
		if err != nil {
			return err
		}
		if code != 0 {
			return fmt.Errorf("guest setup exited %d: %s", code, body)
		}
		return nil
	}
	if err := checked("mkdir -p " + quote(runDir+"/home") + " " + quote(runDir+"/tmp")); err != nil {
		return 1, err
	}
	defer func() {
		if err := checked("rm -rf " + quote(runDir)); err != nil {
			fmt.Fprintln(os.Stderr, "cleanup:", err)
		}
	}()
	binary := args[0]
	if signer := os.Getenv("HMOS_SIGN"); signer != "" {
		// Sign a private copy: go test may reuse its cached executable.
		data, err := os.ReadFile(binary)
		if err != nil {
			return 1, err
		}
		tmp, err := os.MkdirTemp("", "cjv-hmos-exec-")
		if err != nil {
			return 1, err
		}
		defer func() { _ = os.RemoveAll(tmp) }()
		binary = filepath.Join(tmp, "test")
		if err := os.WriteFile(binary, data, 0700); err != nil {
			return 1, err
		}
		if out, err := exec.Command(signer, binary, binary).CombinedOutput(); err != nil {
			return 1, fmt.Errorf("sign: %w: %s", err, out)
		}
	}
	guestBinary := runDir + "/test"
	if out, err := call("file", "send", binary, guestBinary); err != nil {
		return 1, fmt.Errorf("send: %w: %s", err, out)
	}
	command := "chmod 700 " + quote(guestBinary) + " && cd " + quote(dir) +
		" && HOME=" + quote(runDir+"/home") + " TMPDIR=" + quote(runDir+"/tmp") + " GOMAXPROCS=4 " + quote(guestBinary)
	for _, arg := range args[1:] {
		command += " " + quote(arg)
	}
	out, transportErr := call("shell", "( "+command+" ); code=$?; printf '\\n"+marker+"%s\\n' \"$code\"")
	body, code, err := result(out, marker)
	_, _ = os.Stdout.Write(body)
	if transportErr != nil {
		return 1, fmt.Errorf("HDC transport: %w", transportErr)
	}
	return code, err
}

func main() {
	code, err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
	}
	os.Exit(code)
}
