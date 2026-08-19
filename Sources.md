# Sources — Landing Page Inspiration

Reference list for the game-like personal-branding site (Go + PostgreSQL).
Collected 2026-08-10.

## The canonical "portfolio as a game" sites

Look at *what interaction carries the story*, not the polish.

- **bruno-simon.com** — the reference point. You drive a little car around a 3D world;
  his projects are physical objects in it. Extremely ambitious (Three.js + physics).
- **henryheffernan.com** — a 90s CRT computer you boot up and click around. Nostalgia +
  a familiar UI metaphor doing the storytelling.
- **jesse-zhou.com** — a hand-modeled Hong Kong ramen shop you explore. Proof that *one
  small scene* beats a whole open world.
- **neal.fun** — not a portfolio, but the best source alive for "what makes a web page
  feel like a toy." Steal mechanics, not visuals.
- **lynnandtonic.com** — redesigns her site yearly, each one a gimmick executed
  perfectly. Great for scoping ideas *down* to something buildable.

## Craft-level, less heavy

Playful without a 3D engine — closer to what's realistically shippable first.

- **joshwcomeau.com** — the gold standard for "whimsy in ordinary components," and he
  *writes up how he did it*.
- **cassie.codes** — SVG + GSAP animation, illustrative and character-driven.
- **rauno.me**, **emilkowal.ski**, **paco.me** — interaction detail craft. Study the
  micro-feel: hover, focus, transitions.
- **jhey.dev** — CSS-only tricks that look impossible.

## Game UI specifically (the one people miss)

For *game-like*, copy the chrome real games use — inventory screens, skill trees,
dialogue boxes, character sheets.

- **gameuidatabase.com** — thousands of real screenshots from real games, filterable by
  screen type. Best single resource on this list for this goal.
- **interfaceingame.com** — same idea, curated and prettier.
- **kenney.nl** — free CC0 game art/UI packs. Solves "I have no designer" instantly.
- **lospec.com/palette-list** — if going pixel/retro, palettes done right.

## Galleries to browse for volume

- **awwwards.com** (filter Portfolio)
- **thefwa.com**
- **godly.website**
- **land-book.com**
- **onepagelove.com**
- **tympanus.net/codrops** — Playground/Demos section, plus tutorials that hand you the code.

## Creators for technique

- **Bruno Simon — Three.js Journey** (threejs-journey.com), if going 3D.
- **SimonDev** (YouTube) — 3D/game math explained from the mechanism up.
- **Yuri Artiukh / akella** (YouTube) — live WebGL shader coding.
- **gsap.com/showcase** — animation reference.

---

## Two things to settle before collecting screenshots

**The stack pulls against most of this.** Go + Postgres is the server; every site above
is client-side JS/WebGL. Fine — but decide early *where the game lives*. Three broad
options, roughly cheapest to most expensive:

- **(a)** Go templates + CSS/small JS, game-*feel* via UI metaphor (character sheet,
  quest log, skill tree).
- **(b)** Go serving a canvas/2D game where Postgres holds real state (visitor progress,
  unlocked sections, guestbook/leaderboard).
- **(c)** Full 3D scene with Go as a thin API.

Option **(b)** is where Postgres genuinely earns its place instead of being decoration.

**The metaphor matters more than the engine.** The two-track background — wet
lab/genomics *and* backend — is an unusual story, and a well-chosen metaphor tells it for
free: a skill tree with two branches that converge, a lab bench you interact with, an
overworld map with regions per career phase.

---

## Next step

Browse, come back with 3–5 favorites and *why*. Then: turn that into a concrete
landing-page plan and build it one step at a time.

## Art assets

- **Kenney RPG Urban Pack** (https://kenney.nl/assets/rpg-urban-pack) — CC0, no
  attribution required. Source tilesheet kept at `art/kenney_urban_tilemap.png`
  (432x288, 16x16 tiles) with the pack's licence at `art/kenney_LICENSE.txt`.
  `art/build_sprites.py` cuts the avatar sheet out of it into `web/sprites.png`.
