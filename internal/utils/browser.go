package utils

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenURL launches the user's default browser to display url. The function
// returns nil as soon as the launch command is started — it does not wait
// for the browser process. Returns a non-nil error when no suitable launcher
// is available on the host platform or when launching fails synchronously.
func OpenURL(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		// Try xdg-open first, then a couple of common fallbacks. Plain Linux
		// servers without a desktop session will fail at all of these and the
		// caller is expected to print the URL for manual opening.
		for _, launcher := range []string{"xdg-open", "wslview", "sensible-browser"} {
			if _, err := exec.LookPath(launcher); err != nil {
				continue
			}
			return exec.Command(launcher, url).Start()
		}
		return fmt.Errorf("no suitable browser launcher found")
	}
}
