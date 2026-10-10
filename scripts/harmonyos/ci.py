"""Build, install, and test the HarmonyOS CI tools without shell orchestration."""

import argparse
import hashlib
import os
from pathlib import Path
import subprocess


# CI subset of the official archive: Emulator, HDC, and HDC's USB library.
CLI_URL = (
    "https://github.com/Zxilly/static/releases/download/harmonyos-cli-26.0.0.821/"
    "commandline-tools-linux-x64-26.0.0.821-ci.zip"
)
CLI_SHA256 = "ee8ee0ef65453fea847f0a7e92ac1e29a7e59e9ef4e4b5c289fdf6fa8ba2f707"
EMULATOR_LIBRARIES = [
    "libatomic1", "libpulse0", "libegl1", "libgbm1", "libgl1", "libpng16-16t64",
    "libfontconfig1", "libfreetype6", "libxcb-icccm4", "libxcb-image0",
    "libxcb-keysyms1", "libxcb-randr0", "libxcb-render-util0", "libxcb-shape0",
    "libxcb-cursor0", "libxcb-xinerama0", "libxcb-xkb1", "libsm6", "libice6",
    "libxkbcommon-x11-0",
]
PACKAGES = [
    "./internal/ohos/...", "./internal/config", "./internal/env", "./internal/fsops",
    "./internal/target", "./internal/cjverr", "./internal/i18n", "./internal/retry",
]


def run(*args, **kwargs):
    print("+ " + " ".join(map(str, args)), flush=True)
    return subprocess.run(list(map(str, args)), check=True, **kwargs)


def target_env():
    return dict(os.environ, GOOS="openharmony", GOARCH="amd64", CGO_ENABLED="0")


def build():
    temp = Path(os.environ["RUNNER_TEMP"])
    run("go", "test", "./scripts/harmonyos/exec")
    run("go", "build", "-o", temp / "cjv-hmos-exec", "./scripts/harmonyos/exec")
    run(os.environ["HMOS_GO"], "test", "-c", "-o", temp / "selfsign.test",
        "./internal/ohos/selfsign", env=target_env())
    if github_env := os.environ.get("GITHUB_ENV"):
        with open(github_env, "a", encoding="utf-8") as output:
            output.write(f"HMOS_EMULATOR_HOME={temp / 'harmonyos-cli'}\n")


def install():
    root = Path(os.environ["HMOS_EMULATOR_HOME"])
    run("sudo", "chmod", "a+rw", "/dev/kvm")
    run("sudo", "apt-get", "update")
    run("sudo", "apt-get", "install", "-y", *EMULATOR_LIBRARIES)
    if os.access(root / "command-line-tools/emulator/Emulator", os.X_OK):
        return
    root.mkdir(parents=True, exist_ok=True)
    archive = Path(os.environ["RUNNER_TEMP"]) / "hmos-cli.zip"
    try:
        run("curl", "-fL", "--retry", "3", "--connect-timeout", "30", "--max-time", "900",
            CLI_URL, "-o", archive)
        with archive.open("rb") as source:
            digest = hashlib.file_digest(source, "sha256").hexdigest()
        if digest != CLI_SHA256:
            raise RuntimeError(f"Emulator archive checksum mismatch: {digest}")
        run("unzip", "-q", archive, "command-line-tools/emulator/*",
            "command-line-tools/sdk/default/openharmony/toolchains/hdc",
            "command-line-tools/sdk/default/openharmony/toolchains/libusb_shared.so",
            "-d", root)
    finally:
        archive.unlink(missing_ok=True)


def test():
    source = Path(os.environ["HMOS_TEST_SOURCE"])
    temp = Path(os.environ["RUNNER_TEMP"])
    wrapper = temp / "cjv-hmos-exec"
    binary = temp / "selfsign.test"
    cwd = source / "internal/ohos/selfsign"
    if not os.environ.get("HMOS_SIGN"):
        raise RuntimeError("HMOS_SIGN is required for the signed test run")
    run(wrapper, binary, "-test.v", "-test.timeout=5m", cwd=cwd,
        env=dict(os.environ, HMOS_SIGN=""))
    run(wrapper, binary, "-test.v", "-test.timeout=5m", cwd=cwd)
    # An invalid flag must preserve the guest's exit code through the HDC wrapper.
    result = subprocess.run([str(wrapper), str(binary), "-invalid-test-flag"],
                            cwd=cwd, stdout=subprocess.DEVNULL, check=False)
    if result.returncode != 2:
        raise RuntimeError(f"Expected guest exit code 2, got {result.returncode}")
    run(os.environ["HMOS_GO"], "test", "-exec", wrapper, "-p", "1", "-count=1",
        "-timeout=5m", *PACKAGES, cwd=source, env=target_env())


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("step", choices=["build", "install", "test"])
    args = parser.parse_args()
    {"build": build, "install": install, "test": test}[args.step]()
