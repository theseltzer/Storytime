# Story_time v2

A CV you walk through.

Someone is lost on a long night street. A frog finds them and starts asking
questions — and the answers lead, piece by piece, through my background. The
person being searched for turns out to be closer than expected.

Anyone who would rather skip the story can take the exit at any moment: the
CV sits one tap away in the corner, the whole way.

<!-- TODO: live URL here once deployed -->
<!-- TODO: screenshot here -->

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
| Walk | `A D` or `← →` · drag anywhere on touch |
| Open a story | `Space` while standing on a spot |

## Layout

```
cmd/game/         the game: input, camera, parallax layers, drawing, the JS bridge
cmd/server/       static files, /api/spots, /cv, /cv/tr
internal/spot/    the Spot type both sides share
internal/skyline/ seeded skyline generator — same seed, same city, in the game and on /cv
sql/              schema and seed — the content itself
templates/        the /cv page (kept out of web/, or it would be downloadable raw)
web/              everything served to the browser
```

## Licence

<!-- TODO: pick one — MIT for the code is the usual choice. The CV content in
     sql/schema.sql is mine and not covered by it. -->
