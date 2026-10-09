python
import gdb
frame = gdb.newest_frame()
index = 0
while frame is not None and index < 16:
    frame.select()
    print("[TCG diagnostic] frame", index, frame.name(), hex(frame.pc()))
    for reg in ["x0", "x1", "x2", "x3", "x19", "x20", "x21", "x22", "x23", "x24", "x30"]:
        try:
            value = int(frame.read_register(reg))
            print(reg, hex(value))
            if value > 0x10000:
                data = bytes(gdb.selected_inferior().read_memory(value, 96)).split(b"\0", 1)[0]
                if data and all(32 <= ch < 127 for ch in data):
                    print("  string:", data.decode())
        except (gdb.error, ValueError):
            pass
    try:
        import struct
        fp = int(frame.read_register("x29"))
        words = struct.unpack("<16Q", bytes(gdb.selected_inferior().read_memory(fp, 128)))
        print("frame memory", hex(fp), [hex(value) for value in words])
        for value in words:
            try:
                data = bytes(gdb.selected_inferior().read_memory(value, 128)).split(b"\0", 1)[0]
                if data and all(32 <= ch < 127 for ch in data):
                    print("  stack string:", data.decode())
            except gdb.error:
                pass
    except (gdb.error, ValueError):
        pass
    frame = frame.older()
    index += 1
end
