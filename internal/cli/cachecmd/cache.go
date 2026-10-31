// Package cachecmd manages reusable metadata and verified SDK archives.
package cachecmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/spf13/cobra"
)

type result struct {
	SchemaVersion int               `json:"schema_version"`
	Path          string            `json:"path"`
	Entries       []dist.CacheEntry `json:"entries"`
	Removed       *int              `json:"removed,omitempty"`
	action        string
}

func (r result) Text() string {
	if r.action == "dir" {
		return r.Path
	}
	if r.Removed != nil {
		return fmt.Sprintf(i18n.T("CacheRemoved", nil), *r.Removed)
	}
	var b strings.Builder
	fmt.Fprintln(&b, r.Path)
	for _, e := range r.Entries {
		fmt.Fprintf(&b, i18n.T("CacheEntry", nil), e.Key, e.Kind, e.Size)
		if e.Problem != "" {
			fmt.Fprintf(&b, "  (%s)", e.Problem)
		}
		fmt.Fprintln(&b)
	}
	return b.String()
}

func Register(root *cobra.Command, shared *clictx.Context) {
	cache := &cobra.Command{Use: "cache", Short: i18n.T("CacheCmdShort", nil)}
	shortIDs := map[string]string{
		"dir": "CacheDirShort", "list": "CacheListShort",
		"clean": "CacheCleanShort", "prune": "CachePruneShort",
	}
	for _, action := range []string{"dir", "list", "clean", "prune"} {
		var age time.Duration
		cmd := &cobra.Command{Use: action, Short: i18n.T(shortIDs[action], nil), Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			directory, err := dist.CacheDirectory()
			if err != nil {
				return err
			}
			r := result{SchemaVersion: 1, Path: directory, Entries: []dist.CacheEntry{}, action: action}
			switch action {
			case "list":
				r.Entries, err = dist.ListCache(directory)
			case "clean", "prune":
				var cutoff time.Time
				if action == "prune" {
					if age < 0 {
						return errors.New(i18n.T("CacheInvalidMaxAge", nil))
					}
					cutoff = time.Now().Add(-age)
				}
				var removed int
				removed, err = dist.PruneCache(cmd.Context(), directory, cutoff)
				r.Removed = &removed
			}
			if err != nil {
				return err
			}
			return shared.Output.RenderTo(cmd.OutOrStdout(), r)
		}}
		if action == "prune" {
			cmd.Flags().DurationVar(&age, "max-age", 30*24*time.Hour, i18n.T("CacheFlagMaxAge", nil))
		}
		cache.AddCommand(cmd)
	}
	root.AddCommand(cache)
}
