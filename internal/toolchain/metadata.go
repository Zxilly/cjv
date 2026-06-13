package toolchain

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/utils"
)

const nightlyReleaseMetadataPath = ".cjv/nightly-release.json"

// NightlyReleaseMetadata records the GitCode release identity used to install
// a nightly toolchain. ReleaseTag can differ from Version when upstream
// publishes assets whose embedded SDK version does not match the tag.
type NightlyReleaseMetadata struct {
	ReleaseTag string `json:"release_tag,omitempty"`
	Version    string `json:"version,omitempty"`
}

func WriteNightlyReleaseMetadata(tcDir string, meta NightlyReleaseMetadata) error {
	if meta.ReleaseTag == "" && meta.Version == "" {
		return nil
	}
	path := filepath.Join(tcDir, nightlyReleaseMetadataPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return utils.WriteFileAtomic(path, data, 0o644)
}

func ReadNightlyReleaseMetadata(tcDir string) (NightlyReleaseMetadata, error) {
	data, err := os.ReadFile(filepath.Join(tcDir, nightlyReleaseMetadataPath))
	if err != nil {
		return NightlyReleaseMetadata{}, err
	}
	var meta NightlyReleaseMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return NightlyReleaseMetadata{}, err
	}
	return meta, nil
}
