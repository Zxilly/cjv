"""Try the official PC guest with the original frontend forced to TCG."""

import importlib.util
import os
from pathlib import Path
import shutil

from patch import patch


spec = importlib.util.spec_from_file_location("official_probe", Path(__file__).resolve().parents[1] / "harmonyos-emulator/probe.py")
probe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(probe)
probe.LOGS.mkdir(exist_ok=True)
probe.EMULATOR = patch(probe.EMULATOR, probe.EMULATOR.with_name("Emulator-frontend-tcg"), "frontend")
os.environ["LIBGL_ALWAYS_SOFTWARE"] = "1"
code = probe.boot()
if code:
    probe.run("crash-backtrace", [
        "gdb", "-q", "-batch", "-ex", "set pagination off",
        "-ex", "set confirm off", "-ex", "handle SIGUSR1 SIGUSR2 SIGPIPE nostop noprint pass",
        "-ex", "start", "-ex", "set $emu_base = (unsigned long)&main - 0xc4512c",
        "-ex", "break *($emu_base + 0xf21c38)",
        "-ex", "break *($emu_base + 0x6ada88)", "-ex", "continue",
        "-ex", "bt 30", "-ex", "info proc mappings",
        "-x", str(Path(__file__).with_name("crash.gdb").resolve()),
        "--args", str(probe.EMULATOR), "-start", "cjv_pc_ci",
        "-instancePath", str(probe.ROOT / "instances"),
        "-imageRoot", str(probe.IMAGES), "-hdcPort", "15555",
        "-bootMode", "reset", "-noWindow",
    ], env=probe.emulator_env(), cwd=probe.EMULATOR.parent, timeout=90)
# The CLI's logZip can fail after a crash; preserve small native log/config files.
for root in [probe.ROOT / "instances", probe.EMULATOR.parent, Path.home() / ".Huawei"]:
    for source in root.rglob("*"):
        if source.is_file() and source.suffix in {".log", ".json", ".ini", ".txt"} and source.stat().st_size < 10_000_000:
            target = probe.LOGS / "native" / root.name / source.relative_to(root)
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, target)
raise SystemExit(code)
