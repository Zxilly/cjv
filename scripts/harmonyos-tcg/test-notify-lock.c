/* Execute the actual AArch64 wrapper against both BQL ownership states. */
#include <assert.h>
#include <stdbool.h>
#include <stdio.h>
#include <string.h>

static bool held;
static int locks, unlocks, notifications;
static int device, queue;

bool bql_locked(void) { return held; }
void bql_lock_impl(const char *file, int line) {
    assert(!held && !strcmp(file, "cjv-tcg-teleport-notify") && line == 1);
    held = true;
    locks++;
}
void bql_unlock(void) {
    assert(held);
    held = false;
    unlocks++;
}
void virtio_notify(void *dev, void *vq) {
    assert(held && dev == &device && vq == &queue);
    notifications++;
}
extern void notify_with_bql(void *, void *);

int main(void) {
    notify_with_bql(&device, &queue);
    assert(!held && locks == 1 && unlocks == 1 && notifications == 1);
    held = true;
    notify_with_bql(&device, &queue);
    assert(held && locks == 1 && unlocks == 1 && notifications == 2);
    puts("Both BQL ownership paths preserve arguments and lock ownership");
}
