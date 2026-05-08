package cli

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/cli/output"
	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/utils"
	"github.com/spf13/cobra"
)

// openURLFunc is overridable in tests to suppress real browser launches.
var openURLFunc = utils.OpenURL

var (
	docPath      bool
	docToolchain string
)

var docCmd = &cobra.Command{
	Use:     "doc [topic]",
	Aliases: []string{"docs"},
	Short:   i18n.T("DocCmdShort", nil),
	Args:    cobra.MaximumNArgs(1),
	RunE:    runDoc,
}

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

func init() {
	docCmd.Flags().BoolVar(&docPath, "path", false, i18n.T("DocFlagPath", nil))
	docCmd.Flags().StringVar(&docToolchain, "toolchain", "", i18n.T("DocFlagToolchain", nil))
	rootCmd.AddCommand(docCmd)
}

func runDoc(cmd *cobra.Command, args []string) error {
	topic := ""
	if len(args) > 0 {
		topic = args[0]
	}

	_, parsedName, err := resolveToolchainArg(docToolchain)
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
	if docPath || output.IsJSON() {
		return output.RenderTo(cmdOutput(cmd), docResult{Toolchain: tcName, Topic: topic, Path: docFile, Opened: false})
	}

	fmt.Fprintln(os.Stderr, i18n.T("OpeningDocs", nil))
	if err := openURLFunc(fileURL(docFile)); err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("OpeningDocsBrowserFailed", i18n.MsgData{"Path": docFile}))
		return err
	}
	return output.RenderTo(cmdOutput(cmd), docResult{Toolchain: tcName, Topic: topic, Path: docFile, Opened: true})
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
