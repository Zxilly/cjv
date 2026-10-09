"""Inspect the official ARM64 emulator's CLI and embedded QEMU without KVM."""
import importlib.util
import json
from pathlib import Path
import socket
import subprocess
import time

from patch import patch

spec = importlib.util.spec_from_file_location("official_probe", Path(__file__).resolve().parents[1] / "harmonyos-emulator/probe.py")
probe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(probe)
probe.LOGS.mkdir(exist_ok=True)
probe.hardware()
probe.download()
probe.emulator("version", "-version")
probe.emulator("help", "-help")
for label, args in [
    ("accel-help", ["-accel", "help"]),
    ("qemu-accel-help", ["qemu", "-accel", "help"]),
    ("dash-qemu-accel-help", ["-qemu", "-accel", "help"]),
    ("double-dash-accel-help", ["--", "-accel", "help"]),
    ("machine-help", ["-machine", "help"]),
    ("tcg-empty-machine", ["-machine", "virt", "-accel", "tcg", "-cpu", "max", "-display", "none", "-nodefaults", "-S"]),
]:
    probe.emulator(label, *args, timeout=15)
# Check whether argv[0] selects the embedded QEMU entry point.
alias = probe.EMULATOR.parent / "qemu-system-aarch64"
alias.symlink_to(probe.EMULATOR.name)
probe.run("qemu-argv0-accel-help", [str(alias), "-accel", "help"], env=probe.emulator_env(), cwd=alias.parent, timeout=15)
probe.report("Parameter inspection complete; exit status does not establish guest boot readiness.")

raw = patch(probe.EMULATOR, probe.EMULATOR.with_name("Emulator-qemu-tcg"))
for label, args in [
    ("embedded-version", ["--version"]),
    ("embedded-accel-help", ["-accel", "help"]),
    ("embedded-machine-help", ["-machine", "help"]),
]:
    probe.run(label, [str(raw), *args], env=probe.emulator_env(), cwd=raw.parent, timeout=15)
if "tcg" not in (probe.LOGS / "embedded-accel-help.log").read_text():
    raise SystemExit("The embedded QEMU entry did not expose TCG")

# Execute real AArch64 instructions under TCG. The bare-metal guest prints to
# the standard virt machine's PL011 UART; no HarmonyOS image is involved yet.
marker = "OFFICIAL_EMULATOR_TCG_ARM64_OK\n"
assembly = [".global _start", ".text", "_start:", "mov x0, #0x09000000"]
for character in marker:
    assembly.extend([f"mov w1, #{ord(character)}", "str w1, [x0]"])
assembly.extend(["1: wfi", "b 1b"])
source = probe.LOGS / "tcg-marker.S"
source.write_text("\n".join(assembly) + "\n")
guest = probe.LOGS / "tcg-marker.elf"
subprocess.run(["gcc", "-nostdlib", "-static", "-Wl,--build-id=none,-Ttext=0x40080000,-e,_start",
                "-o", str(guest), str(source)], check=True)
serial = probe.LOGS / "tcg-marker-serial.log"
qmp = probe.ROOT / "tcg-marker.qmp"
command = [str(raw), "-machine", "virt", "-accel", "tcg,thread=multi", "-cpu", "max",
           "-m", "128M", "-display", "none", "-monitor", "none",
           "-serial", f"file:{serial}", "-qmp", f"unix:{qmp},server=on,wait=off",
           "-device", f"loader,file={guest},cpu-num=0"]
with (probe.LOGS / "tcg-marker-process.log").open("w") as log:
    log.write("$ " + " ".join(command) + "\n")
    log.flush()
    process = subprocess.Popen(command, stdout=log, stderr=subprocess.STDOUT,
                               env=probe.emulator_env(), cwd=raw.parent)
    try:
        deadline = time.monotonic() + 30
        while time.monotonic() < deadline and process.poll() is None:
            if serial.exists() and marker.strip() in serial.read_text(errors="replace"):
                break
            time.sleep(0.2)
        if not serial.exists() or marker.strip() not in serial.read_text(errors="replace"):
            raise RuntimeError("TCG did not execute the AArch64 UART marker")
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as client:
            client.settimeout(5)
            client.connect(str(qmp))
            stream = client.makefile("rwb")
            responses = [stream.readline().decode()]
            for request in ("qmp_capabilities", "query-kvm", "quit"):
                stream.write((json.dumps({"execute": request, "id": request}) + "\n").encode())
                stream.flush()
                while True:
                    line = stream.readline()
                    if not line:
                        raise RuntimeError("QMP disconnected before replying")
                    responses.append(line.decode())
                    if json.loads(line).get("id") == request:
                        break
            (probe.LOGS / "tcg-marker-qmp.log").write_text("".join(responses))
        probe.report("**TCG verified:** the official embedded QEMU executed an AArch64 guest and printed the UART marker without KVM.")
    finally:
        if process.poll() is None:
            process.terminate()
        try:
            process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait()
        print((probe.LOGS / "tcg-marker-process.log").read_text(errors="replace"))
        if serial.exists():
            print(serial.read_text(errors="replace"))
