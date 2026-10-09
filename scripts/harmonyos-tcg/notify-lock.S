/* Exact Emulator 26.0.0.402, linked at unused RX alignment padding 0x789c200.
 * Teleport's five call sites use this in place of virtio_notify.
 * Keep the GIC lock assertion; acquire/release the real BQL only when needed.
 */
.text
.global notify_with_bql
/* Resolve these external symbols with ld --defsym when rebuilding. */
notify_with_bql:
    stp x29, x30, [sp, #-48]!
    mov x29, sp
    str x19, [sp, #16]
    stp x0, x1, [sp, #32]
    bl bql_locked
    mov w19, w0
    cbnz w19, 1f
    adr x0, diagnostic_file
    mov w1, #1
    bl bql_lock_impl
1:
    ldp x0, x1, [sp, #32]
    bl virtio_notify
    cbnz w19, 2f
    bl bql_unlock
2:
    ldr x19, [sp, #16]
    ldp x29, x30, [sp], #48
    ret
diagnostic_file:
    .asciz "cjv-tcg-teleport-notify"
