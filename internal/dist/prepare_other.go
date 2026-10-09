//go:build !openharmony

package dist

import "context"

func preparePlatformTree(context.Context, string) error { return nil }
