"""Probe the official emulator on disposable Linux CI runners, without Go."""

import hashlib
import json
import os
from pathlib import Path
import platform
import struct
import subprocess
import sys
import time


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


def emulator_env():
    env = dict(os.environ)
    env["QT_QPA_PLATFORM"] = "offscreen"
    env["QT_QPA_PLATFORM_PLUGIN_PATH"] = str(EMULATOR.parent / "plugins/platforms")
    env["LD_LIBRARY_PATH"] = f"{EMULATOR.parent}:{EMULATOR.parent / 'lib'}"
    return env


def cli():
    with EMULATOR.open("rb") as binary:
        header = binary.read(20)
    if header[:6] != b"\x7fELF\x02\x01":
        raise RuntimeError("Expected an ELF64 little-endian official emulator")
    machine = struct.unpack_from("<H", header, 18)[0]
    report(f"- Official Emulator ELF machine: {machine} (62=x86_64, 183=AArch64)")
    run("elf", ["file", str(EMULATOR)])
    env = emulator_env()
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


def boot():
    env = emulator_env()
    images = ROOT / "images"
    instances = ROOT / "instances"
    name = "cjv-pc-ci"
    version = "HarmonyOS 6.1.1(24)"
    hdc = ROOT / "command-line-tools/sdk/default/openharmony/toolchains/hdc"

    def emulator(label, *args, timeout=120):
        return run(label, [str(EMULATOR), *map(str, args)], timeout=timeout,
                   env=env, cwd=EMULATOR.parent)

    if emulator("license", "-license", "accept"):
        return 1
    if emulator("install", "-install", "-deviceType", "2in1", "-osVersion", version,
                "-imageRoot", images, "-force", timeout=1200):
        report("**Blocked:** official PC image installation failed; see install.log.")
        return 1
    run("image-files", ["find", str(images), "-maxdepth", "5", "-type", "f",
                        "-printf", "%p %s bytes\n"])
    if emulator("create", "-create", name, "-deviceType", "2in1", "-osVersion", version,
                "-imageRoot", images, "-instancePath", instances, "-storage", "6",
                "-memory", "4", "-hotBoot", "false"):
        report("**Blocked:** official PC emulator creation failed; see create.log.")
        return 1
    run("hdc-version", [str(hdc), "-v"], env=env)
    # The launcher may stay attached for the lifetime of the virtual machine.
    with (LOGS / "start.log").open("w") as output:
        process = subprocess.Popen([
            str(EMULATOR), "-start", name, "-instancePath", str(instances),
            "-imageRoot", str(images), "-hdcport", "5555", "-bootMode", "reset", "-noWindow",
        ], stdout=output, stderr=subprocess.STDOUT, stdin=subprocess.DEVNULL,
            env=env, cwd=EMULATOR.parent)
        try:
            deadline = time.monotonic() + 300
            while time.monotonic() < deadline:
                run("hdc-connect", [str(hdc), "tconn", "127.0.0.1:5555"], env=env, timeout=15)
                code = run("guest-uname", [str(hdc), "-t", "127.0.0.1:5555", "shell",
                                          "uname", "-a"], env=env, timeout=15)
                if code == 0 and "Linux " in (LOGS / "guest-uname.log").read_text():
                    run("guest-environment", [str(hdc), "-t", "127.0.0.1:5555", "shell",
                        "id; uname -m; echo HOME=$HOME; pwd; mount; ls -ld /data /storage"], env=env)
                    report("**Ready:** official PC emulator booted and HDC executed a guest shell.")
                    return 0
                if process.poll() not in (None, 0):
                    break
                time.sleep(10)
            report("**Blocked:** no usable HDC shell after emulator startup; see start.log.")
            return 1
        finally:
            emulator("stop", "-stop", name)
            if process.poll() is None:
                process.terminate()
                try:
                    process.wait(timeout=15)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()
            print((LOGS / "start.log").read_text(errors="replace"), flush=True)


if __name__ == "__main__":
    LOGS.mkdir(exist_ok=True)
    sys.exit({"hardware": hardware, "download": download, "cli": cli, "boot": boot}[sys.argv[1]]())
