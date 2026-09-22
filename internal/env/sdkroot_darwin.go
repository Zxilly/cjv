//go:build darwin

package env

import (
	"os/exec"
	"strings"
	"sync"
)

var (
	darwinSDKRootOnce  sync.Once
	darwinSDKRootValue string
	darwinSDKRootErr   error
)

func queryDarwinSDKRoot() (string, error) {
	darwinSDKRootOnce.Do(func() {
		out, err := exec.Command("xcrun", "--sdk", "macosx", "--show-sdk-path").Output()
		if err != nil {
			darwinSDKRootErr = err
			return
		}
		darwinSDKRootValue = strings.TrimSpace(string(out))
	})
	return darwinSDKRootValue, darwinSDKRootErr
}

func applyPlatformVars(vars map[string]string, baseEnv []string) {
	current, _ := LookupValue(baseEnv, "SDKROOT")
	applyDarwinSDKRoot(vars, current, queryDarwinSDKRoot)
}
