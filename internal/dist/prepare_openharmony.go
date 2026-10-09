//go:build openharmony

package dist

import (
	"context"

	"github.com/Zxilly/cjv/internal/ohos"
)

func preparePlatformTree(ctx context.Context, root string) error {
	return ohos.PrepareTree(ctx, root)
}
