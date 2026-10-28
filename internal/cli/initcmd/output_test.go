package initcmd

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

func TestInitTextPreservesCommandsAndLineBreaks(t *testing.T) {
	text := "Use `cjv install lts` to install a toolchain.\n运行 `cjv self uninstall` 卸载。"
	rendered := renderInitText(text, false)
	assert.Equal(t, "\n  Use cjv install lts to install a toolchain.\n  运行 cjv self uninstall 卸载。\n\n", rendered)
	assert.NotContains(t, rendered, "\x1b")
	assert.Contains(t, renderInitText("Unclosed `command", false), "Unclosed `command")
}

func TestInitTextWrapsWideAndColoredText(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	previous := color.NoColor
	color.NoColor = false
	t.Cleanup(func() { color.NoColor = previous })
	for _, text := range []string{
		strings.Repeat("安装仓颉工具链", 25),
		strings.Repeat("Use `cjv install lts` to install a toolchain. ", 8),
	} {
		rendered := renderInitText(text, true)
		for line := range strings.SplitSeq(rendered, "\n") {
			assert.LessOrEqual(t, ansi.StringWidth(line), 100)
		}
		// Wrapping must neither drop nor duplicate content, including CJK.
		compact := func(value string) string { return strings.Join(strings.Fields(value), "") }
		assert.Equal(t, compact(strings.ReplaceAll(text, "`", "")), compact(ansi.Strip(rendered)))
	}
	assert.Contains(t, renderInitText("Run `cjv`", true), "\x1b[")
	assert.NotContains(t, renderInitText("Run `cjv`", false), "\x1b[")
	color.NoColor = true
	assert.NotContains(t, renderInitText("Run `cjv`", true), "\x1b[")
}
