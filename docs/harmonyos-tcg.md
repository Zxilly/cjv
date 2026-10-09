# Experimental official HarmonyOS ARM64 PC guest under TCG

This experiment uses Huawei's original Linux ARM64 emulator and PC image on a
native ARM64 Linux runner without `/dev/kvm`. It is limited to one exact binary;
the patcher rejects other versions and never overwrites its input.

**Current result: not ready for CI.** Live SSH diagnosis identified a missing
64-bit atomic fast path and a broken surfaceless EGL configuration. One virtual
CPU and an explicit Pbuffer configuration remove those two observed failures,
but a fresh instance still fails the 20-minute HDC readiness gate. Guest services
exceed startup deadlines and restart; a usable HDC shell has not been verified.

## Pinned inputs

- CLI archive: `commandline-tools-linux-arm64-26.0.0.851.zip`
- Archive SHA-256: `c36fdba852ce02354ffcc1ad5f3ab4bdeac917403fe9bbbd60c499a4416246a0`
- Emulator version: `26.0.0.402`, ELF AArch64
- Emulator SHA-256: `62dd21c38f6e21e242f66f6c17753dfa445dd4073e072d851d4af41f8ff9ab4e`
- Guest image: official `HarmonyOS 6.1.1(24)`, `pc_all_arm`
- Frontend patch SHA-256: `d8e5fa4e85b3715351c8932f4c7ce9a78f1cd4cb2ef6330137d56df97016042d`

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

The headless frontend copy also replaces the redundant `EGL_COLOR_BUFFER_TYPE,
EGL_RGB_BUFFER` pair at `0x2239410` with `EGL_SURFACE_TYPE, EGL_PBUFFER_BIT`.
The original attribute list implicitly requires `EGL_WINDOW_BIT`, which cannot
match Mesa's surfaceless configurations. GDB verified `eglChooseConfig` returns
success with zero configurations before the change and one configuration after
it. The fixed emulator initializes llvmpipe and reaches `native windows inited`.
This patch is specifically for the headless Linux experiment.

## Reproduce

Run `.github/workflows/harmonyos-tcg.yml` on the experiment branch. It downloads
and verifies the pinned CLI, installs/caches the official PC image, executes the
lock wrapper regression test, boots the patched copy, and uploads diagnostics.
TCG cold boot uses one virtual CPU and has a 20-minute readiness window.
The original KVM probe keeps its default CPU count.

For interactive diagnosis, manually run `harmonyos-tcg-debug.yml` with an SSH
public key and a virtual CPU count. The workflow starts Upterm, publishes the
connection JSON as the `debug-connection` artifact while the job is running,
and limits the whole job to 60 minutes. Only the supplied public key can join;
no private key is uploaded. Normal CI does not start a debug session.

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

- [Live SSH diagnosis](https://github.com/Zxilly/cjv/actions/runs/37889866635):
  `pidstat`, `perf`, and GDB show the four virtual CPUs repeatedly entering
  `cpu_exec_step_atomic` and waking other CPUs. One sample has about 130,000
  voluntary CPU-thread context switches per second and 69.5% system CPU time
  (100% is one host core). A separate five-second strace sample records 321,316
  futex calls. The original kernel's `STXR` at `0xffffffc0100a5d9c` enters this
  path. A breakpoint at `cpu_loop_exit_atomic` identifies the immediate caller
  as `helper_exit_atomic`, rather than `atomic_mmu_lookup`.
- The binary contains `atomic_cmpxchgl_le` but no `atomic_cmpxchgq_le` helper.
  Together with the observed caller and [QEMU 7.1's CONFIG_ATOMIC64 branch](https://github.com/qemu/qemu/blob/v7.1.0/tcg/tcg-op.c#L3182),
  this identifies the missing 64-bit TCG atomic fast path in this build. One
  virtual CPU avoids `CF_PARALLEL`: the observed CPU thread drops to 157 voluntary
  context switches per second, with about 2% system CPU time. Appspawn starts at
  roughly 69 seconds instead of 141 seconds. These are live observations on the
  same restarted instance, not a clean end-to-end benchmark; cached filesystem
  state differs and later startup stages improve less.
- The single-CPU control still has repeated guest `render_service` SIGSEGVs.
  A read-only snapshot of its userdata overlay recovers the crash report:
  `RenderContextGL::SetUpGpuContext` calls `strlen` with address `0x1f02`, the
  `GL_VERSION` token. The guest's `graphic.cfg` restarts foundation, allocator,
  and composer services when render_service restarts. The host's EGL selection
  failure is therefore a separate blocker, beyond TCG execution speed.
- With the EGL fix, the live guest creates real host contexts and reports
  `renderservice.ready.true` at guest time 688 seconds. The original SIGSEGV
  does not recur during the observed interval. However, render_service exits
  with code 0 at 701 seconds, restarts, reports ready again at 1050 seconds,
  and exits with code 0 at 1053 seconds. Each exit also resets foundation.
  Thus an EGL context or a render-service ready event alone is insufficient.
- A read-only userdata snapshot recovers system logs with a RenderService
  watchdog warning (`blocked 5s`), display-composer dependency failures,
  and an audio-service timeout that explicitly says the process will exit.
  This demonstrates guest-side startup deadlines being exceeded. The exact
  code-0 exit path of render_service is not yet confirmed. Extending the outer
  CI deadline does not extend these guest-side deadlines.
- [Fresh-instance validation](https://github.com/Zxilly/cjv/actions/runs/37891970510)
  at `afcda8e639e436fb06fd6a6c455e956180454d31`: the kernel confirms one CPU;
  host EGL/llvmpipe initializes and there is no render-service SIGSEGV. However,
  render_service exits with code 0 at 674 and 1012 seconds. HDC daemon execution
  begins at 930 seconds, is reset, and starts again at 988 seconds. No usable
  HDC connection exists by the 1200-second deadline. Diagnostics are preserved
  as the `harmonyos-arm64-tcg` artifact. This independently reproduces the
  remaining failure without reusing the live-debug instance.

The full patch changes 155 bytes, confined to guarded ranges. The lock wrapper
test passes on the native runner; an independent assembly rebuild matches all
92 embedded bytes. The original executable's checksum remains unchanged.
Increasing the timeout alone does not fix the observed render-service crash
loop. Successful EGL initialization does not establish guest/HDC readiness;
the fresh-instance test still fails that gate.

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
