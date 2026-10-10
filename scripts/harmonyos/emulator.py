"""Run a command while an official x86_64 PC emulator is available through HDC."""

import os
from pathlib import Path
import subprocess
import sys
import time


root = Path(os.environ["HMOS_EMULATOR_HOME"]).resolve()
emulator = root / "command-line-tools/emulator/Emulator"
toolchains = root / "command-line-tools/sdk/default/openharmony/toolchains"
images = root / "images"
instances = root / "instances"
logs = Path(os.environ.get("HMOS_LOGS", str(root / "logs"))).resolve()
version = "HarmonyOS 6.1.1(24)"
name = "cjv_tests"
target = "127.0.0.1:15555"
guest = "/data/local/tmp/cjv-tests"
env = dict(os.environ, QT_QPA_PLATFORM="offscreen",
           QT_QPA_PLATFORM_PLUGIN_PATH=str(emulator.parent / "plugins/platforms"),
           LD_LIBRARY_PATH=f"{emulator.parent}:{emulator.parent / 'lib'}:{toolchains}",
           HDC=str(toolchains / "hdc"), HDC_TARGET=target,
           HMOS_TEST_SOURCE=str(Path.cwd()), HMOS_TEST_ROOT=guest)


def run(command, timeout=120):
    return subprocess.run(list(map(str, command)), env=env, stdin=subprocess.DEVNULL,
                          check=True, timeout=timeout)


def emu(*args, timeout=120):
    print(f"Emulator {args[0]}", flush=True)
    with (logs / "setup.log").open("a") as output:
        return subprocess.run(list(map(str, [emulator, *args])), env=env,
                              stdin=subprocess.DEVNULL, stdout=output, stderr=subprocess.STDOUT,
                              check=True, timeout=timeout)


def hdc(*args):
    return run([env["HDC"], "-t", target, *args])


def main():
    logs.mkdir(parents=True, exist_ok=True)
    instances.mkdir(parents=True, exist_ok=True)
    # Fail early if the runner cannot actually create a VM.
    import fcntl
    with open("/dev/kvm", "rb+") as device:
        os.close(fcntl.ioctl(device.fileno(), 0xAE01, 0))
    emu("-license", "accept")
    if not any(images.glob("system-image/HarmonyOS-6.1.1/pc*/system.img")):
        emu("-install", "-deviceType", "2in1", "-osVersion", version,
            "-imageRoot", images, "-force", timeout=1200)
    if not any(images.glob("system-image/HarmonyOS-6.1.1/pc*/system.img")):
        raise RuntimeError("PC image installation failed")
    emu("-create", name, "-deviceType", "2in1", "-osVersion", version,
        "-imageRoot", images, "-instancePath", instances,
        "-storage", "6", "-memory", "4", "-hotBoot", "false")
    if not (instances / name / "config.ini").is_file():
        raise RuntimeError("PC emulator creation failed")
    with (logs / "emulator.log").open("w") as output:
        process = subprocess.Popen([
            str(emulator), "-start", name, "-instancePath", str(instances),
            "-imageRoot", str(images), "-hdcPort", "15555", "-bootMode", "reset", "-noWindow",
        ], env=env, stdin=subprocess.DEVNULL, stdout=output, stderr=subprocess.STDOUT)
        try:
            deadline = time.monotonic() + 300
            while time.monotonic() < deadline:
                subprocess.run([env["HDC"], "tconn", target], env=env, stdin=subprocess.DEVNULL,
                               capture_output=True, timeout=15)
                probe = subprocess.run([env["HDC"], "-t", target, "shell", "uname -m; id; getenforce"],
                                       env=env, stdin=subprocess.DEVNULL,
                                       capture_output=True, text=True, timeout=15)
                if probe.returncode == 0 and "x86_64" in probe.stdout and "uid=" in probe.stdout:
                    print(probe.stdout, flush=True)
                    (logs / "guest.txt").write_text(probe.stdout)
                    break
                if process.poll() not in (None, 0):
                    raise RuntimeError("emulator exited before HDC was ready")
                time.sleep(5)
            else:
                raise RuntimeError("HDC startup timed out")
            # Use tracked and untracked source paths so local runs test pending edits too.
            files = subprocess.check_output(["git", "ls-files", "-z", "--cached", "--others", "--exclude-standard"],
                                            stdin=subprocess.DEVNULL)
            archive = logs / "source.tar"
            subprocess.run(["tar", "--null", "-T", "-", "-cf", str(archive)], input=files, check=True)
            hdc("shell", f"mkdir -p {guest}/src")
            hdc("file", "send", archive, guest + "/source.tar")
            extracted = subprocess.check_output([
                env["HDC"], "-t", target, "shell",
                f"cd {guest}/src && tar xf ../source.tar && echo CJV_SOURCE_READY",
            ], env=env, stdin=subprocess.DEVNULL, timeout=120, text=True)
            if "CJV_SOURCE_READY" not in extracted:
                raise RuntimeError(f"guest source extraction failed: {extracted}")
            return subprocess.run(sys.argv[1:], env=env, check=False).returncode
        finally:
            try:
                emu("-stop", name, "-instancePath", instances)
            finally:
                if process.poll() is None:
                    process.terminate()
                    try:
                        process.wait(timeout=15)
                    except subprocess.TimeoutExpired:
                        process.kill()
                        process.wait()


if __name__ == "__main__":
    sys.exit(main())
