//go:build !windows

package selfupdate

func hideUpdateFile(string) error {
	return nil
}
