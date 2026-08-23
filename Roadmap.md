# Roadmap — the side-scrolling street

Written 2026-08-23, when the top-down tile world was retired.

Kept in English to match `README.md` and `Deployment.md`. If this should stay
private, add it to `.gitignore` — nothing else references it.

---

## The concept, and why it changed

The first world was top-down: a 25x18 tile map, walls, rooms, doors, free
movement on both axes. It was abandoned for three reasons, all of them design
rather than technical:

- The world was too large to cross at walking speed.
- Free movement on two axes means the visitor can get lost, and a visitor who is
  lost is a visitor who leaves.
- A CV has an order. A world without one throws that away.

A 3D version (three.js, in the spirit of bruno-simon.com) was costed and
shelved — not on money, which is near zero, but on time: three to six months
part-time before anything is presentable.

**What replaced it: the Kingdom Two Crowns model.**

| Rule | Consequence here |
|---|---|
| One axis of movement | The street *is* the timeline. 2015 at one end, today at the other |
| No jump, no crouch, no skill | A recruiter can look without learning controls |
| The world reacts to proximity | Interaction needs no tutorial |
| Depth comes from parallax, not from geometry | Cheap to build, cheap to download |
| Almost no HUD | The content is the interface |

Dropping the vertical axis also dropped gravity, jumping, crouching and
collision. That is the whole reason this plan is short.

---

## Phases

| Phase | Theme | Status |
|---|---|---|
| **B0** | The street — collapse the two-axis world to one | **done** |
| **B1** | Parallax layers | next |
| **B2** | Buildings as experiences | |
| **B3** | Day/night and atmosphere | |
| **B4** | The story | |

---

## B0 — the street (done)

**Goal:** the avatar stands on a street, walks left and right, the camera
follows horizontally. Two flat colour bands. Deliberately ugly.

### Removed

| Where | What |
|---|---|
| `const` | `tileSize`, `srcTileSize`, `tileScale`, `wallSideHeight` |
| `var` | `tileSrc`, `colorGrass`, `colorWall`, `colorDoor` |
| `Game` | `tiles`, `mapCh`, `tileImg`, `tilesetCh`, `wallSide` |
| types | `mapSonuc` |
| `Update` | the `mapCh` and `tilesetCh` cases |
| functions | `placeBike`, `blocked`, `wallAt`, `fits`, `normalize` |
| `Draw` | the tile loop and the wall side-face pass |
| `main` | two channels, the `/map.txt` fetch, `fetchImage("/tiles.png", …)` |
| files | `web/map.txt`, `web/tiles.png` |

The Kenney source sheet stays at `art/kenney_urban_tilemap.png`. It is an
*urban* tilemap, so it comes back in B2 for the buildings.

### Added

| Name | What it is |
|---|---|
| `groundY` | The world Y the avatar rides at. Feet land `bikeSize/2` below it |
| `groundDepth` | How much road stays visible below the feet |
| `worldWidth` | Length of the street. Placeholder until the spot X values are re-authored |
| `startX` | Where the avatar begins. Replaces the map's `S` tile |
| `colorSky`, `colorGround` | The two bands |
| `groundAt(x)` | Returns `groundY`. A function so a sloped street later changes one place |

### Two decisions worth remembering

**The camera is anchored to the bottom of the view, not the top of the world.**

```go
camY := groundY + bikeSize/2 + groundDepth - viewH
```

If `camY` were fixed at 0, a short browser window would show sky and no street.
Deriving it from `viewH` keeps the road the same distance from the bottom edge
at every window height. A tall window shows more sky, never more road.

**Y is a result, not an input.** `Update` overwrites `bikeY` with
`groundAt(bikeX)` every frame. With no jump there is nothing that could lift the
avatar off the ground, so storing a vertical velocity would be storing a number
that is always zero.

---

## B1 — parallax layers (next)

What makes a flat 2D scene look rich, in order of how much each contributes:

1. **Five to eight layers.** Distant ones scroll slowly, near ones quickly. This
   is most of the effect on its own.
2. **Atmospheric perspective.** Distant layers are paler, less saturated and
   shifted towards blue. The retired wall side-face already used this idea, at
   `ColorScale.Scale(0.72, 0.72, 0.76, 1)`.
3. **A gradient sky** that changes with time of day.
4. **A pure silhouette foreground.** One flat colour, no detail. It is the
   cheapest depth cue there is.
5. **Particles** — dust, litter, a bird. Few, always moving.
6. **Camera feel.** Damping, look-ahead in the direction of travel, a slight
   zoom-out at speed.
7. **Five to seven colours total.** More reads as cheap.

The layers do not need art files. `ebiten/v2/vector` has `Path` with `MoveTo`,
`LineTo`, `QuadTo`, `CubicTo` and `Close`, plus `FillPath` — a skyline is a list
of points filled with one flat colour. No PNG, no download, sharp at any
resolution.

---

## B2 — buildings as experiences

Each row in the `spots` table becomes a building along the street.

**Interaction splits in two**, deliberately:

| Proximity, no key | Explicit key or tap |
|---|---|
| The building's windows light up | The story panel opens |
| Its sign appears | |
| Its colour warms | |

Kingdom has no interaction key at all, but a panel that opens by itself would
hijack the walk and flash past anyone crossing the street. Proximity keeps the
"the world notices you" feeling; the panel stays under the visitor's control.

**Content work this phase needs:** the spot `X` values in Postgres are still
top-down coordinates in the 0–800 range, and `Y` is meaningless on a street.
They have to be re-authored as distances along the street, in chronological
order. That is a data edit, not a code change — which is the point of having
kept the content in the database.

---

## B3 — day/night and atmosphere

Time of day drives one colour-interpolation function; every layer reads its
tint from it. Sunset over the walk is the single strongest atmosphere lever
available, and it costs one lerp per layer per frame.

---

## B4 — the story

Open. The street is a timeline, so the narrative shape is already there; what is
undecided is whose voice tells it and how much of it is diegetic.

---

## Visual references

| Game | What to take from it |
|---|---|
| **Kingdom Two Crowns** | The whole mechanic. One axis, proximity, no HUD |
| **Alto's Odyssey** | Silhouettes, gradient skies, weather, day/night |
| **Old Man's Journey** | Layered cut-paper hills, warm palette, the feeling of a journey |
| **Limbo / INSIDE** | Silhouette and fog: expensive-looking, cheap to draw |
| **Badland** | Black foreground against a rich, soft background |
| **Thomas Was Alone** | Proof that plain rectangles plus light, camera and narrative are enough |
| **Rayman Origins** | Vector 2D at its best — curves and flat colour, no pixels |

---

## What survived the rewrite

The Go HTTP server, `/api/spots`, `/cv`, the Postgres schema, the two-language
content model, the `showStory` / `closeStory` bridge, the loading and error
indicators, the async fetch pattern, and the deploy setup. None of it was
touched.

What was retired was one client. That is the argument for having kept the
content out of the Go source from the first day: the world changed shape
completely and the content did not have to move.
