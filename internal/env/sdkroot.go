package env

func applyDarwinSDKRoot(cfg *EnvConfig, current string, lookup func() (string, error)) {
	if current != "" || cfg == nil || lookup == nil {
		return
	}
	value, err := lookup()
	if err != nil || value == "" {
		return
	}
	if cfg.Vars == nil {
		cfg.Vars = make(map[string]string)
	}
	cfg.Vars["SDKROOT"] = value
}
