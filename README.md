# Story_time

A CV you ride through.

Instead of a page you scroll, this is a small top-down world you move around in.
Ride up to a spot, press a key, and that part of my background opens in a panel.
The same content is also served as a plain HTML page at `/cv`, so recruiters,
crawlers and phones on a bad connection get the boring version instantly.

Built with Go and PostgreSQL, deliberately without a frontend framework.

The CV page is served in both languages: `/cv` (English) and `/cv/tr` (Turkish).

<!-- TODO: live URL here once deployed -->
<!-- TODO: screenshot of the world here -->

## The idea worth explaining

One content store, two renderers.

```
                    ┌──────────────┐
                    │  PostgreSQL  │   one row per spot,
                    │    spots     │   both languages on the row
                    └──────┬───────┘
                           │
                  ┌────────┴────────┐
                  │   Go HTTP srv   │
                  └────┬───────┬────┘
                       │       │
          /api/spots   │       │   /cv  and  /cv/tr
             (JSON)    │       │   (server-rendered HTML)
                       ▼       ▼
              ┌────────────┐  ┌──────────────┐
              │ WASM game  │  │ plain page   │
              │  (canvas)  │  │ no JS at all │
              └────────────┘  └──────────────┘
```

A canvas is invisible to search engines, to ATS parsers, and to anyone who
closes a page that takes too long to load. So the game is not allowed to be the
only way in. Both views read the same rows, which is what makes the database
earn its place instead of being decoration: editing a row changes the game and
the CV page at once, with no rebuild.

## Decisions I would defend in an interview

**Movement: direction vectors, not click-to-move.** The first version let you
click a spot and the avatar rode there. Then walls entered the design, and
"ride there" suddenly meant A\* pathfinding — a large chunk of work to build and
tune. Switching to direct control (WASD/arrows, or an invisible on-screen
joystick on touch) made collision about fifteen lines: test the four corners of
the box, test each axis separately, and wall-sliding falls out for free. I
removed an entire algorithm by changing a requirement instead of implementing it.

**The joystick has a dead zone.** A thumb resting on a phone screen moves one or
two pixels. Without a dead zone the avatar twitches while you are not touching
anything, which reads as a bug even though every line is correct.

**The map is fetched, not compiled in.** `web/map.txt` is one character per
tile. It arrives over HTTP like any other data, so changing the world is editing
a text file and refreshing — no rebuild. Collision reads the same characters, so
the picture and the physics can never disagree.

**A goroutine cannot return an error.** Three things load over the network:
spots, map, art. Each can fail three ways — the request, a non-200, a decode
error — and none of it may block the game loop. Each fetch runs in a goroutine
and sends one value that carries either the data or the reason there is none, so
the update loop stays the only writer of game state and there is no shared
memory to race over.

**Two languages are a closed set.** Instead of a `lang` column and a query per
language, each row carries both. The API ships both, so switching language is a
re-render with no refetch, and `/cv` and `/cv/tr` are two URLs rather than
content negotiation — a crawler indexes a URL, not a header, and a link you send
someone has to open the same page for them as it did for you.

## Stack

| | |
|---|---|
| Game | Go + [Ebitengine](https://ebitengine.org/), compiled to WebAssembly |
| Server | Go standard library (`net/http`, `html/template`) |
| Database | PostgreSQL, via `database/sql` + [pgx](https://github.com/jackc/pgx) |
| Frontend | ~30 lines of hand-written JS bridging the canvas and the DOM |
| Art | [Kenney RPG Urban Pack](https://kenney.nl/assets/rpg-urban-pack) (CC0) |

Go 1.26. The only direct dependency is Ebitengine; pgx comes in behind
`database/sql` so the API knowledge transfers to any other database.

## Running it locally

Needs Go 1.26+ and a PostgreSQL server.

```bash
# 1. database
createdb storytime
psql "$DATABASE_URL" -f sql/schema.sql

# 2. connection string
export DATABASE_URL="postgres://user:password@localhost:5432/storytime"

# 3. build the game to WebAssembly
./build.sh

# 4. run the server
go run ./cmd/server
```

Then open <http://localhost:8080>.

`build.sh` compiles `cmd/game` to `web/game.wasm` and copies the matching
`wasm_exec.js` out of your Go installation. Both are build output and are not
committed, so **a fresh clone has to run `build.sh` before the page will work.**

### Controls

| | |
|---|---|
| Move | `W A S D` or arrow keys · drag anywhere on touch |
| Open a story | `Space` while standing on a spot |
| Get off the bike | `B` |

## Layout

```
cmd/game/      the game: input, collision, camera, drawing, the JS bridge
cmd/server/    static files, /api/spots, /cv, /cv/tr
internal/spot/ the Spot type both sides share
sql/           schema and seed — the content itself
templates/     the /cv page (kept out of web/, or it would be downloadable raw)
web/           page, map, art — everything served to the browser
art/           Kenney source sheet + the script that cuts the avatar out of it
```

## Credits

Art from [Kenney](https://kenney.nl) (CC0). Licence text in
`art/kenney_LICENSE.txt`.

## Licence

<!-- TODO: pick one — MIT for the code is the usual choice. The CV content in
     sql/schema.sql is mine and not covered by it. -->
