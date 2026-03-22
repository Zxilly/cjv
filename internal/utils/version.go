package utils

// AppVersion is set at build time via ldflags.
var AppVersion = "dev"

// Version returns the application version string.
func Version() string {
	return AppVersion
}
