//go:build !mirror

package config

// DefaultManifestURL is the SDK manifest URL the binary ships with.
// The default build pulls from GitHub.
const DefaultManifestURL = "https://raw.githubusercontent.com/Zxilly/cangjie-version-manifest/refs/heads/master/versions.json"
