"""Inspect the official ARM64 emulator's CLI and embedded QEMU without KVM."""
import importlib.util
import os
from pathlib import Path
import subprocess

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
