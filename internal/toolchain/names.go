package toolchain

import (
	"fmt"
	"strings"

	"github.com/Zxilly/cjv/internal/config"
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
	Channel Channel
	Version string // empty means "latest"
	Host    string // explicit host tuple, independent of cross-target SDK variants
	Target  string // non-empty for installed target SDK variants (e.g. linux-x64-ohos)
	Custom  string // non-empty for custom/linked toolchain names (e.g. "my-sdk")
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
		name := n.Version
		if n.Target != "" {
			name += "-" + n.Target
		} else if n.Host != "" {
			name += "-" + n.Host
		}
		return name
	}
	if n.Version == "" {
		name := n.Channel.String()
		if n.Target != "" {
			name += "-" + n.Target
		} else if n.Host != "" {
			name += "-" + n.Host
		}
		return name
	}
	name := n.Channel.String() + "-" + n.Version
	if n.Target != "" {
		name += "-" + n.Target
	} else if n.Host != "" {
		name += "-" + n.Host
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

	// Reserve scratch names on every platform, including case-insensitive
	// filesystems and Win32 names with ignored trailing dots/spaces, before
	// classifying channel versions or custom names.
	if config.IsScratchName(strings.TrimRight(strings.ToLower(input), ". ")) {
		return ToolchainName{}, fmt.Errorf("invalid toolchain name %q: reserved for installation recovery", input)
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
			if id, err := target.ParseIdentity(version); err == nil {
				if !id.IsTargetVariant() {
					return ToolchainName{Channel: ch, Host: version}, nil
				}
				return ToolchainName{Channel: ch, Target: version}, nil
			}
			version, tuple := target.SplitVariantSuffix(version)
			if tuple != "" {
				if id, err := target.ParseIdentity(tuple); err == nil && !id.IsTargetVariant() {
					return ToolchainName{Channel: ch, Version: version, Host: tuple}, nil
				}
			}
			if tuple == "" {
				for i := range version {
					if version[i] != '-' {
						continue
					}
					if id, err := target.ParseIdentity(version[i+1:]); err == nil && !id.IsTargetVariant() {
						return ToolchainName{Channel: ch, Version: version[:i], Host: version[i+1:]}, nil
					}
				}
			}
			return ToolchainName{Channel: ch, Version: version, Target: tuple}, nil
		}
	}

	// Bare version number (starts with digit)
	if len(input) > 0 && input[0] >= '0' && input[0] <= '9' {
		version, tuple := target.SplitVariantSuffix(input)
		if tuple != "" {
			id, _ := target.ParseIdentity(tuple)
			if !id.IsTargetVariant() {
				return ToolchainName{Channel: UnknownChannel, Version: version, Host: tuple}, nil
			}
			return ToolchainName{Channel: UnknownChannel, Version: version, Target: tuple}, nil
		}
		return ToolchainName{Channel: UnknownChannel, Version: input}, nil
	}

	// Custom/linked toolchain name (e.g. "my-sdk")
	return ToolchainName{Custom: input}, nil
}
