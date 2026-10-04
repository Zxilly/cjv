package lifecycle

import "github.com/Zxilly/cjv/internal/toolchain"

func selectedIdentity(rt ResolvedToolchain, tracking bool) string {
	if !tracking {
		return rt.Name
	}
	name, _ := toolchain.ParseToolchainName(rt.Name)
	name.Version = ""
	return name.String()
}
