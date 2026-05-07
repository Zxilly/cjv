//go:build mirror

package config

// DefaultManifestURL is the SDK manifest URL the binary ships with.
// The mirror build pulls from GitCode so it works in environments without
// reliable GitHub access.
const DefaultManifestURL = "https://raw.gitcode.com/Zxilly/cangjie-version-manifest/raw/master/versions.json"
