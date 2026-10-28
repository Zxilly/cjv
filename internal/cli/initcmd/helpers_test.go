package initcmd

import (
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/cli/clitest"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) { clitest.Main(m) }

// newTestCommands returns init with its flag defaults applied.
func newTestCommands() *commands {
	c := &commands{ctx: clictx.New("dev", "")}
	c.command()
	return c
}

// Huh keeps a string-based form model and adapts it when Run starts Bubble
// Tea. Observe complete views before the renderer turns them into cursor diffs;
// terminal output is not an append-only transcript of the visible text.
type formTestModel struct {
	huh.Model
	view atomic.Value
}

func (m *formTestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.Model.Update(msg)
	m.Model = next
	return m, cmd
}

func (m *formTestModel) View() tea.View {
	content := m.Model.View()
	m.view.Store(ansi.Strip(content))
	view := tea.NewView(content)
	view.ReportFocus = true
	return view
}

func (m *formTestModel) waitForText(t *testing.T, text string) {
	t.Helper()
	// Read only the immutable snapshot, never the live form being updated by
	// Bubble Tea's event loop. Failed assertions show the last complete view.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		view, _ := m.view.Load().(string)
		assert.Contains(c, view, text)
	}, 2*time.Second, 10*time.Millisecond)
}
