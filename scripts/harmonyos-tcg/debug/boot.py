"""Keep the experimental guest available for bounded live SSH diagnosis."""

import importlib.util
import os
from pathlib import Path
import sys

HERE = Path(__file__).resolve()
sys.path.insert(0, str(HERE.parents[1]))
from patch import patch

spec = importlib.util.spec_from_file_location("probe", HERE.parents[2] / "harmonyos-emulator/probe.py")
probe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(probe)
probe.LOGS.mkdir(exist_ok=True)
probe.EMULATOR = patch(probe.EMULATOR, probe.EMULATOR.with_name("Emulator-frontend-tcg"), "frontend")
os.environ["LIBGL_ALWAYS_SOFTWARE"] = "1"
# The debug shell can stop this wait by terminating this process; the normal
# boot helper still collects native logs and stops the emulator on completion.
raise SystemExit(probe.boot(timeout=3000, cpu_count=int(os.environ.get("HARMONYOS_CPU_COUNT", "1"))))
