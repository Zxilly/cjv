# Experimental official HarmonyOS ARM64 PC guest under TCG

This experiment uses Huawei's original Linux ARM64 emulator and PC image on a
native ARM64 Linux runner without `/dev/kvm`. It is limited to one exact binary;
the patcher rejects other versions and never overwrites its input.

**Current result:** the patched emulator executes the official ARM64 kernel and
userspace without the two observed QEMU assertions. A 20-minute cold boot still
does not yield an HDC shell, so this is not a ready ARM64 CI environment.

## Pinned inputs

- CLI archive: `commandline-tools-linux-arm64-26.0.0.851.zip`
- Archive SHA-256: `c36fdba852ce02354ffcc1ad5f3ab4bdeac917403fe9bbbd60c499a4416246a0`
- Emulator version: `26.0.0.402`, ELF AArch64
- Emulator SHA-256: `62dd21c38f6e21e242f66f6c17753dfa445dd4073e072d851d4af41f8ff9ab4e`
- Guest image: official `HarmonyOS 6.1.1(24)`, `pc_all_arm`
- Frontend patch SHA-256: `128b818a3d15bf13e4c8ba1c08da757a42040551fdf05dd78265e7d61f54dfac`

## What the patch does

The original frontend does not accept QEMU options or a `-qemu` pass-through.
Two independent patch modes expose different paths:

1. `qemu`: redirect `main` at RVA `0xc4512c` to `RunQemuMain` at `0xc509a4`.
   This preserves ELF constructors and exposes QEMU 7.1.0's command parser.
   `--version`, `-accel help`, and `-machine help` work. Full guests also need
   Huawei's frontend instance state, so this mode alone is insufficient.
2. `frontend`: retain the frontend, skip its KVM availability rejection, change
   CPU `host` to `max` and accelerator `kvm` to `tcg`, and retain the original
   highmem/GICv3 settings with `its=off`. The build does not register the software
   `arm-gicv3-its` object; its default ITS path aborts before guest execution.

The frontend mode also wraps the five Teleport `virtio_notify` calls with the
original QEMU BQL helpers. Device background threads otherwise inject interrupts
without the lock required by software GICv3. A 92-byte AArch64 wrapper occupies
verified zero alignment padding at `0x789c200`; the first RX segment is extended
within that gap. Existing addresses and file size stay unchanged. The wrapper
preserves already-held locks and acquires/releases the lock only when needed.
It does not remove the GIC assertion or replace Huawei's device implementations.

## Reproduce

Run `.github/workflows/harmonyos-tcg.yml` on the experiment branch. It downloads
and verifies the pinned CLI, installs/caches the official PC image, executes the
lock wrapper regression test, boots the patched copy, and uploads diagnostics.
TCG cold boot has a 20-minute readiness window.

To create a patched copy manually after extracting the official archive:

```sh
python3 scripts/harmonyos-tcg/patch.py \
  /path/to/emulator/Emulator /path/to/emulator/Emulator-tcg frontend
```

Keep the copy beside the original libraries/resources and use the normal
frontend commands and environment from `scripts/harmonyos-emulator/probe.py`.
The patcher needs only Python; the workflow's emulator dependencies are listed
in its install step.

To rebuild the wrapper bytes for inspection:

```sh
as -o /tmp/notify-lock.o scripts/harmonyos-tcg/notify-lock.S
ld -Ttext=0x789c200 -e notify_with_bql \
  --defsym=bql_locked=0x953490 --defsym=bql_lock_impl=0x9534c0 \
  --defsym=bql_unlock=0x953554 --defsym=virtio_notify=0x76cdd0 \
  -o /tmp/notify-lock.elf /tmp/notify-lock.o
objcopy -O binary -j .text /tmp/notify-lock.elf /tmp/notify-lock.bin
```

Use AArch64 binutils for these commands. The bytes must match `NOTIFY_CODE` in
`patch.py`. `test-notify-lock.c` exercises the assembly with both initial lock
states and verifies notification arguments and final ownership.

## Evidence and limits

- [Raw embedded QEMU entry](https://github.com/Zxilly/cjv/actions/runs/37883646811):
  QEMU 7.1.0, both `tcg` and `kvm` listed.
- [ITS fault identified](https://github.com/Zxilly/cjv/actions/runs/37885340609):
  direct breakpoint at the null-type assertion recovered `arm-gicv3-its`.
- [Real kernel execution](https://github.com/Zxilly/cjv/actions/runs/37885557292):
  Linux 5.10.210 boots, storage enumerates, and Teleport/GPS initialize, followed
  by the missing-BQL assertion.
- [Teleport call chain](https://github.com/Zxilly/cjv/actions/runs/37885877120):
  an AArch64 frame-pointer walk identifies `express_input_device_sync`.
- [All notification paths wrapped](https://github.com/Zxilly/cjv/actions/runs/37886586664):
  no emulator assertion; guest reaches userspace, mounts `/data`, and keeps
  initializing services throughout the five-minute window. HDC is not ready
  before that deadline. This is not yet proof of a usable CI guest.
- [20-minute readiness test](https://github.com/Zxilly/cjv/actions/runs/37887315116)
  at `1fe2ae579677ae8b3790783bac193c9cd15c5948`: no QEMU assertion or kernel reboot;
  userspace continues initializing through guest time 1200 seconds. HDC's data
  directory and USB function setup appear in the log, but no usable HDC shell is
  obtained. At guest time 1193 seconds, the network manager reports a system
  ability initialization time of 749067 ms; ueventd still reports
  `init not complete` at 1196 seconds. The run fails the readiness gate and saves
  its first-boot logs without resetting the guest under the debugger.

The full patch changes 152 bytes, confined to guarded ranges. The lock wrapper
test passes on the native runner; an independent assembly rebuild matches all
92 embedded bytes. The original executable's checksum remains unchanged.
Further guest startup/performance work is still required. The current evidence
does not identify a single cause for the slow userspace initialization, and
does not establish that increasing the timeout again would produce a ready
or stable environment.

The readiness gate requires an HDC shell reporting `aarch64`, successful file
transfer in both directions, and the writable HOME smoke test. It does not
validate the graphical desktop or compiler execution. No Go/compiler code or
published official archives are changed by this experiment.

Public stock-QEMU attempts are not a substitute for Huawei's device model:
[this ARM64 PC experiment](https://hu60.cn/q.php/bbs.topic.106781.html) panicked
in Teleport/GPS initialization and its author later reported no success.
[Upstream QEMU 7.1.0](https://github.com/qemu/qemu/blob/v7.1.0/hw/arm/virt.c)
documents the `its` property in code; the missing registration here is a finding
specific to the pinned Huawei build.
