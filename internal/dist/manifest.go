package dist

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/toolchain"
)

type DownloadInfo struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	URL    string `json:"url"`
}

type ChannelInfo struct {
	Latest   string                             `json:"latest"`
	Versions map[string]map[string]DownloadInfo `json:"versions"` // version -> platform -> info
}

type Manifest struct {
	Channels struct {
		LTS ChannelInfo `json:"lts"`
		STS ChannelInfo `json:"sts"`
	} `json:"channels"`
}

func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}
	if err := m.validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

func (m *Manifest) GetDownloadInfo(channel toolchain.Channel, version, platformKey string) (*DownloadInfo, error) {
	ch, err := m.getChannel(channel)
	if err != nil {
		return nil, err
	}

	platforms, ok := ch.Versions[version]
	if !ok {
		return nil, &cjverr.VersionNotFoundError{Version: version}
	}

	info, ok := platforms[platformKey]
	if !ok {
		return nil, &cjverr.VersionNotAvailableError{Version: version, Platform: platformKey}
	}

	return &info, nil
}

func (m *Manifest) GetLatestVersion(channel toolchain.Channel) (string, error) {
	ch, err := m.getChannel(channel)
	if err != nil {
		return "", err
	}
	if ch.Latest == "" {
		return "", fmt.Errorf("channel %s has no latest version", channel)
	}
	return ch.Latest, nil
}

// FindVersionChannel searches for a version across channels (LTS first, then STS).
func (m *Manifest) FindVersionChannel(version string) (toolchain.Channel, error) {
	if _, ok := m.Channels.LTS.Versions[version]; ok {
		return toolchain.LTS, nil
	}
	if _, ok := m.Channels.STS.Versions[version]; ok {
		return toolchain.STS, nil
	}
	return toolchain.UnknownChannel, &cjverr.VersionNotFoundError{Version: version}
}

func (m *Manifest) getChannel(ch toolchain.Channel) (*ChannelInfo, error) {
	switch ch {
	case toolchain.LTS:
		return &m.Channels.LTS, nil
	case toolchain.STS:
		return &m.Channels.STS, nil
	default:
		return nil, &cjverr.UnknownChannelError{Channel: ch.String()}
	}
}

func (m *Manifest) validate() error {
	if err := validateChannel(toolchain.LTS, m.Channels.LTS); err != nil {
		return err
	}
	if err := validateChannel(toolchain.STS, m.Channels.STS); err != nil {
		return err
	}
	return nil
}

func validateChannel(channel toolchain.Channel, ch ChannelInfo) error {
	label := channel.String()
	if ch.Latest == "" {
		return fmt.Errorf("channel %s has no latest version", label)
	}
	if len(ch.Versions) == 0 {
		return fmt.Errorf("channel %s has no versions", label)
	}
	if _, ok := ch.Versions[ch.Latest]; !ok {
		return fmt.Errorf("channel %s latest version %s is missing from versions", label, ch.Latest)
	}
	for version, platforms := range ch.Versions {
		if len(platforms) == 0 {
			return fmt.Errorf("channel %s version %s has no platforms", label, version)
		}
		for platform, info := range platforms {
			if err := validateDownloadInfo(label, version, platform, info); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateDownloadInfo(channel, version, platform string, info DownloadInfo) error {
	if info.Name == "" {
		return fmt.Errorf("channel %s version %s platform %s has empty name", channel, version, platform)
	}
	if info.URL == "" {
		return fmt.Errorf("channel %s version %s platform %s has empty url", channel, version, platform)
	}
	if len(info.SHA256) != 64 {
		return fmt.Errorf("channel %s version %s platform %s has invalid sha256 length", channel, version, platform)
	}
	if _, err := hex.DecodeString(info.SHA256); err != nil {
		return fmt.Errorf("channel %s version %s platform %s has invalid sha256: %w", channel, version, platform, err)
	}
	return nil
}
