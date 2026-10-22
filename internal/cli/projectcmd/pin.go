package projectcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/BurntSushi/toml"
	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/fsops"
	"github.com/Zxilly/cjv/internal/lifecycle"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

type pinResult struct {
	SchemaVersion int    `json:"schema_version"`
	Path          string `json:"path"`
	Toolchain     string `json:"toolchain"`
	Checked       bool   `json:"checked"`
}

func (r pinResult) Text() string {
	if r.Checked {
		return fmt.Sprintf("%s pins %s", r.Path, r.Toolchain)
	}
	return fmt.Sprintf("Pinned %s in %s", r.Toolchain, r.Path)
}

func newPinCommand(shared *clictx.Context) *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use: "pin [toolchain]", Short: "Pin a concrete SDK release in the current project",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if check && (len(args) != 0 || shared.Selector != "") {
				return fmt.Errorf("pin --check validates the current project file and does not accept a toolchain selector")
			}
			path, err := filepath.Abs(config.ToolchainFileName)
			if err != nil {
				return err
			}
			project, document, err := readPinProject(path)
			if err != nil && (!errors.Is(err, os.ErrNotExist) || check) {
				return err
			}
			if check {
				if _, err := concretePin(project.Toolchain.Channel); err != nil || project.Toolchain.Path != "" {
					return fmt.Errorf("%s does not pin a concrete published release; run cjv pin <toolchain>", path)
				}
				return shared.Output.RenderTo(cmd.OutOrStdout(), pinResult{1, path, project.Toolchain.Channel, true})
			}
			selector := shared.Selector
			if len(args) > 0 {
				selector = args[0]
			}
			if selector == "" {
				_, settings, settingsErr := config.LoadDefaultSettings()
				selected, err := toolchain.SelectActive(settings, settingsErr, "")
				if err != nil {
					return err
				}
				selector = selected.Name
			}
			release, err := resolvePin(cmd.Context(), selector, shared)
			if err != nil {
				return err
			}
			section, ok := document["toolchain"].(map[string]any)
			if !ok {
				section = map[string]any{}
				document["toolchain"] = section
			}
			section["channel"] = release
			delete(section, "path")
			var buf bytes.Buffer
			if err := toml.NewEncoder(&buf).Encode(document); err != nil {
				return err
			}
			if err := fsops.WriteFileAtomic(path, buf.Bytes(), 0o644); err != nil {
				return err
			}
			return shared.Output.RenderTo(cmd.OutOrStdout(), pinResult{1, path, release, false})
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "Check that the current project pins a concrete release without writing or accessing the network")
	return cmd
}

func readPinProject(path string) (*config.ToolchainFileContent, map[string]any, error) {
	document := map[string]any{}
	project, err := config.ParseToolchainFile(path)
	if err != nil {
		return nil, document, err
	}
	if _, err := component.NormalizeList(project.Toolchain.Components); err != nil {
		return nil, nil, err
	}
	if _, err := sdktarget.NormalizeList(project.Toolchain.Targets); err != nil {
		return nil, nil, err
	}
	_, err = toml.DecodeFile(path, &document)
	return project, document, err
}

var concreteVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[.+-][0-9A-Za-z.-]+)?$`)

func concretePin(raw string) (toolchain.ToolchainName, error) {
	name, err := toolchain.ParseActiveName(raw)
	if err != nil {
		return name, err
	}
	if name.IsCustom() || name.Channel == toolchain.UnknownChannel || !concreteVersion.MatchString(name.Version) || toolchain.IsVersionSelector(name.Version) {
		return name, fmt.Errorf("%q is not a concrete published release", raw)
	}
	return name, nil
}

func resolvePin(ctx context.Context, raw string, shared *clictx.Context) (string, error) {
	if filepath.IsAbs(raw) {
		return "", fmt.Errorf("cannot pin a path SDK as a published release; specify an official toolchain, for example cjv pin lts")
	}
	name, err := toolchain.ParseActiveName(raw)
	if err != nil {
		return "", err
	}
	if name.IsCustom() {
		return "", fmt.Errorf("cannot pin custom SDK %q as a published release; specify an official toolchain, for example cjv pin lts", raw)
	}
	if dir, err := toolchain.FindInstalled(name); err == nil {
		release, err := toolchain.InstalledRelease(dir)
		if err != nil {
			return "", err
		}
		if _, err := concretePin(release.String()); err != nil {
			return "", err
		}
		return release.String(), nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	distribution, err := lifecycle.OpenDistribution(shared.LifecycleOptions())
	if err != nil {
		return "", err
	}
	release, err := distribution.Resolve(ctx, name, "")
	if err != nil {
		return "", err
	}
	if _, err := concretePin(release.Name); err != nil {
		return "", err
	}
	return release.Name, nil
}
