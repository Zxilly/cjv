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
NOTIFY_STUB = 0x789C200
# Assembled from notify-lock.S, linked at NOTIFY_STUB. Calls resolve to the
# pinned build's BQL helpers and virtio_notify, with ASLR-safe PC-relative BL.
NOTIFY_CODE = bytes.fromhex(
    "fd7bbda9fd030091f30b00f9e00702a9a0dc4296f303002a9300003540010010"
    "21008052a7dc4296e00742a9e9423b9653000035c8dc4296f30b40f9fd7bc3a8"
    "c0035fd6636a762d7463672d74656c65706f72742d6e6f7469667900"
)


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
        if data[0x43C2420:0x43C2448] != b"type=virt,highmem=on,gic-version=3".ljust(40, b"\0"):
            raise ValueError("Unexpected launcher machine constant")
        struct.pack_into("<I", data, 0x1055788, 0x14000015)  # b 0x10557dc
        data[0x43C2408:0x43C240D] = b"max\0\0"
        data[0x43C2450:0x43C2454] = b"tcg\0"
        # This build has no registered software arm-gicv3-its object. Preserve
        # GICv3/highmem but disable ITS to avoid object_new_with_type(NULL).
        data[0x43C2420:0x43C2448] = b"virt,highmem=on,gic-version=3,its=off".ljust(40, b"\0")
        # The Linux surfaceless path omits EGL_SURFACE_TYPE, whose default is
        # EGL_WINDOW_BIT. Mesa's surfaceless backend then returns zero configs.
        # RGB_BUFFER is already the default; use this pair to request a Pbuffer
        # config explicitly for this headless CI copy. Keep the source intact.
        if data[0x2239410:0x2239418] != bytes.fromhex("3f3000008e300000"):
            raise ValueError("Unexpected Linux EGL configuration attributes")
        struct.pack_into("<II", data, 0x2239410, 0x3033, 1)  # EGL_SURFACE_TYPE, EGL_PBUFFER_BIT
        # Teleport input and distribution threads notify without the BQL.
        # Wrap all five notification call sites in that device module.
        notify_calls = {
            0x77AF9C: "8dc7ff97", 0x77C74C: "a1c1ff97",
            0x77F004: "73b7ff97", 0x77F080: "54b7ff97",
            0x7812E8: "baaeff97",
        }
        for address, expected in notify_calls.items():
            if data[address:address + 4] != bytes.fromhex(expected):
                raise ValueError("Unexpected Teleport notification call")
        if any(data[NOTIFY_STUB:NOTIFY_STUB + len(NOTIFY_CODE)]):
            raise ValueError("Expected unused RX segment alignment padding")
        ph = 64 + 2 * 56  # First PT_LOAD: offset=VA=0, flags=R|X.
        if struct.unpack_from("<IIQQQQQQ", data, ph) != (1, 5, 0, 0, 0, 0x789C1BD, 0x789C1BD, 0x10000):
            raise ValueError("Unexpected executable segment layout")
        # Extend only into the zero alignment gap before the next segment at
        # file offset 0x789cb50. Existing addresses and file size stay unchanged.
        end = NOTIFY_STUB + len(NOTIFY_CODE)
        if end >= 0x789CB50:
            raise ValueError("Notification wrapper exceeds alignment padding")
        struct.pack_into("<QQ", data, ph + 32, end, end)
        data[NOTIFY_STUB:end] = NOTIFY_CODE
        for address in notify_calls:
            struct.pack_into("<I", data, address, 0x94000000 | ((NOTIFY_STUB - address) // 4))
    else:
        raise ValueError(f"Unknown patch mode: {mode}")
    destination.write_bytes(data)
    destination.chmod(source.stat().st_mode)
    print(f"Patched copy: {destination}; SHA-256: {hashlib.sha256(data).hexdigest()}")
    return destination


if __name__ == "__main__":
    patch(Path(sys.argv[1]), Path(sys.argv[2]), sys.argv[3] if len(sys.argv) > 3 else "qemu")
