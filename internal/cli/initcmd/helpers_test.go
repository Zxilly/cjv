package initcmd

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/cli/clitest"
)

func TestMain(m *testing.M) { clitest.Main(m) }

// newTestCommands returns init with its flag defaults applied.
func newTestCommands() *commands {
	c := &commands{ctx: clictx.New("dev", "")}
	c.command()
	return c
}

// Huh keeps a string-based form model and adapts it when Run starts Bubble
// Tea. Use the same view contract in the terminal-driven test harness.
type formTestModel struct{ huh.Model }

func (m formTestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.Model.Update(msg)
	return formTestModel{next}, cmd
}

func (m formTestModel) View() tea.View {
	view := tea.NewView(m.Model.View())
	view.ReportFocus = true
	return view
}
