package dist

import sdktarget "github.com/Zxilly/cjv/internal/target"

type tupleMapping struct {
	Tuple       string // canonical target tuple (used as the manifest index key)
	NightlyOS   string // OS segment in nightly filenames
	NightlyArch string // arch segment in nightly filenames
}

func lookup(goos, goarch string) (tupleMapping, error) {
	id, err := sdktarget.HostIdentity(goos, goarch)
	if err != nil {
		return tupleMapping{}, err
	}
	return tupleMapping{
		Tuple:       id.Tuple(),
		NightlyOS:   id.NightlyOS(),
		NightlyArch: id.NightlyArch(),
	}, nil
}

func lookupTuple(tuple string) (tupleMapping, error) {
	id, err := sdktarget.ParseIdentity(tuple)
	if err != nil {
		return tupleMapping{}, err
	}
	return tupleMapping{
		Tuple:       id.Tuple(),
		NightlyOS:   id.NightlyOS(),
		NightlyArch: id.NightlyArch(),
	}, nil
}

// HostTupleFromGo returns the host target tuple for a (goos, goarch) pair.
func HostTupleFromGo(goos, goarch string) (string, error) {
	m, err := lookup(goos, goarch)
	if err != nil {
		return "", err
	}
	return m.Tuple, nil
}

// CurrentHostTuple returns the current host's target tuple. If defaultHost is
// non-empty it is parsed as "goos-goarch"; otherwise runtime values are used.
func CurrentHostTuple(defaultHost string) (string, error) {
	return sdktarget.CurrentHostTuple(defaultHost)
}

// CurrentTargetTuple resolves the host tuple and combines it with an optional
// environment value (e.g. "ohos") into a target tuple usable as a manifest
// index. An empty environment yields the bare host tuple.
func CurrentTargetTuple(defaultHost, environment string) (string, error) {
	id, err := sdktarget.CurrentTargetIdentity(defaultHost, environment)
	if err != nil {
		return "", err
	}
	return id.Tuple(), nil
}

func NightlyFilename(goos, goarch, version string) (string, error) {
	m, err := lookup(goos, goarch)
	if err != nil {
		return "", err
	}
	ext := ArchiveExt(goos)
	return "cangjie-sdk-" + m.NightlyOS + "-" + m.NightlyArch + "-" + version + ext, nil
}

// NightlyFilenameForTuple builds the nightly archive filename that corresponds
// to a target tuple, including the cross-compile target suffix when present.
func NightlyFilenameForTuple(tuple, version string) (string, error) {
	id, err := sdktarget.ParseIdentity(tuple)
	if err != nil {
		return "", err
	}
	return id.NightlyFilename(version), nil
}

// NightlyGOOS maps the SDK manifest's OS name to Go's GOOS (mac → darwin).
func NightlyGOOS(nightlyOS string) string {
	switch nightlyOS {
	case "windows":
		return "windows"
	case "mac":
		return "darwin"
	default:
		return nightlyOS
	}
}

func ArchiveExt(goos string) string {
	if goos == "windows" {
		return ".zip"
	}
	return ".tar.gz"
}
