package selfupdate

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zxilly/cjv/internal/dist"
	goversion "github.com/hashicorp/go-version"

	"github.com/Zxilly/cjv/internal/progress"
)

type releaseArtifact struct {
	AssetName   string
	BinaryName  string
	AssetURL    string
	ChecksumURL string
	// Progress receives the asset download progress; nil reports nothing.
	Progress progress.Sink
}

func newerReleaseVersion(currentVersion, tag string) (string, bool, error) {
	latest := strings.TrimPrefix(strings.TrimSpace(tag), "v")
	latestVersion, err := goversion.NewVersion(latest)
	if err != nil {
		return "", false, fmt.Errorf("invalid release version %q: %w", tag, err)
	}
	current, err := goversion.NewVersion(currentVersion)
	if err != nil {
		return "", false, fmt.Errorf("invalid current version %q: %w", currentVersion, err)
	}
	return latest, latestVersion.GreaterThan(current), nil
}

func releaseAssetName(binary, goos, goarch string) string {
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("%s_%s_%s%s", binary, goos, goarch, ext)
}

func platformBinaryName(binary, goos string) string {
	if goos == "windows" {
		return binary + ".exe"
	}
	return binary
}

func checksumForAsset(data []byte, assetName string) (string, error) {
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.TrimPrefix(fields[1], "*") != assetName {
			continue
		}
		digest := strings.ToLower(fields[0])
		if len(digest) != sha256HexLength {
			return "", fmt.Errorf("invalid SHA-256 for %s", assetName)
		}
		if _, err := hex.DecodeString(digest); err != nil {
			return "", fmt.Errorf("invalid SHA-256 for %s: %w", assetName, err)
		}
		return digest, nil
	}
	return "", fmt.Errorf("checksums.txt has no entry for %s", assetName)
}

const sha256HexLength = 64

func fetchReleaseFile(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := dist.HTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort cleanup
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: HTTP %d", rawURL, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, dist.MaxResponseSize))
}

func installReleaseArtifact(ctx context.Context, artifact releaseArtifact) error {
	checksums, err := fetchReleaseFile(ctx, artifact.ChecksumURL)
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}
	expected, err := checksumForAsset(checksums, artifact.AssetName)
	if err != nil {
		return err
	}

	managedExe, err := ManagedExecutablePath()
	if err != nil {
		return err
	}
	tmpDir, err := os.MkdirTemp("", "cjv-self-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck // best-effort cleanup

	archivePath := filepath.Join(tmpDir, filepath.Base(artifact.AssetName))
	if err := dist.DownloadFile(ctx, artifact.AssetURL, archivePath, expected, artifact.Progress); err != nil {
		return fmt.Errorf("download release asset: %w", err)
	}
	extractDir := filepath.Join(tmpDir, "extract")
	if _, err := dist.ExtractFlattened(ctx, archivePath, extractDir, true); err != nil {
		return fmt.Errorf("extract release asset: %w", err)
	}
	binaryPath := filepath.Join(extractDir, filepath.FromSlash(artifact.BinaryName))
	info, err := os.Stat(binaryPath)
	if err != nil {
		return fmt.Errorf("release binary %s: %w", artifact.BinaryName, err)
	}
	if info.IsDir() {
		return fmt.Errorf("release binary %s is a directory", artifact.BinaryName)
	}

	binary, err := os.Open(binaryPath)
	if err != nil {
		return err
	}
	defer binary.Close() //nolint:errcheck // read-only
	return applyExecutableUpdate(binary, managedExe, 0)
}
