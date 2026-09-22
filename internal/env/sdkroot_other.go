//go:build !darwin

package env

func applyPlatformVars(_ map[string]string, _ []string) {}
