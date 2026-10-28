package lifecycle

import (
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// groupedTrackingTargets derives ownership from one observed installation list.
// Both planning and updating omit targets already covered by a tracking host.
// Legacy records without Tuple use the same explicit/default host fallback.
func groupedTrackingTargets(names []string, records map[string]toolchain.Installation, defaultHost string) map[string]bool {
	hosts := make(map[string]bool)
	for _, identity := range names {
		name, err := toolchain.ParseToolchainName(identity)
		if err != nil || name.IsCustom() || name.Version != "" || name.Target != "" {
			continue
		}
		record, ok := records[identity]
		if !ok {
			continue
		}
		tuple := record.Tuple
		if tuple == "" {
			tuple = name.Host
		}
		if tuple == "" {
			tuple = defaultHost
		}
		hosts[name.Channel.String()+"-"+tuple] = true
	}
	grouped := make(map[string]bool)
	for _, identity := range names {
		name, err := toolchain.ParseToolchainName(identity)
		if err != nil || name.IsCustom() || name.Version != "" || name.Target == "" {
			continue
		}
		parts, err := sdktarget.ParseTuple(name.Target)
		if err == nil && hosts[name.Channel.String()+"-"+parts.Host] {
			grouped[identity] = true
		}
	}
	return grouped
}
