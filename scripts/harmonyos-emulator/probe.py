"""Probe the official emulator on disposable Linux CI runners, without Go."""

import hashlib
import json
import os
from pathlib import Path
import platform
import struct
import subprocess
import sys


LOGS = Path("emulator-diagnostics").resolve()
ROOT = Path(os.environ.get("RUNNER_TEMP", "/tmp")) / "harmonyos-cli"
EMULATOR = ROOT / "command-line-tools/emulator/Emulator"
METADATA = json.loads(Path(__file__).with_name("cli.json").read_text())


def report(message):
    print(message, flush=True)
    with (LOGS / "summary.md").open("a") as output:
        output.write(message + "\n")
    if os.environ.get("GITHUB_STEP_SUMMARY"):
        with open(os.environ["GITHUB_STEP_SUMMARY"], "a") as output:
            output.write(message + "\n")


def run(label, command, timeout=120, env=None, cwd=None):
    log = LOGS / (label + ".log")
    with log.open("w") as output:
        output.write("$ " + " ".join(map(str, command)) + "\n")
        output.flush()
        try:
            result = subprocess.run(
                command, stdout=output, stderr=subprocess.STDOUT,
                stdin=subprocess.DEVNULL, timeout=timeout, env=env, cwd=cwd,
                check=False,
            )
            code = result.returncode
        except (OSError, subprocess.TimeoutExpired) as error:
            output.write(str(error) + "\n")
            code = 126
    print(log.read_text(errors="replace"), flush=True)
    report(f"- `{label}` exit code: {code}")
    return code


def hardware():
    import fcntl

    report(f"## Runner: {platform.machine()} / official CLI {METADATA['version']}")
    run("hardware", ["sh", "-c", "uname -a; lscpu; id; df -h /; ls -l /dev/kvm /dev/dri"])
    # Check actual access and the KVM API, not just a CPU virtualization flag.
    try:
        with open("/dev/kvm", "rb+") as device:
            version = fcntl.ioctl(device.fileno(), 0xAE00, 0)
        report(f"- KVM API version: {version}")
    except OSError as error:
        report(f"- KVM unavailable: {error}")
    report(f"- DRM nodes: {[str(path) for path in Path('/dev/dri').glob('*')]}")


def download():
    ROOT.mkdir(parents=True, exist_ok=True)
    archive = ROOT / METADATA["archive"]
    subprocess.run([
        "curl", "--fail", "--location", "--retry", "3", "--connect-timeout", "30",
        "--max-time", "900", "--output", str(archive), METADATA["url"],
    ], check=True)
    with archive.open("rb") as source:
        checksum = hashlib.file_digest(source, "sha256").hexdigest()
    report(f"- Download SHA-256: `{checksum}`")
    if checksum != METADATA["sha256"]:
        raise RuntimeError("Official CLI archive does not match the pinned checksum")
    subprocess.run([
        "unzip", "-q", str(archive), "command-line-tools/emulator/*",
        "command-line-tools/sdk/default/openharmony/toolchains/hdc", "-d", str(ROOT),
    ], check=True)
    archive.unlink()


def cli():
    with EMULATOR.open("rb") as binary:
        header = binary.read(20)
    if header[:6] != b"\x7fELF\x02\x01":
        raise RuntimeError("Expected an ELF64 little-endian official emulator")
    machine = struct.unpack_from("<H", header, 18)[0]
    report(f"- Official Emulator ELF machine: {machine} (62=x86_64, 183=AArch64)")
    run("elf", ["file", str(EMULATOR)])
    env = dict(os.environ)
    env["QT_QPA_PLATFORM"] = "offscreen"
    env["QT_QPA_PLATFORM_PLUGIN_PATH"] = str(EMULATOR.parent / "plugins/platforms")
    env["LD_LIBRARY_PATH"] = f"{EMULATOR.parent}:{EMULATOR.parent / 'lib'}"
    version = run("version", [str(EMULATOR), "-version"], env=env, cwd=EMULATOR.parent)
    native_machine = {"x86_64": 62, "aarch64": 183}.get(platform.machine())
    if machine != native_machine:
        report("**Blocked:** the official Linux CLI package is not native to this runner.")
        return 1
    if version:
        run("shared-libraries", ["ldd", str(EMULATOR)])
        report("**Blocked:** the native emulator executable could not run.")
        return 1
    run("help", [str(EMULATOR), "-help"], env=env, cwd=EMULATOR.parent)
    catalog = run("images-2in1", [str(EMULATOR), "-imageList", "-deviceType", "2in1",
                                "-downloaded", "false"], timeout=180, env=env,
                  cwd=EMULATOR.parent)
    run("images-all", [str(EMULATOR), "-imageList", "-downloaded", "false"],
        timeout=180, env=env, cwd=EMULATOR.parent)
    report("CLI/catalog probe finished. A booted guest with an HDC shell is still required for readiness.")
    return int(catalog != 0)


if __name__ == "__main__":
    LOGS.mkdir(exist_ok=True)
    sys.exit({"hardware": hardware, "download": download, "cli": cli}[sys.argv[1]]())
