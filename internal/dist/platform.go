package dist

import sdktarget "github.com/Zxilly/cjv/internal/target"

// HostTupleFromGo returns the host target tuple for a (goos, goarch) pair.
func HostTupleFromGo(goos, goarch string) (string, error) {
	id, err := sdktarget.HostIdentity(goos, goarch)
	if err != nil {
		return "", err
	}
	return id.Tuple(), nil
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
