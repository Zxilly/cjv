package env

func applyDarwinSDKRoot(vars map[string]string, current string, lookup func() (string, error)) {
	if current != "" || vars["SDKROOT"] != "" || lookup == nil {
		return
	}
	value, err := lookup()
	if err != nil || value == "" {
		return
	}
	vars["SDKROOT"] = value
}
