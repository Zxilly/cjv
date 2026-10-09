"""Try the official PC guest with the original frontend forced to TCG."""

import importlib.util
import os
from pathlib import Path

from patch import patch


spec = importlib.util.spec_from_file_location("official_probe", Path(__file__).resolve().parents[1] / "harmonyos-emulator/probe.py")
probe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(probe)
probe.LOGS.mkdir(exist_ok=True)
probe.EMULATOR = patch(probe.EMULATOR, probe.EMULATOR.with_name("Emulator-frontend-tcg"), "frontend")
os.environ["LIBGL_ALWAYS_SOFTWARE"] = "1"
raise SystemExit(probe.boot())
