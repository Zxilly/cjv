"""Expose the embedded QEMU entry point in one exact official ARM64 build.

This experimental copy retains the original ELF loader and constructors, but
replaces main's first instruction with a branch to RunQemuMain(argc, argv).
The original binary and release archive are never changed.
"""

import hashlib
from pathlib import Path
import struct
import sys


EMULATOR_SHA256 = "62dd21c38f6e21e242f66f6c17753dfa445dd4073e072d851d4af41f8ff9ab4e"
MAIN = 0xC4512C
RUN_QEMU_MAIN = 0xC509A4


def patch(source: Path, destination: Path, mode="qemu"):
    if source.resolve() == destination.resolve():
        raise ValueError("Refusing to overwrite the original emulator")
    data = bytearray(source.read_bytes())
    if hashlib.sha256(data).hexdigest() != EMULATOR_SHA256:
        raise ValueError("Expected the official Linux ARM64 Emulator 26.0.0.402")
    if data[MAIN:MAIN + 16].hex() != "fd7bb8a9fd030091f35301a9a02f00b9":
        raise ValueError("Unexpected main entry instructions")
    if data[RUN_QEMU_MAIN:RUN_QEMU_MAIN + 16].hex() != "fd7bbea9fd030091a01f00b9a10b00f9":
        raise ValueError("Unexpected embedded QEMU entry instructions")
    # Both virtual addresses equal file offsets in this pinned executable.
    if mode == "qemu":
        branch = 0x14000000 | (((RUN_QEMU_MAIN - MAIN) // 4) & 0x3FFFFFF)
        struct.pack_into("<I", data, MAIN, branch)
    elif mode == "frontend":
        # Preserve all Huawei device and instance setup. Skip only the host KVM
        # availability guard and select the software CPU/accelerator instead.
        if data[0x1055788:0x105578C] != bytes.fromhex("a0020054"):
            raise ValueError("Unexpected KVM guard instruction")
        if data[0x43C2408:0x43C240D] != b"host\0" or data[0x43C2450:0x43C2454] != b"kvm\0":
            raise ValueError("Unexpected launcher CPU/accelerator constants")
        struct.pack_into("<I", data, 0x1055788, 0x14000015)  # b 0x10557dc
        data[0x43C2408:0x43C240D] = b"max\0\0"
        data[0x43C2450:0x43C2454] = b"tcg\0"
    else:
        raise ValueError(f"Unknown patch mode: {mode}")
    destination.write_bytes(data)
    destination.chmod(source.stat().st_mode)
    print(f"Patched copy: {destination}; SHA-256: {hashlib.sha256(data).hexdigest()}")
    return destination


if __name__ == "__main__":
    patch(Path(sys.argv[1]), Path(sys.argv[2]), sys.argv[3] if len(sys.argv) > 3 else "qemu")
