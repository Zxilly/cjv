package cli

import (
	"fmt"
	"os"
	"path/filepath"

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

	tcDir, _, err := resolveToolchainArg(docToolchain)
	if err != nil {
		return err
	}

	roots, err := componentlib.RootsFor(filepath.Base(tcDir))
	if err != nil {
		return err
	}
	docFile, err := componentlib.ResolveDocPath(roots, topic)
	if err != nil {
		return err
	}

	if docPath {
		fmt.Println(docFile)
		return nil
	}

	fmt.Fprintln(os.Stderr, i18n.T("OpeningDocs", nil))
	if err := openURLFunc(fileURL(docFile)); err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("OpeningDocsBrowserFailed", i18n.MsgData{"Path": docFile}))
		return err
	}
	return nil
}

// fileURL returns a file:// URL pointing at an absolute local file path. We
// do this by hand instead of pulling net/url.Parse to keep Windows drive
// letters intact ("C:\\foo" → "file:///C:/foo").
func fileURL(absPath string) string {
	clean := filepath.ToSlash(absPath)
	if len(clean) > 0 && clean[0] != '/' {
		clean = "/" + clean
	}
	return "file://" + clean
}
