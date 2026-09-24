package cli

import (
	"github.com/Zxilly/cjv/internal/cli/output"
	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/spf13/cobra"
)

// application owns the command tree, flags, and renderer for one CLI execution.
type application struct {
	output                          *output.Renderer
	checkCmd                        *cobra.Command
	componentAddCmd                 *cobra.Command
	componentAddForce               bool
	componentCmd                    *cobra.Command
	componentLinkCmd                *cobra.Command
	componentLinkForce              bool
	componentLinkFunc               func(componentlib.Roots, componentlib.Name, string, bool) (string, error)
	componentListCmd                *cobra.Command
	componentListInstalledOnly      bool
	componentListQuiet              bool
	componentRemoveCmd              *cobra.Command
	componentToolchain              string
	docCmd                          *cobra.Command
	docPath                         bool
	docToolchain                    string
	execCmd                         *cobra.Command
	forceInstall                    bool
	initCmd                         *cobra.Command
	initComponents                  []string
	initDefaultToolchain            string
	initNoModifyPath                bool
	initYes                         bool
	installCmd                      *cobra.Command
	installComponents               []string
	installTargets                  []string
	linkForce                       bool
	linkNoStdx                      bool
	linkSHA256                      string
	noSelfUpdate                    bool
	openURLFunc                     func(string) error
	rootCmd                         *cobra.Command
	runCmd                          *cobra.Command
	showActiveCmd                   *cobra.Command
	showCmd                         *cobra.Command
	showHomeCmd                     *cobra.Command
	showInstalledCmd                *cobra.Command
	toolchainCmd                    *cobra.Command
	toolchainLinkCmd                *cobra.Command
	toolchainListCmd                *cobra.Command
	toolchainListRemoteAllPlatforms bool
	toolchainListRemoteChannel      string
	toolchainListRemoteCmd          *cobra.Command
	toolchainListRemoteLimit        int
	toolchainListRemoteTarget       string
	toolchainUninstallCmd           *cobra.Command
	uninstallCmd                    *cobra.Command
	uninstallYes                    bool
	updateCmd                       *cobra.Command
	updateURL                       string
	version                         string
	whichCmd                        *cobra.Command
}

func newApplication(ver, updURL string) *application {
	app := &application{version: ver, updateURL: updURL, output: &output.Renderer{}}
	app.initRootCommands()
	app.initCheckCommands()
	app.initComponentCommands()
	app.initDocCommands()
	app.initEnvsetupCommands()
	app.initExecCommands()
	app.initInitCommands()
	app.initInstallCommands()
	app.initRunCommands()
	app.initShowCommands()
	app.initToolchainCommands()
	app.initToolchainListRemoteCommands()
	app.initUninstallCommands()
	app.initUpdateCommands()
	app.initWhichCommands()
	app.configureRoot()
	return app
}
