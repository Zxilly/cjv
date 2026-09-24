package lifecycle

// UpgradeToolchain exposes the upgrade step with a caller-chosen current
// name, so tests can pin which installed version is replaced independently
// of the channel lookup UpdateInstalled performs.
var UpgradeToolchain = upgradeToolchain
