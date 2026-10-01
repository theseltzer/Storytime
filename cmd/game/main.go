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

	"log"
	"math"
	"net/http"
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

	// The street. groundY is the world Y the avatar's centre rides at; its feet
	// land bikeSize/2 below that, which is where the road surface is drawn.
	//
	// groundDepth is how much road stays visible below the feet. The camera is
	// anchored to it rather than to the top of the world, so a short browser
	// window still shows the street instead of a screenful of sky.
	groundY     = 400
	groundDepth = 96

	// The street runs from 0 to worldWidth. A placeholder: the real length will
	// come out of the spot positions once they are re-authored for a world that
	// only has an X axis.
	worldWidth = 3200
	startX     = 120
)

// Facing directions, in the row order of the sprite sheet.
const (
	faceDown = iota
	faceLeft
	faceRight
	faceUp
)

var (
	colorBike     = color.RGBA{0xe8, 0x9c, 0x2a, 0xff}
	colorSpot     = color.RGBA{0x2d, 0x54, 0x29, 0xff} // Dark green when away
	colorSpotNear = color.RGBA{0xf1, 0xc4, 0x0f, 0xff} // bright yellow when close

	// Two flat bands until the parallax layers land. Dusk blue and near-black
	// asphalt, picked so the silhouette look of the next step already reads.
	colorSky    = color.RGBA{0x2b, 0x3a, 0x55, 0xff}
	colorGround = color.RGBA{0x1a, 0x1c, 0x22, 0xff}
)

// bar is one building silhouette: where it starts along the street, how wide,
// how tall. Its base always sits on the road, so no Y is needed.
type bar struct {
	x float64
	w float64
	h float64
}

// layer is one depth band of the skyline. factor is how fast it scrolls
// relative to the camera: 1 is street level, smaller is farther away.
type layer struct {
	factor float64
	clr    color.RGBA
	bars   []bar
}

// Game holds everything that changes over time. Ebiten calls Update and Draw
// on this one value forever, so all game state lives here.
type Game struct {
	bikeX, bikeY float64
	spots        []spot.Spot
	activeSpot   int
	spotsCh      chan fetchSonuc

	// Set once the spots have landed, so the DOM loading indicator is taken
	// down exactly once rather than sixty times a second.
	loaded bool
	failed bool

	// The size of the window the world is being watched through, in world
	// pixels. Set by Layout every frame; the world itself is far larger.
	viewW, viewH int

	sprites   *ebiten.Image
	spritesCh chan imageSonuc
	layers    []layer

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

	case sp := <-g.spritesCh:
		if sp.err != nil {
			// Not fatal: Draw falls back to a plain rectangle, so the game is
			// still playable without art.
			log.Printf("fetch sprites: %v", sp.err)
		} else {
			g.sprites = ebiten.NewImageFromImage(sp.img)
		}

	// Without this the select waits for a channel that has nothing left to
	// send, and Update never returns. An empty default is what makes the whole
	// block "take a result if one is ready, otherwise carry on".
	default:
	}

	// The indicator can only come down once, so the flag is an edge trigger
	// rather than a copy of what the DOM already knows: the bridge is crossed
	// one time instead of sixty times a second.
	if !g.loaded && !g.failed && g.spots != nil {
		g.loaded = true
		js.Global().Call("hideStatus")
	}

	// All input collapses into one number, so the movement below never learns
	// which device produced it. While the story panel is open it stays zero and
	// the bike simply stops.
	var dx float64
	if g.activeSpot == -1 {
		dx = g.input()
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

	// One axis, so there is nothing to test separately and nothing to slide
	// along: the street has no walls. Clamped to the ends of the street, which
	// is the only limit left now that the tile map is gone.
	g.bikeX = min(max(g.bikeX+dx*speed, 0), worldWidth)

	// Y is no longer an input, it is a result. The avatar is on the ground
	// every frame because there is no jump to lift it off.
	g.bikeY = groundAt(g.bikeX)

	// Two directions instead of four: a side-scroller only ever faces the way
	// it is walking.
	if dx != 0 {
		g.facing = faceLeft
		if dx > 0 {
			g.facing = faceRight
		}

		g.animTick++
		g.animFrame = (g.animTick / animTicksPerFrame) % 4
	} else {
		// Standing still shows frame 0 but keeps the last facing, otherwise the
		// avatar would snap back to one direction every time it stopped.
		g.animFrame = 0
	}

	// Proximity along the street only. Both the avatar and the spots sit on the
	// same ground line, so the Y term of a distance would always be zero - and
	// the spot Y values still hold top-down coordinates that mean nothing here.
	for i := range g.spots {
		g.spots[i].IsNear = math.Abs(g.bikeX-g.spots[i].X) <= g.spots[i].Radius
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

// groundAt is the world Y the avatar rides at for a given position along the
// street. Constant today; it exists as a function so a sloped or stepped street
// later is a change here and nowhere else.
func groundAt(x float64) float64 {
	return groundY
}

// camera returns the world coordinate drawn at the top-left of the screen.
//
// X behaves as before: the avatar is centred, then clamped so the view never
// slides past either end of the street.
//
// Y no longer follows anything. It is derived from the ground line so the road
// always sits groundDepth pixels above the bottom edge, whatever height the
// browser window happens to be. A tall window shows more sky, not more road.
func (g *Game) camera() (float64, float64) {
	viewW, viewH := g.view()

	// The outer max guards a view wider than the street, where the two clamp
	// bounds would otherwise cross over.
	camX := min(max(g.bikeX-viewW/2, 0), max(worldWidth-viewW, 0))
	camY := groundY + bikeSize/2 + groundDepth - viewH

	return camX, camY
}

// input returns this frame's direction: -1, 0 or 1. Keyboard wins when both are
// live; AppendTouchIDs is always empty on desktop, so the touch half costs
// nothing there.
func (g *Game) input() float64 {
	var dx float64

	// IsKeyPressed, not IsKeyJustPressed: this asks "is the key down right
	// now", which is what holding a direction means. Adding instead of
	// choosing makes A+D cancel to zero on its own.
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		dx--
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		dx++
	}

	if dx == 0 {
		// The stick reports how far the thumb has travelled, in pixels. Only its
		// sign matters here - feeding the raw distance in would make speed
		// depend on how far the thumb happened to slide.
		if tx, _ := g.joystick(); tx < 0 {
			dx = -1
		} else if tx > 0 {
			dx = 1
		}
	}

	return dx
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

// Draw paints the current state. It must not change anything.
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(colorSky)

	// Everything below is drawn at world coordinate minus camera, which is what
	// turns a 3200px street into a window onto part of it.
	camX, camY := g.camera()
	viewW, viewH := g.view()

	// The road: one band from the ground line down past the bottom edge. It
	// spans the screen rather than the world, because a flat colour has no
	// features to slide - drawing it in world coordinates would cost a
	// translation nobody could see.
	roadTop := float32(groundY + bikeSize/2 - camY)

	for _, l := range g.layers {
		for _, b := range l.bars {
			screenX := b.x - camX*l.factor
			topY := float32(roadTop) - float32(b.h)
			vector.FillRect(screen, float32(screenX), float32(topY), float32(b.w), float32(b.h), l.clr, false)
		}
	}

	vector.DrawFilledRect(screen, 0, roadTop, float32(viewW), float32(viewH), colorGround, false)

	// Placeholder markers until the buildings arrive. They are pinned to the
	// ground line, not to their stored Y: those Y values were authored for a
	// top-down world and mean nothing on a street.
	for _, s := range g.spots {
		drawColor := colorSpot
		if s.IsNear {
			drawColor = colorSpotNear
		}

		vector.DrawFilledCircle(
			screen,
			float32(s.X-camX), float32(groundY-camY),
			float32(s.Radius),
			drawColor,
			true,
		)
	}

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
		// The start position is a constant now. It used to be the map's S tile,
		// but there is no map to read it out of - and on a street there is only
		// one interesting question about where to begin: how far along.
		bikeX:      startX,
		bikeY:      groundAt(startX),
		activeSpot: -1,
		onBike:     true, // the concept is riding a bike through the CV
		spotsCh:    make(chan fetchSonuc, 1),
		spritesCh:  make(chan imageSonuc, 1),
		layers: []layer{
			{
				factor: 0.35,
				clr:    color.RGBA{0x22, 0x2e, 0x44, 0xff},
				bars: []bar{
					{x: 60, w: 140, h: 180},
					{x: 260, w: 90, h: 120},
					{x: 420, w: 180, h: 220},
					{x: 700, w: 110, h: 150},
					{x: 900, w: 160, h: 200},
				},
			},
		},
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

	// The art is a second, independent request. Same shape as the one above,
	// except the payload is decoded with image.Decode instead of a JSON parse.
	go fetchImage("/sprites.png", g.spritesCh)

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
