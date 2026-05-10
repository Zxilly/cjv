package toolchain

import (
	"fmt"
	"strings"

	"github.com/Zxilly/cjv/internal/target"
)

type Channel int

const (
	UnknownChannel Channel = iota
	LTS
	STS
	Nightly
)

func (c Channel) String() string {
	switch c {
	case LTS:
		return "lts"
	case STS:
		return "sts"
	case Nightly:
		return "nightly"
	default:
		return "unknown"
	}
}

func ParseChannel(s string) (Channel, bool) {
	switch strings.ToLower(s) {
	case "lts":
		return LTS, true
	case "sts":
		return STS, true
	case "nightly":
		return Nightly, true
	default:
		return UnknownChannel, false
	}
}

// ToolchainName represents a parsed toolchain identifier.
type ToolchainName struct {
	Channel     Channel
	Version     string // empty means "latest"
	Target string // non-empty for installed target SDK variants (e.g. linux-x64-ohos)
	Custom      string // non-empty for custom/linked toolchain names (e.g. "my-sdk")
}

// IsCustom returns true if this is a custom/linked toolchain name.
func (n ToolchainName) IsCustom() bool {
	return n.Custom != ""
}

func (n ToolchainName) String() string {
	if n.Custom != "" {
		return n.Custom
	}
	if n.Channel == UnknownChannel {
		return n.Version
	}
	if n.Version == "" {
		return n.Channel.String()
	}
	name := n.Channel.String() + "-" + n.Version
	if n.Target != "" {
		name += "-" + n.Target
	}
	return name
}

func (n ToolchainName) IsChannelOnly() bool {
	return n.Custom == "" && n.Version == "" && n.Target == ""
}

// ParseToolchainName parses user input into a ToolchainName.
// Supported formats: lts, lts-1.0.5, sts-1.1.0-beta.23, nightly-xxx, or bare version 1.0.5.
func ParseToolchainName(input string) (ToolchainName, error) {
	input = strings.TrimSpace(input)
	input = strings.TrimRight(input, "/\\")
	if input == "" {
		return ToolchainName{}, fmt.Errorf("toolchain name cannot be empty")
	}
	if strings.HasPrefix(input, "+") {
		return ToolchainName{}, fmt.Errorf("invalid toolchain name '%s': do not use '+' prefix; use the name directly", input)
	}
	if strings.ContainsAny(input, "/\\") {
		return ToolchainName{}, fmt.Errorf("invalid toolchain name '%s': must not contain path separators", input)
	}
	if input == "." || input == ".." {
		return ToolchainName{}, fmt.Errorf("invalid toolchain name '%s'", input)
	}

	lowerInput := strings.ToLower(input)
	for _, ch := range []Channel{LTS, STS, Nightly} {
		prefix := ch.String()
		if lowerInput == prefix {
			return ToolchainName{Channel: ch}, nil
		}
		if strings.HasPrefix(lowerInput, prefix+"-") {
			version := input[len(prefix)+1:]
			if version == "" {
				return ToolchainName{}, fmt.Errorf("empty version in toolchain name '%s'", input)
			}
			version, tuple := target.SplitVariantSuffix(version)
			return ToolchainName{Channel: ch, Version: version, Target: tuple}, nil
		}
	}

	// Bare version number (starts with digit)
	if len(input) > 0 && input[0] >= '0' && input[0] <= '9' {
		return ToolchainName{Channel: UnknownChannel, Version: input}, nil
	}

	// Custom/linked toolchain name (e.g. "my-sdk")
	return ToolchainName{Custom: input}, nil
}
