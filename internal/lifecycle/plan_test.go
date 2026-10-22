package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/stretchr/testify/require"
)

func TestUpdatePlanIsReadOnlyAndMatchesGroupApplication(t *testing.T) {
	home := updateHome(t)
	selectRelease(t, "1.0.0", "ohos")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, Options{}))
	selectRelease(t, "2.0.0")
	before, err := historyTreeDigest(t.Context(), home)
	require.NoError(t, err)
	plan, err := PlanUpdates(t.Context(), nil, Options{})
	require.NoError(t, err)
	require.Len(t, plan.Toolchains, 1)
	require.Equal(t, "blocked", plan.Toolchains[0].Status)
	require.Equal(t, "sts-1.0.0", plan.Toolchains[0].Selected)
	require.Equal(t, "sts-2.0.0", plan.Toolchains[0].Latest)
	require.NotEmpty(t, plan.Toolchains[0].Blockers)
	require.False(t, plan.Toolchains[0].UpdateAvailable)
	after, err := historyTreeDigest(t.Context(), home)
	require.NoError(t, err)
	require.Equal(t, before, after)
	_, err = UpdateInstalled(t.Context(), parse(t, "sts"), Options{})
	require.Error(t, err)
	require.Equal(t, "sts-1.0.0", readRelease(t, home, "sts").Release)
	selectRelease(t, "2.0.0", "ohos")
	plan, err = PlanUpdates(t.Context(), nil, Options{})
	require.NoError(t, err)
	require.Len(t, plan.Toolchains, 1)
	require.True(t, plan.Toolchains[0].UpdateAvailable)
	require.Len(t, plan.Toolchains[0].Members, 2)
	_, err = UpdateInstalled(t.Context(), parse(t, "sts"), Options{})
	require.NoError(t, err)
	for _, member := range plan.Toolchains[0].Members {
		require.Equal(t, member.Selected, readRelease(t, home, member.Name).Release)
	}
}

func TestUpdatePlanDoesNotCreateOrRecoverHome(t *testing.T) {
	home := filepath.Join(t.TempDir(), "not-created")
	configtest.Isolate(t, home)
	report, err := PlanUpdates(t.Context(), nil, Options{})
	require.NoError(t, err)
	require.True(t, report.NoneInstalled)
	require.NotNil(t, report.Toolchains)
	require.NoDirExists(t, home)
	require.NoError(t, os.MkdirAll(filepath.Join(home, config.ToolchainsSubdir, "sts.staging"), 0o755))
	_, err = PlanUpdates(t.Context(), nil, Options{})
	require.NoError(t, err)
	require.DirExists(t, filepath.Join(home, config.ToolchainsSubdir, "sts.staging"))
}
