package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"

	// Blank import: image.Decode only knows the formats whose decoders have
	// registered themselves. Without this a perfectly good PNG fails with
	// "unknown format".
	_ "image/png"

	"io"
	"log"
	"math"
	"net/http"
	"strings"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/theseltzer/storytime/internal/spot"
)

const (
	// Desktop window size only. The browser canvas decides its own size, and
	// Layout hands that straight through - see viewW/viewH.
	screenWidth  = 800
	screenHeight = 576

	bikeSize  = 24
	walkSpeed = 2.2
	bikeSpeed = 3.4

	// One cell of web/sprites.png. The sheet is 4 frames wide and 8 rows tall:
	// rows 0-3 walking (down, left, right, up), rows 4-7 the same on the bike.
	//
	// The cell is 20px but the character art only fills the top 16 of them;
	// the bottom 4 hold the bike. So spriteFoot, not spriteSize, is where the
	// feet are - anchoring by the cell bottom would float the avatar 4px up.
	spriteSize        = 20
	spriteFoot        = 16
	animTicksPerFrame = 8

	// Two separate knobs, because "bigger character" and "less world on
	// screen" are different questions.
	//
	// spriteScale enlarges only the avatar, so it grows relative to the
	// buildings. Kept a whole number: pixel art scaled by 2 stays crisp,
	// scaled by 1.6 it does not.
	//
	// zoom enlarges everything, which is the same as showing less world:
	// at 1.25 the visible width and height are each 80% of before.
	spriteScale = 2.0
	zoom        = 1.25

	// How far the thumb must travel from where it landed before the invisible
	// joystick reports a direction. Without it, a resting finger twitches the
	// avatar.
	deadZone = 12.0

	// One tile is 32px, so web/map.txt at 25x18 covers exactly 800x576.
	tileSize = 32

	// The Kenney tileset is drawn at 16px, so every tile is blown up 2x on its
	// way to the screen. Two constants rather than a bare 2, because 16 is a
	// fact about the source art and 32 is a fact about the world.
	srcTileSize = 16
	tileScale   = tileSize / srcTileSize
)

// Facing directions, in the row order of the sprite sheet.
const (
	faceDown = iota
	faceLeft
	faceRight
	faceUp
)

var (
	colorGrass    = color.RGBA{0x3a, 0x6b, 0x35, 0xff}
	colorBike     = color.RGBA{0xe8, 0x9c, 0x2a, 0xff}
	colorSpot     = color.RGBA{0x2d, 0x54, 0x29, 0xff} // Dark green when away
	colorSpotNear = color.RGBA{0xf1, 0xc4, 0x0f, 0xff} // bright yellow when close
	colorWall     = color.RGBA{0x4a, 0x4a, 0x4a, 0xff} // '#'
	colorDoor     = color.RGBA{0x8a, 0x6a, 0x3a, 0xff} // '_'
)

// Where each map character's picture sits in web/tiles.png, counted in source
// tiles of 16px. The whole Kenney sheet ships as-is rather than a trimmed one,
// so a new tile type later is a line here and nothing else - no image to
// rebuild, no coordinates to renumber.
var tileSrc = map[byte][2]int{
	'.': {1, 1},  // grass, outdoors
	'S': {1, 1},  // the start tile is only a marker; it looks like grass
	',': {9, 4},  // indoor floor, so a building is not a walled lawn
	'#': {18, 2}, // red brick
	'_': {1, 4},  // the doorway, a different paving from both sides of it
}

// Game holds everything that changes over time. Ebiten calls Update and Draw
// on this one value forever, so all game state lives here.
type Game struct {
	bikeX, bikeY float64
	spots        []spot.Spot
	activeSpot   int
	spotsCh      chan fetchSonuc

	// One row per line of web/map.txt, indexed tiles[y][x]. Nil until the
	// fetch lands, which is safe: ranging over a nil slice runs zero times.
	tiles  []string
	mapCh  chan mapSonuc
	loaded bool
	failed bool

	// The size of the window the world is being watched through, in world
	// pixels. Set by Layout every frame; the world itself is far larger.
	viewW, viewH int

	sprites   *ebiten.Image
	spritesCh chan imageSonuc

	// The terrain is kept as one ready-to-draw image per map character rather
	// than as the whole sheet: see the tileset case in Update for why.
	tileImg   map[byte]*ebiten.Image
	tilesetCh chan imageSonuc
	facing    int
	animTick  int
	animFrame int
	onBike    bool

	// Invisible joystick. The anchor is wherever the finger landed rather than
	// a fixed point on screen, so the stick is always under the thumb.
	touchID                    ebiten.TouchID
	touchActive                bool
	touchOriginX, touchOriginY float64
	touchIDs                   []ebiten.TouchID // reused, see joystick
}

// fetchSonuc carries both halves of one fetch attempt: the data, or the reason
// there is none. Packing them into a single value means a single send on a
// single channel, so Update stays the only writer of g.spots.
type fetchSonuc struct {
	spots []spot.Spot
	err   error
}

// mapSonuc is the same idea for the terrain. A separate type and a separate
// channel because the two fetches are independent: either can land first, and
// nothing may wait on the other.
type mapSonuc struct {
	tiles []string
	err   error
}

// imageSonuc carries a decoded image.Image, not an *ebiten.Image. Decoding is
// plain Go and safe anywhere, but turning the result into a GPU-backed Ebiten
// image is the game loop's job, so that conversion happens in Update.
//
// One type for both PNGs: the avatar sheet and the tileset differ only in what
// Update does with the result.
type imageSonuc struct {
	img image.Image
	err error
}

// Update runs 60 times a second and advances the world by one tick.
// It handles input and movement, never drawing.
func (g *Game) Update() error {

	select {
	case r := <-g.spotsCh:
		if r.err != nil {
			// The console gets the technical detail, the visitor gets a
			// sentence they can actually read.
			log.Printf("fetch spots: %v", r.err)
			g.failed = true
			js.Global().Call("showError", "spots_failed")
		} else {
			g.spots = r.spots
		}

	case m := <-g.mapCh:
		if m.err != nil {
			log.Printf("fetch map: %v", m.err)
			g.failed = true
			js.Global().Call("showError", "map_failed")
		} else {
			g.tiles = m.tiles
			g.placeBike()
		}

	case sp := <-g.spritesCh:
		if sp.err != nil {
			// Not fatal: Draw falls back to a plain rectangle, so the game is
			// still playable without art.
			log.Printf("fetch sprites: %v", sp.err)
		} else {
			g.sprites = ebiten.NewImageFromImage(sp.img)
		}

	case t := <-g.tilesetCh:
		if t.err != nil {
			// Not fatal either: Draw falls back to flat colours.
			log.Printf("fetch tileset: %v", t.err)
		} else {
			sheet := ebiten.NewImageFromImage(t.img)

			// Every tile is cut once, here, rather than by calling SubImage
			// inside Draw. Draw runs 60 times a second over roughly 500 visible
			// tiles and SubImage allocates a new image on every call, so doing
			// it there would be 30000 pointless allocations a second.
			g.tileImg = make(map[byte]*ebiten.Image, len(tileSrc))
			for ch, p := range tileSrc {
				g.tileImg[ch] = sheet.SubImage(image.Rect(
					p[0]*srcTileSize, p[1]*srcTileSize,
					(p[0]+1)*srcTileSize, (p[1]+1)*srcTileSize,
				)).(*ebiten.Image)
			}
		}

	default:
	}

	// Two independent loads share one indicator, so it can only come down once
	// both have landed. The loaded flag is not a copy of what the DOM already
	// knows - it is an edge trigger, so the bridge is crossed once instead of
	// 60 times a second.
	if !g.loaded && !g.failed && g.spots != nil && g.tiles != nil {
		g.loaded = true
		js.Global().Call("hideStatus")
	}

	// All input collapses into one direction vector, so the movement below
	// never learns which device produced it. While the story panel is open the
	// vector stays zero and the bike simply stops.
	var dx, dy float64
	if g.activeSpot == -1 {
		dx, dy = g.input()
	}

	// B swaps the avatar. Nothing else about the mechanics changes - same
	// collision box, same controls, only the sprite and a little more speed.
	if inpututil.IsKeyJustPressed(ebiten.KeyB) && g.activeSpot == -1 {
		g.onBike = !g.onBike
	}

	speed := walkSpeed
	if g.onBike {
		speed = bikeSpeed
	}

	// The two axes are tested separately on purpose. Riding into a wall at an
	// angle then keeps the component that is still free, so the avatar slides
	// along the wall instead of stopping dead against it.
	if nx := g.bikeX + dx*speed; g.fits(nx, g.bikeY) {
		g.bikeX = nx
	}
	if ny := g.bikeY + dy*speed; g.fits(g.bikeX, ny) {
		g.bikeY = ny
	}

	// Four-direction art from a free-angle vector: whichever axis is larger
	// decides which way the avatar is turned.
	if dx != 0 || dy != 0 {
		if math.Abs(dx) > math.Abs(dy) {
			g.facing = faceLeft
			if dx > 0 {
				g.facing = faceRight
			}
		} else {
			g.facing = faceUp
			if dy > 0 {
				g.facing = faceDown
			}
		}

		g.animTick++
		g.animFrame = (g.animTick / animTicksPerFrame) % 4
	} else {
		// Standing still shows frame 0 but keeps the last facing, otherwise the
		// avatar would snap to face south every time it stopped.
		g.animFrame = 0
	}
	// Proximity check for each spot
	for i := range g.spots {
		dist := math.Hypot(g.bikeX-g.spots[i].X, g.bikeY-g.spots[i].Y)
		g.spots[i].IsNear = dist <= g.spots[i].Radius
	}

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) && g.activeSpot == -1 {
		for i := range g.spots {
			if g.spots[i].IsNear {
				g.activeSpot = i

				// Both languages go across at once and JS picks. That keeps the
				// toggle instant - flipping it re-renders an already open panel
				// without asking Go, or the server, for anything.
				js.Global().Call("showStory",
					g.spots[i].TitleEN, g.spots[i].BodyEN,
					g.spots[i].TitleTR, g.spots[i].BodyTR)

				// prevents collision in matching
				break
			}
		}
	}

	return nil

}

// placeBike moves the bike onto the tile marked S in the map. Keeping the start
// position in the map file instead of in Go means moving it is a text edit
// rather than a rebuild, the same rule the rest of the content follows.
func (g *Game) placeBike() {
	for y, row := range g.tiles {
		if x := strings.IndexByte(row, 'S'); x >= 0 {
			g.bikeX = float64(x*tileSize + tileSize/2)
			g.bikeY = float64(y*tileSize + tileSize/2)
			return
		}
	}

	log.Print("map has no S tile; bike stays at 0,0")
}

// camera returns the world coordinate that is drawn at the top-left of the
// screen. The bike is centred, then the view is clamped so it never slides past
// the edge of the map and shows blank space.
func (g *Game) camera() (float64, float64) {
	if len(g.tiles) == 0 {
		return 0, 0
	}

	worldW := float64(len(g.tiles[0]) * tileSize)
	worldH := float64(len(g.tiles) * tileSize)
	viewW, viewH := g.view()

	// The outer max guards a view larger than the world, where the two clamp
	// bounds would otherwise cross over.
	return min(max(g.bikeX-viewW/2, 0), max(worldW-viewW, 0)),
		min(max(g.bikeY-viewH/2, 0), max(worldH-viewH, 0))
}

// input returns this frame's direction, as a vector of length 0 or 1.
// Keyboard wins when both are live; AppendTouchIDs is always empty on desktop,
// so the touch half costs nothing there.
func (g *Game) input() (float64, float64) {
	var dx, dy float64

	// IsKeyPressed, not IsKeyJustPressed: this asks "is the key down right
	// now", which is what holding a direction means. Adding instead of
	// choosing makes W+S cancel to zero on its own.
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		dy--
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		dy++
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		dx--
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		dx++
	}

	if dx == 0 && dy == 0 {
		dx, dy = g.joystick()
	}

	return normalize(dx, dy)
}

// joystick reads the invisible on-screen stick: the first finger to land sets
// the anchor, and the vector from that anchor to where the finger is now is
// the direction.
func (g *Game) joystick() (float64, float64) {
	if !g.touchActive {
		// Slicing to [:0] keeps the capacity, so this does not allocate a new
		// slice sixty times a second.
		g.touchIDs = inpututil.AppendJustPressedTouchIDs(g.touchIDs[:0])
		if len(g.touchIDs) == 0 {
			return 0, 0
		}

		g.touchID = g.touchIDs[0]
		x, y := ebiten.TouchPosition(g.touchID)
		g.touchOriginX, g.touchOriginY = float64(x), float64(y)
		g.touchActive = true

		// Anchor and finger are the same point on this first frame, so there
		// is no direction to report yet.
		return 0, 0
	}

	// Only IsTouchJustReleased can report a lifted finger. TouchPosition
	// answers (0, 0) for a touch that is gone, and (0, 0) is the top-left
	// corner - a real position a real thumb can be at.
	if inpututil.IsTouchJustReleased(g.touchID) {
		g.touchActive = false
		return 0, 0
	}

	x, y := ebiten.TouchPosition(g.touchID)
	dx, dy := float64(x)-g.touchOriginX, float64(y)-g.touchOriginY

	if math.Hypot(dx, dy) < deadZone {
		return 0, 0
	}

	return dx, dy
}

// fits reports whether the whole bike, not just its centre, can stand at this
// position. Checking the four corners of its box is enough: a tile is 32px and
// the bike is 24px, so no wall can fit between the corners unnoticed.
func (g *Game) fits(x, y float64) bool {
	const half = bikeSize / 2

	return !g.blocked(x-half, y-half) &&
		!g.blocked(x+half, y-half) &&
		!g.blocked(x-half, y+half) &&
		!g.blocked(x+half, y+half)
}

// blocked turns a pixel position into a tile and says whether it is wall.
// Anything off the map counts as wall, which is also what freezes the bike
// until the map arrives: with no tiles there is nowhere legal to ride.
func (g *Game) blocked(x, y float64) bool {
	if x < 0 || y < 0 {
		// Go truncates towards zero, so -1/32 would be tile 0 rather than -1.
		// Catching negatives here keeps that from wrapping onto the map.
		return true
	}

	tx, ty := int(x)/tileSize, int(y)/tileSize
	if ty >= len(g.tiles) || tx >= len(g.tiles[ty]) {
		return true
	}

	return g.tiles[ty][tx] == '#'
}

// normalize scales a vector to length 1 and leaves a zero vector alone.
// Without it the diagonal (1,1) would be 1.41 long, making diagonal riding
// 41% faster than riding straight.
func normalize(dx, dy float64) (float64, float64) {
	if dx == 0 && dy == 0 {
		// Dividing by a zero length would give NaN, and a NaN coordinate never
		// becomes a number again.
		return 0, 0
	}

	l := math.Hypot(dx, dy)
	return dx / l, dy / l
}

// Draw paints the current state. It must not change anything.
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(colorGrass)

	// Everything below is drawn at world coordinate minus camera, which is what
	// turns a 2400x1728 world into an 800x576 window onto it.
	camX, camY := g.camera()

	// Terrain first, so the spots and the bike land on top of it. Only the
	// tiles the camera can see are drawn: the world holds 4050 of them and
	// painting the off-screen ones is work nobody can look at. Floor tiles are
	// skipped too, because the Fill above already painted that colour.
	viewW, viewH := g.view()
	y0, y1 := int(camY)/tileSize, int(camY+viewH)/tileSize
	x0, x1 := int(camX)/tileSize, int(camX+viewW)/tileSize

	// One options value reused for every tile, for the same reason as above: a
	// fresh one inside the loop would be another 500 allocations per frame.
	op := &ebiten.DrawImageOptions{}

	for y := max(y0, 0); y <= min(y1, len(g.tiles)-1); y++ {
		row := g.tiles[y]
		for x := max(x0, 0); x <= min(x1, len(row)-1); x++ {
			tx := float64(x*tileSize) - camX
			ty := float64(y*tileSize) - camY

			if g.tileImg != nil {
				img, ok := g.tileImg[row[x]]
				if !ok {
					continue
				}

				// Reset first: GeoM accumulates, so without it every tile would
				// inherit the previous tile's translation.
				op.GeoM.Reset()
				op.GeoM.Scale(tileScale, tileScale)
				op.GeoM.Translate(tx, ty)
				screen.DrawImage(img, op)
				continue
			}

			// Fallback until the tileset lands, or if it never does. Floor is
			// skipped here because the Fill above already painted that colour.
			var c color.RGBA
			switch row[x] {
			case '#':
				c = colorWall
			case '_':
				c = colorDoor
			default:
				continue
			}

			vector.DrawFilledRect(screen, float32(tx), float32(ty), tileSize, tileSize, c, false)
		}
	}

	for _, s := range g.spots {
		drawColor := colorSpot
		if s.IsNear {
			drawColor = colorSpotNear
		}

		vector.DrawFilledCircle(
			screen,
			float32(s.X-camX), float32(s.Y-camY),
			float32(s.Radius),
			drawColor,
			true,
		)
	}

	// The sprite is 32px while the collision box is 24px, which is deliberate:
	// a shoulder overlapping a wall looks fine, a body stopped short of one
	// does not.
	if g.sprites != nil {
		row := g.facing
		if g.onBike {
			row += 4
		}

		// SubImage does not copy pixels, it returns a view into the same sheet.
		src := g.sprites.SubImage(image.Rect(
			g.animFrame*spriteSize, row*spriteSize,
			(g.animFrame+1)*spriteSize, (row+1)*spriteSize,
		)).(*ebiten.Image)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(spriteScale, spriteScale)

		// Anchored by the feet, not the centre. A sprite taller than its
		// collision box has to stand ON the box, or the avatar looks like it is
		// hovering above the ground it actually occupies.
		op.GeoM.Translate(
			g.bikeX-camX-spriteSize*spriteScale/2,
			g.bikeY-camY+bikeSize/2-spriteFoot*spriteScale,
		)
		screen.DrawImage(src, op)

		return
	}

	// Fallback while the sheet is loading, or if it failed to load at all.
	vector.DrawFilledRect(
		screen,
		float32(g.bikeX-camX)-bikeSize/2, float32(g.bikeY-camY)-bikeSize/2,
		bikeSize, bikeSize,
		colorBike,
		false,
	)
}

// Layout reports the resolution the game renders at. Returning the real canvas
// size instead of a fixed one means 1:1 pixels - nothing is upscaled, and a
// bigger window shows more of the world rather than the same view stretched.
//
// This is also the zoom knob: returning half of each would render half as many
// logical pixels and draw each one twice as large.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Rendering fewer logical pixels than the canvas has means Ebiten scales
	// the result up, so everything is drawn zoom times larger and zoom times
	// less world fits on screen.
	g.viewW = int(float64(outsideWidth) / zoom)
	g.viewH = int(float64(outsideHeight) / zoom)

	return g.viewW, g.viewH
}

// view is the size of the visible window, falling back to the fixed size for
// the first frames, before Layout has been called.
func (g *Game) view() (float64, float64) {
	if g.viewW == 0 || g.viewH == 0 {
		return screenWidth, screenHeight
	}

	return float64(g.viewW), float64(g.viewH)
}

func main() {

	g := &Game{
		// No start position here: the map's S tile decides it, and the map has
		// not arrived yet. Until it does, blocked() reports wall everywhere and
		// the bike cannot move off 0,0 anyway.
		activeSpot: -1,
		onBike:     true, // the concept is riding a bike through the CV
		spotsCh:    make(chan fetchSonuc, 1),
		mapCh:      make(chan mapSonuc, 1),
		spritesCh:  make(chan imageSonuc, 1),
		tilesetCh:  make(chan imageSonuc, 1),
	}

	js.Global().Set("closeStory", js.FuncOf(func(this js.Value, args []js.Value) any {
		g.activeSpot = -1
		return nil
	}))

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Story_time")

	// Say "loading" before starting the load, so the two read in order. Go
	// sends a key rather than a sentence: the wording is JS's business, because
	// JS is where the chosen language lives.
	js.Global().Call("showStatus", "loading")

	// Fetch the spots in the background. This must not block: RunGame below
	// takes over the thread and drives Update at 60fps, so anything waiting on
	// the network has to happen off to the side and hand its result over later.
	go func() {
		resp, err := http.Get("/api/spots")
		if err != nil {
			// A goroutine has no caller, so there is nowhere to return an
			// error to. It travels down the channel instead, and Update
			// decides what the player sees.
			g.spotsCh <- fetchSonuc{err: fmt.Errorf("istek: %w", err)}
			return
		}
		defer resp.Body.Close()

		// err above only means the request never happened. A 500 arrives with
		// err == nil, so the status has to be checked separately.
		if resp.StatusCode != http.StatusOK {
			g.spotsCh <- fetchSonuc{err: fmt.Errorf("beklenmeyen status: %d", resp.StatusCode)}
			return
		}

		var spots []spot.Spot
		if err := json.NewDecoder(resp.Body).Decode(&spots); err != nil {
			g.spotsCh <- fetchSonuc{err: fmt.Errorf("decode: %w", err)}
			return
		}

		g.spotsCh <- fetchSonuc{spots: spots}
	}()

	// The terrain is a second, independent request. Same three failure modes,
	// same shape - only the decoding differs, because map.txt is plain text
	// rather than JSON.
	go func() {
		resp, err := http.Get("/map.txt")
		if err != nil {
			g.mapCh <- mapSonuc{err: fmt.Errorf("istek: %w", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			g.mapCh <- mapSonuc{err: fmt.Errorf("beklenmeyen status: %d", resp.StatusCode)}
			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			g.mapCh <- mapSonuc{err: fmt.Errorf("okuma: %w", err)}
			return
		}

		// Strip \r first in case the file was saved from Windows, then trim the
		// trailing newline - splitting on it would leave an empty last row that
		// tiles[y][x] would panic on.
		text := strings.ReplaceAll(string(body), "\r", "")
		g.mapCh <- mapSonuc{tiles: strings.Split(strings.TrimSpace(text), "\n")}
	}()

	// Third and fourth fetches: the art. Same shape as the two above, except
	// the payload is decoded with image.Decode instead of a JSON or text parse.
	// Both PNGs are the same job, so it is one function called twice.
	go fetchImage("/sprites.png", g.spritesCh)
	go fetchImage("/tiles.png", g.tilesetCh)

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

// fetchImage downloads one PNG and decodes it, reporting either the image or
// the reason there is none on ch. It runs as a goroutine, which is why it
// cannot return the error: a goroutine has no caller to return it to.
func fetchImage(url string, ch chan imageSonuc) {
	resp, err := http.Get(url)
	if err != nil {
		ch <- imageSonuc{err: fmt.Errorf("istek: %w", err)}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ch <- imageSonuc{err: fmt.Errorf("beklenmeyen status: %d", resp.StatusCode)}
		return
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		ch <- imageSonuc{err: fmt.Errorf("decode: %w", err)}
		return
	}

	ch <- imageSonuc{img: img}
}
