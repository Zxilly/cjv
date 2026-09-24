package cli

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/spf13/cobra"
)

type docResult struct {
	Toolchain string `json:"toolchain"`
	Topic     string `json:"topic,omitempty"`
	Path      string `json:"path"`
	Opened    bool   `json:"opened"`
}

func (r docResult) Text() string {
	if !r.Opened {
		return r.Path
	}
	return ""
}

func (app *application) runDoc(cmd *cobra.Command, args []string) error {
	topic := ""
	if len(args) > 0 {
		topic = args[0]
	}

	_, parsedName, err := resolveToolchainArg(app.docToolchain)
	if err != nil {
		return err
	}
	tcName := parsedName.String()

	roots, err := componentlib.RootsFor(tcName)
	if err != nil {
		return err
	}
	docFile, err := componentlib.ResolveDocPath(roots, topic)
	if err != nil {
		return err
	}

	// In JSON mode, never launch a browser — JSON consumers want the path,
	// not a side effect.
	if app.docPath || app.output.IsJSON() {
		return app.output.RenderTo(cmdOutput(cmd), docResult{Toolchain: tcName, Topic: topic, Path: docFile, Opened: false})
	}

	// Attempt the launch first, then report the outcome — printing "opening..."
	// before a launch that immediately fails (e.g. headless/SSH session)
	// produces contradictory "opening" + "failed" messages.
	if err := app.openURLFunc(fileURL(docFile)); err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("OpeningDocsBrowserFailed", i18n.MsgData{"Path": docFile}))
		return err
	}
	fmt.Fprintln(os.Stderr, i18n.T("OpeningDocs", nil))
	return app.output.RenderTo(cmdOutput(cmd), docResult{Toolchain: tcName, Topic: topic, Path: docFile, Opened: true})
}

// fileURL returns a file:// URL pointing at an absolute local file path. We
// do this by hand instead of pulling net/url.Parse to keep Windows drive
// letters intact ("C:\\foo" → "file:///C:/foo").
func fileURL(absPath string) string {
	clean := filepath.ToSlash(absPath)
	if len(clean) > 0 && clean[0] != '/' {
		clean = "/" + clean
	}
	return (&url.URL{Scheme: "file", Path: clean}).String()
}

// openURL launches the user's default browser to display url. It returns nil
// as soon as the launch command is started — it does not wait for the browser
// process. Returns a non-nil error when no suitable launcher is available on
// the host platform or when launching fails synchronously.
func openURL(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		// Try xdg-open first, then a couple of common fallbacks. Plain Linux
		// servers without a desktop session will fail at all of these and the
		// caller is expected to print the URL for manual opening.
		for _, launcher := range []string{"xdg-open", "wslview", "sensible-browser"} {
			if _, err := exec.LookPath(launcher); err != nil {
				continue
			}
			return exec.Command(launcher, url).Start()
		}
		return fmt.Errorf("no suitable browser launcher found")
	}
}

func (app *application) initDocCommands() {
	app.openURLFunc = openURL
	app.docCmd = &cobra.Command{
		Use:     "doc [topic]",
		Aliases: []string{"docs"},
		Short:   i18n.T("DocCmdShort", nil),
		Args:    cobra.MaximumNArgs(1),
		RunE:    app.runDoc,
	}

	app.docCmd.Flags().BoolVar(&app.docPath, "path", false, i18n.T("DocFlagPath", nil))
	app.docCmd.Flags().StringVar(&app.docToolchain, "toolchain", "", i18n.T("DocFlagToolchain", nil))
	app.rootCmd.AddCommand(app.docCmd)

}
