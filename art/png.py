import zlib, struct

def write_png(path, w, h, px):
    """px: bytearray of w*h*4 RGBA"""
    raw = bytearray()
    for y in range(h):
        raw.append(0)
        raw += px[y*w*4:(y+1)*w*4]
    def chunk(t, data):
        c = t + data
        return struct.pack(">I", len(data)) + c + struct.pack(">I", zlib.crc32(c) & 0xffffffff)
    out = b"\x89PNG\r\n\x1a\n"
    out += chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 6, 0, 0, 0))
    out += chunk(b"IDAT", zlib.compress(bytes(raw), 9))
    out += chunk(b"IEND", b"")
    open(path, "wb").write(out)

class Sheet:
    def __init__(self, raw_path, w, h):
        self.d = open(raw_path, "rb").read(); self.w = w; self.h = h
    def cell(self, cx, cy, size=16):
        """returns list of rows, each row = bytes of size*4"""
        out = []
        for y in range(cy*size, cy*size+size):
            o = (y*self.w + cx*size)*4
            out.append(self.d[o:o+size*4])
        return out

def blank(w, h):
    return bytearray(w*h*4)

def blit(dst, dw, rows, dx, dy, size=16):
    """alpha-aware paste"""
    for j, r in enumerate(rows):
        for i in range(size):
            s = r[i*4:i*4+4]
            if s[3] == 0:
                continue
            o = ((dy+j)*dw + dx+i)*4
            dst[o:o+4] = s
