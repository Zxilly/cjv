package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/selfmgmt"
	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/reachable"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	glowutils "github.com/charmbracelet/glow/v2/utils"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

const (
	menuProceed   = "proceed"
	menuCustomize = "customize"
	menuCancel    = "cancel"
)

func yesNoStr(b bool) string {
	if b {
		return i18n.T("Yes", nil)
	}
	return i18n.T("No", nil)
}

func initComponentsStr(values []string) string {
	components, err := componentlib.NormalizeList(values)
	if err != nil {
		return strings.Join(values, ", ")
	}
	if len(components) == 0 {
		return i18n.T("InitComponentsNone", nil)
	}
	parts := make([]string, 0, len(components))
	for _, c := range components {
		parts = append(parts, string(c))
	}
	return strings.Join(parts, ", ")
}

func initComponentOptions() []huh.Option[string] {
	known := componentlib.KnownComponents()
	options := make([]huh.Option[string], 0, len(known))
	for _, c := range known {
		name := string(c)
		options = append(options, huh.NewOption(name, name))
	}
	return options
}

func renderInitMarkdown(markdown string) (string, error) {
	style := styles.AutoStyle
	if !initStdoutIsTerminal() {
		style = styles.NoTTYStyle
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithColorProfile(lipgloss.ColorProfile()),
		glowutils.GlamourStyle(style, false),
		glamour.WithWordWrap(100),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return "", err
	}
	return r.Render(markdown)
}

func printInitMarkdown(markdown string) {
	rendered, err := renderInitMarkdown(markdown)
	if err != nil {
		fmt.Println(markdown)
		return
	}
	fmt.Print(rendered)
	if !strings.HasSuffix(rendered, "\n") {
		fmt.Println()
	}
}

func initStdoutIsTerminal() bool {
	fd := os.Stdout.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

func initStdinIsTerminal() bool {
	fd := os.Stdin.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

func normalizeInitHomePath(input string) (string, error) {
	path := strings.TrimSpace(input)
	if path == "" {
		return "", errors.New(i18n.T("InitHomePathEmpty", nil))
	}
	if strings.ContainsRune(path, 0) {
		return "", errors.New(i18n.T("InitHomePathInvalid", i18n.MsgData{"Path": path}))
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T("InitHomePathInvalid", i18n.MsgData{"Path": path}), err)
	}
	abs = filepath.Clean(abs)
	if info, err := os.Stat(abs); err == nil {
		if !info.IsDir() {
			return "", errors.New(i18n.T("InitHomePathNotDir", i18n.MsgData{"Path": abs}))
		}
		return abs, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("%s: %w", i18n.T("InitHomePathInvalid", i18n.MsgData{"Path": abs}), err)
	}
	return abs, nil
}

func ensureInitHomePath(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return errors.New(i18n.T("InitHomePathCreateFailed", i18n.MsgData{
			"Path": path,
			"Err":  err.Error(),
		}))
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("InitHomePathInvalid", i18n.MsgData{"Path": path}), err)
	}
	if !info.IsDir() {
		return errors.New(i18n.T("InitHomePathNotDir", i18n.MsgData{"Path": path}))
	}
	return nil
}

func activateInitHomePath(path string) error {
	if err := ensureInitHomePath(path); err != nil {
		return err
	}
	sf, err := config.DefaultSettingsFile()
	if err != nil {
		return err
	}
	if _, err := sf.Update(config.SettingsUpdate{Home: &path}); err != nil {
		return err
	}
	return os.Setenv(config.EnvHome, path)
}

type initCustomizeOptions struct {
	home       string
	toolchain  string
	components []string
	modifyPath bool
}

func newInitCustomizeForm(opts *initCustomizeOptions) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(i18n.T("InitInstallPathQuestion", nil)).
				Description(i18n.T("InitInstallPathDescription", nil)).
				Value(&opts.home).
				Validate(func(value string) error {
					_, err := normalizeInitHomePath(value)
					return err
				}),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(i18n.T("InitToolchainQuestion", nil)).
				Options(
					huh.NewOption("lts", "lts"),
					huh.NewOption("sts", "sts"),
					huh.NewOption("nightly", "nightly"),
					huh.NewOption(i18n.T("InitToolchainNone", nil), "none"),
				).
				Value(&opts.toolchain),
		),
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(i18n.T("InitComponentsQuestion", nil)).
				Options(initComponentOptions()...).
				Value(&opts.components),
		).WithHideFunc(func() bool {
			if opts.toolchain == "none" {
				opts.components = nil
				return true
			}
			return false
		}),
		huh.NewGroup(
			huh.NewConfirm().
				Title(i18n.T("InitModifyPathQuestion", nil)).
				Value(&opts.modifyPath),
		),
	)
}

func runInitCustomizePrompt(opts *initCustomizeOptions) error {
	if err := newInitCustomizeForm(opts).Run(); err != nil {
		return err
	}
	normalizedHome, err := normalizeInitHomePath(opts.home)
	if err != nil {
		return err
	}
	opts.home = normalizedHome
	if opts.toolchain == "none" {
		opts.components = nil
	}
	return nil
}

func (app *application) runInit(cmd *cobra.Command, _ []string) error {
	if app.output.IsJSON() {
		return &cjverr.UnsupportedForJSONError{Command: "init"}
	}
	selfmgmt.CheckSudoSafety()

	home, err := config.Home()
	if err != nil {
		return err
	}
	initialHome := home
	binDir, err := config.BinDir()
	if err != nil {
		return err
	}

	// Effective options — initialized from CLI flags, may be modified by interactive menu
	toolchain := app.initDefaultToolchain
	components := append([]string(nil), app.initComponents...)
	modifyPath := !app.initNoModifyPath

	fmt.Println()
	color.Cyan(i18n.T("InitWelcome", nil))
	fmt.Println()
	printInitMarkdown(i18n.T("InitDescription", nil))

	fmt.Println(i18n.T("InitDataDir", nil))
	fmt.Println()
	fmt.Printf("    %s\n", home)
	fmt.Println()
	printInitMarkdown(i18n.T("InitDataDirEnvHint", nil))

	printInitMarkdown(i18n.T("InitCommandsAvailable", nil))
	fmt.Printf("    %s\n", binDir)
	fmt.Println()

	if !modifyPath {
		fmt.Println(i18n.T("InitPathNeedManual", nil))
	} else if runtime.GOOS == "windows" {
		fmt.Println(i18n.T("InitRegistryPath", nil))
	} else {
		fmt.Println(i18n.T("InitShellConfigs", nil))
		fmt.Println()
		posix, fish := reachable.ShellConfigPaths()
		for _, rc := range posix {
			fmt.Printf("    %s\n", rc)
		}
		if fish != "" {
			fmt.Printf("    %s\n", fish)
		}
	}
	fmt.Println()

	printInitMarkdown(i18n.T("InitUninstallHint", nil))

	// The interactive menu relies on huh forms reading from a terminal. When
	// stdin is not a TTY (the documented `curl ... | sh` / `irm ... | iex`
	// bootstrap pipes the script through stdin) we cannot prompt, so fall back
	// to a non-interactive standard install with the default options instead of
	// failing with an opaque form error.
	interactive := !app.initYes && initStdinIsTerminal()
	if !app.initYes && !interactive {
		fmt.Println()
		fmt.Println(i18n.T("InitNonInteractive", nil))
	}

	if interactive {
		customized := false
	menuLoop:
		for {
			fmt.Println()
			fmt.Println(i18n.T("InitCurrentOptions", nil))
			fmt.Println()
			fmt.Printf("   %s %s\n", i18n.T("InitOptInstallPath", nil), home)
			fmt.Printf("   %s %s\n", i18n.T("InitOptToolchain", nil), toolchain)
			fmt.Printf("   %s %s\n", i18n.T("InitOptComponents", nil), initComponentsStr(components))
			fmt.Printf("   %s %s\n", i18n.T("InitOptModifyPath", nil), yesNoStr(modifyPath))
			fmt.Println()

			proceedLabel := i18n.T("InitProceedStandard", nil)
			if customized {
				proceedLabel = i18n.T("InitProceedSelected", nil)
			}

			var choice string
			if err := huh.NewSelect[string]().
				Options(
					huh.NewOption(proceedLabel, menuProceed),
					huh.NewOption(i18n.T("InitCustomize", nil), menuCustomize),
					huh.NewOption(i18n.T("InitCancelInstall", nil), menuCancel),
				).
				Value(&choice).
				Run(); err != nil {
				return err
			}

			switch choice {
			case menuCancel:
				return nil
			case menuProceed:
				break menuLoop
			case menuCustomize:
				customized = true

				opts := initCustomizeOptions{
					home:       home,
					toolchain:  toolchain,
					components: components,
					modifyPath: modifyPath,
				}
				if err := runInitCustomizePrompt(&opts); err != nil {
					return err
				}
				home = opts.home
				binDir = filepath.Join(home, "bin")
				toolchain = opts.toolchain
				components = opts.components
				modifyPath = opts.modifyPath
			}
		}
	}

	if toolchain == "none" && len(components) > 0 {
		return errors.New(i18n.T("InitComponentsRequireToolchain", nil))
	}

	managedPath := filepath.Join(binDir, sdktools.CjvBinaryName())
	if _, err := os.Stat(managedPath); err == nil {
		fmt.Println(i18n.T("InitAlreadyInstalled", i18n.MsgData{"Path": managedPath}))

		if interactive {
			confirm := false
			if err := huh.NewConfirm().
				Title(i18n.T("InitReinstallConfirm", nil)).
				Value(&confirm).
				Run(); err != nil {
				return err
			}
			if !confirm {
				return nil
			}
		}
	}

	return app.installInit(cmd.Context(), initialHome, initCustomizeOptions{
		home:       home,
		toolchain:  toolchain,
		components: components,
		modifyPath: modifyPath,
	})
}

// installInit applies the selected options after the interactive menu and
// reinstall confirmation. It is shared by interactive and unattended init.
func (app *application) installInit(ctx context.Context, initialHome string, opts initCustomizeOptions) (retErr error) {
	home := opts.home
	binDir := filepath.Join(home, "bin")
	if home != initialHome {
		previousHome, hadHome := os.LookupEnv(config.EnvHome)
		defer func() {
			var err error
			if hadHome {
				err = os.Setenv(config.EnvHome, previousHome)
			} else {
				err = os.Unsetenv(config.EnvHome)
			}
			if err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("restore %s after init: %w", config.EnvHome, err))
			}
		}()
		// The selected home stays persisted, but its environment override is
		// needed only while this installation targets that directory.
		if err := activateInitHomePath(home); err != nil {
			return err
		}
		var err error
		home, err = config.Home()
		if err != nil {
			return err
		}
		binDir, err = config.BinDir()
		if err != nil {
			return err
		}
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if err := config.EnsureDirs(); err != nil {
		return err
	}
	// A (re)installation always leaves the running cjv behind as the managed
	// binary and refreshes the env scripts; PATH follows the user's choice.
	if err := reachable.Ensure(reachable.Policy{
		ForceManagedBinary: true,
		EnvScripts:         true,
		ConfigurePath:      opts.modifyPath,
	}); err != nil {
		return err
	}

	if opts.toolchain != "none" {
		// Init has already handled PATH; the toolchain install must not touch
		// it again. The policy travels with this invocation.
		installOpts := app.lifecycleOptions()
		installOpts.ConfigurePath = false
		if err := lifecycle.Install(ctx, lifecycle.InstallRequest{Toolchain: opts.toolchain, Components: opts.components}, installOpts); err != nil {
			fmt.Fprintf(os.Stderr, "\n%s\n", i18n.T("InitToolchainFailed", i18n.MsgData{
				"Name": opts.toolchain,
				"Err":  err.Error(),
			}))
		}
	}

	fmt.Println()
	color.Green(i18n.T("InitComplete", nil))
	fmt.Println()
	printInitMarkdown(i18n.T("InitSourceHint", nil))
	printInitMarkdown(i18n.T("InitSourceHintRun", nil))
	if runtime.GOOS != "windows" {
		envPath := filepath.Join(home, "env")
		fmt.Printf("    source \"%s\"\n", envPath)
	} else {
		ps1Path := filepath.Join(home, "env.ps1")
		batPath := filepath.Join(home, "env.bat")
		fmt.Printf("    PowerShell: . \"%s\"\n", ps1Path)
		fmt.Printf("    CMD:        \"%s\"\n", batPath)
	}
	fmt.Println()

	if opts.toolchain == "none" {
		printInitMarkdown(i18n.T("InitInstallHint", nil))
		fmt.Println("    cjv install <toolchain>")
		fmt.Println()
	}
	if !opts.modifyPath {
		fmt.Println(i18n.T("InitNoModifyPath", i18n.MsgData{"BinDir": binDir}))
	}

	return nil
}

func (app *application) initInitCommands() {
	app.initCmd = &cobra.Command{
		Use:   "init",
		Short: i18n.T("InitCmdShort", nil),
		Long:  i18n.T("InitCmdLong", nil),
		RunE:  app.runInit,
	}

	app.initCmd.Flags().BoolVarP(&app.initYes, "yes", "y", false, i18n.T("FlagSkipConfirm", nil))
	app.initCmd.Flags().StringVar(&app.initDefaultToolchain, "default-toolchain", "lts", i18n.T("InitFlagDefaultToolchain", nil))
	app.initCmd.Flags().StringSliceVarP(&app.initComponents, "component", "c", nil, i18n.T("InstallFlagComponent", nil))
	app.initCmd.Flags().BoolVar(&app.initNoModifyPath, "no-modify-path", false, i18n.T("InitFlagNoModifyPath", nil))
	app.rootCmd.AddCommand(app.initCmd)

}
