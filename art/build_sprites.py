#!/usr/bin/env python3
"""Builds web/sprites.png out of the Kenney RPG Urban Pack tilemap.

Run from this folder:   python3 build_sprites.py
Needs ffmpeg, only to turn the source PNG into raw RGBA (no Python image
library is installed here, and pulling one in for one decode was not worth it).

Change CHAR below to pick a different one of the pack's six characters.
"""
import subprocess, os, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

SRC_PNG = "kenney_urban_tilemap.png"
RAW = "kenney_urban_tilemap.raw"
OUT = "../web/sprites.png"

if not os.path.exists(RAW):
    subprocess.run(["ffmpeg", "-y", "-loglevel", "error", "-i", SRC_PNG,
                    "-f", "rawvideo", "-pix_fmt", "rgba", RAW], check=True)

from png import Sheet, write_png, blank

# Source: Kenney "RPG Urban Pack" (CC0), tilemap_packed.png, 16x16 tiles.
# Each character there is 4 columns (left / front / back / right - the right
# one is already the mirror of the left) by 3 rows (idle / walk A / walk B).
#
# The game wants 4 frames across and 8 rows down: rows 0-3 walking
# down/left/right/up, rows 4-7 the same on the bike.
#
# The cell is 20x20 rather than 16x16 because the Kenney art fills its own
# 16x16 cell completely and the bike needs room somewhere: the character is
# pasted at the top (2,0) and the bottom four rows are the bike.

CELL, SRC = 20, 16
OFF_X, OFF_Y = 2, 0
CHAR = 5                       # which of the pack's six characters
BASE = CHAR * 3
DIR_COL   = [24, 23, 26, 25]   # down, left, right, up
FRAME_ROW = [0, 1, 0, 2]       # idle, walk A, idle, walk B - frame 0 is the
                               # standing pose, which is what a motionless avatar shows

TYRE  = (0x1a, 0x1a, 0x1e, 0xff)
FRAME = (0xe8, 0x80, 0x2a, 0xff)   # orange, so the bike reads against grass
BAR   = (0x3a, 0x3a, 0x42, 0xff)
HUB   = (0x8a, 0x8a, 0x92, 0xff)

def bike_side():
    """Seen from the side: two wheels and the frame between them."""
    px = {}
    for x in range(4, 16):
        px[(x, 16)] = FRAME
    for cx in (3, 16):
        for y, w in ((15, 1), (16, 2), (17, 2), (18, 1)):
            for x in range(cx - w + 1, cx + w):
                px[(x, y)] = TYRE
        px[(cx, 16)] = HUB
    return px

def bike_axial():
    """Seen end-on: a handlebar across and one wheel edge-on beneath it."""
    px = {}
    for x in range(3, 17):
        px[(x, 15)] = BAR
    px[(3, 16)] = BAR
    px[(16, 16)] = BAR
    for y in range(15, 19):
        px[(9, y)] = TYRE
        px[(10, y)] = TYRE
    px[(9, 17)] = HUB
    px[(10, 17)] = HUB
    return px

BIKE = [bike_axial(), bike_side(), bike_side(), bike_axial()]

sh = Sheet(RAW, 432, 288)
W, H = 4 * CELL, 8 * CELL
img = blank(W, H)

def put(x, y, c):
    if 0 <= x < W and 0 <= y < H:
        o = (y * W + x) * 4
        img[o:o+4] = bytes(c)

for d in range(4):
    for f, src in enumerate(FRAME_ROW):
        cell = sh.cell(DIR_COL[d], BASE + src)
        for on_bike in (False, True):
            ox, oy = f * CELL, (d + (4 if on_bike else 0)) * CELL
            if on_bike:
                for (x, y), c in BIKE[d].items():
                    put(ox + x, oy + y, c)
            for j in range(SRC):
                for i in range(SRC):
                    p = cell[j][i*4:i*4+4]
                    if p[3] == 0:
                        continue
                    put(ox + OFF_X + i, oy + OFF_Y + j, p)

write_png(OUT, W, H, img)
print("wrote", OUT, W, "x", H)
