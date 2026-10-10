//go:build hmos_sign

// Sign a release ELF using OpenHarmony developer self-signing. Run on the host:
// go run -tags=hmos_sign ./scripts/harmonyos INPUT OUTPUT
package main

import (
	"bytes"
	"debug/buildinfo"
	"debug/elf"
	"fmt"
	"os"

	"github.com/Zxilly/cjv/internal/ohos/selfsign"
)

func main() {
	if len(os.Args) >= 3 && os.Args[1] == "--verify" {
		for _, path := range os.Args[2:] {
			data, err := os.ReadFile(path)
			if err == nil {
				err = checkStaticCLI(data)
			}
			if err == nil {
				err = selfsign.Verify(data)
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, path, err)
				os.Exit(1)
			}
		}
		return
	}
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: sign INPUT OUTPUT | sign --verify FILE...")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(input, output string) error {
	raw, err := os.ReadFile(input)
	if err != nil {
		return err
	}
	if err := checkStaticCLI(raw); err != nil {
		return err
	}
	signed, err := selfsign.Sign(raw)
	if err != nil {
		return err
	}
	if err := selfsign.Verify(signed); err != nil {
		return err
	}
	return os.WriteFile(output, signed, 0o755)
}

// checkStaticCLI validates the architecture and build settings of a release ELF.
func checkStaticCLI(data []byte) error {
	image, err := elf.NewFile(bytes.NewReader(data))
	if err != nil {
		return err
	}
	if image.Type != elf.ET_EXEC || (image.Machine != elf.EM_AARCH64 && image.Machine != elf.EM_X86_64) {
		return fmt.Errorf("expected a static arm64 or amd64 ELF executable")
	}
	for _, p := range image.Progs {
		if p.Type == elf.PT_INTERP || p.Type == elf.PT_DYNAMIC {
			return fmt.Errorf("release binary has a dynamic loader or library dependency")
		}
	}
	info, err := buildinfo.Read(bytes.NewReader(data))
	if err != nil {
		return err
	}
	settings := make(map[string]string, len(info.Settings))
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
	}
	if settings["GOOS"] != "openharmony" || settings["CGO_ENABLED"] != "0" {
		return fmt.Errorf("release binary must use GOOS=openharmony CGO_ENABLED=0")
	}
	return nil
}
