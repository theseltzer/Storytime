# Step 1 Record + Going Live

Written 2026-08-11. Covers what step 1 built and everything needed to put the site
on a real domain later.

---

## Part 1 — Step 1: the toolchain spike

### Files

| File | What it does | Why it's necessary |
|---|---|---|
| `go.mod` | Module `github.com/theseltzer/storytime`, one dependency: `ebiten/v2 v2.9.9` | Named after the GitHub path now so imports never have to be rewritten later |
| `cmd/game/main.go` | The game: green field, orange square, click-to-ride | Ebiten requires a type with `Update`/`Draw`/`Layout`; this is the smallest thing that proves all three run in a browser |
| `cmd/server/main.go` | `net/http` file server on `:8080` serving `web/` | Loading WASM needs a real HTTP origin — opening `index.html` from disk fails on browser security rules |
| `web/index.html` | Fetches and starts `game.wasm` | The Go program can't bootstrap itself; something in the page must hand it to `WebAssembly` |
| `web/wasm_exec.js` | Copied from the Go install (`$(go env GOROOT)/lib/wasm/wasm_exec.js`) | The bridge between Go's runtime and the browser — Go emits a module that can't talk to the DOM without it |
| `build.sh` | `GOOS=js GOARCH=wasm go build` + copies the shim | The shim must match the Go version that built the `.wasm`; scripting it together makes mismatch impossible |

### Running it

```bash
cd /root/Storytime          # the repo root
./build.sh && go run ./cmd/server
```

Open `http://localhost:8080` in the Windows browser — WSL2 forwards localhost automatically.

Verified: all three assets return 200, `game.wasm` served as `application/wasm`,
**10.5 MB raw / 2.6 MB gzipped**.

### Three decisions worth understanding

**`Update` and `Draw` are separate on purpose.** `Update` runs at a fixed 60 ticks/second
and owns all state changes; `Draw` runs at the monitor's refresh rate and only paints.
Keeping them apart means the game behaves identically on a 60 Hz and a 144 Hz screen.
Never mutate state in `Draw`.

**`Layout` fixes the internal resolution at 800x600** and Ebiten scales that to whatever
the canvas actually is. So spot coordinates always live in an 800x600 space — they don't
change when someone resizes the window. That's what makes step 2 (placing hotspots) simple.

**`Cache-Control: no-store` in the server** exists because browsers cache `.wasm`
aggressively. Without it you'd rebuild, refresh, and stare at the old game wondering why
your change did nothing. Remove it before deploy — in production you *want* caching.

---

## Part 2 — Going live

### Firebase is the wrong tool here

- **Firebase Hosting** is a static-file CDN. It could serve `index.html` and `game.wasm` —
  but it cannot run a Go server. It executes nothing.
- **Firestore / Realtime Database** are NoSQL document stores. No SQL, no joins, no
  schemas, no migrations. The stated goal is learning **PostgreSQL**; adopting Firestore
  deletes the reason this project exists.
- **Firebase has no Postgres at all.** Google's Postgres is Cloud SQL — a separate
  product, ~$9-10/month minimum, no free tier.

Also: no need for "online storage" in the object-storage sense (S3, R2, Firebase Storage).
That's for user-uploaded or bulk files. The sprites and map are a few MB that live in Git
and get served by the binary itself.

### What "alive" actually requires — four separate things

People bundle these into one word, which is why it feels confusing. They're independent:

1. **Domain** — the name. Rented yearly from a registrar.
2. **DNS** — the phone book mapping that name to a server's IP address (an `A` record).
3. **A machine** running the Go binary, reachable on the public internet.
4. **TLS certificate** — so it's `https://`, free via Let's Encrypt.

Plus **5. Postgres**, either on that same machine or hosted elsewhere.

### Domain

Buy from **Cloudflare Registrar** (sells at wholesale cost, no renewal markup) or
**Porkbun**. Avoid GoDaddy — cheap first year, expensive renewal, relentless upsells.
Budget **$10-15/year** for a `.com`; `.dev` is ~$15 and forces HTTPS, a nice signal on a
developer CV.

Independent of everything else — **buy whenever the name is settled.** Nothing else has to
exist first.

### Where to run it

Go gives an advantage most languages don't: `CGO_ENABLED=0 go build` produces **one static
binary with no runtime, no interpreter, no `node_modules`**. Copy that single file to a
server and run it.

| Option | Cost | Trade-off |
|---|---|---|
| **Hetzner VPS (CX22)** | ~EUR 4/mo | A bare Linux box. Most learning, fixed price, you own backups and updates. |
| **Fly.io** | ~$5/mo | Deploys a Docker container, scales to zero. Middle ground. |
| **Railway / Render** | ~$5/mo | Push from GitHub, managed Postgres included. Easiest, least learning. Render's free tier sleeps and cold-starts. |

**Recommendation: Hetzner.** Docker and Linux are already known; the learning style is
mechanism-up; and this is the option where DNS -> TLS -> reverse proxy -> app -> database
is all visible instead of hidden behind a dashboard. Fixed EUR 4 with no surprise bills.

The stack on that box: **Docker Compose** running the Go app + Postgres, with **Caddy** in
front. Caddy is worth knowing — point it at the domain and it obtains and renews the
Let's Encrypt certificate automatically, and gzips responses. `https://` and the 2.6 MB
transfer both come free from that one piece.

**Managed Postgres alternative:** **Neon** has a genuinely usable free tier of real
Postgres. Works with any hosting option above.

### One thing to remember

That 10.5 MB -> 2.6 MB gap is compression. Whatever it deploys on **must** serve the
`.wasm` gzipped or Brotli-compressed, or visitors download four times more than they need.
Caddy and Cloudflare both do it by default — just don't end up somewhere that doesn't.

### When to do this

Don't set up hosting yet — that would mean maintaining a server that serves an orange
square. The natural moment is **after step 5**, when there's real content and a `/cv` page.
Buy the domain any time.

---

## Where we are

Step 1 done and confirmed moving. **Next: step 2 — hotspots.** Ahmet writes it, Claude
guides.

Roadmap:

| # | Step | Status |
|---|------|--------|
| 1 | Toolchain spike — server + click-to-ride square | done |
| 2 | 5-6 hotspots hardcoded in Go, proximity highlight | next, Ahmet writes |
| 3 | HTML overlay story panels (the JS glue) | |
| 4 | Postgres `spots` table + migrations + `/api/spots` | |
| 5 | `/cv` server-rendered route from the same rows | |
| 6 | Kenney art, real trail map, touch input | |
| 7 | Loading screen, compression, deploy | |
