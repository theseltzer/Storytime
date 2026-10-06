package skyline

import "math/rand/v2"

type Bar struct {
	X float64
	W float64
	H float64
}

type Config struct {
	Length float64
	MinW   float64
	MaxW   float64
	MinH   float64
	MaxH   float64
	MinGap float64
	MaxGap float64
}

func Generate(seed uint64, c Config) []Bar {
	x := 0.0
	var bars []Bar

	r := rand.New(rand.NewPCG(seed, seed))
	for x < c.Length {
		randWidth := r.Float64()*(c.MaxW-c.MinW) + c.MinW
		randHeight := r.Float64()*(c.MaxH-c.MinH) + c.MinH
		randGAP := r.Float64()*(c.MaxGap-c.MinGap) + c.MinGap
		bars = append(bars, Bar{X: x, W: randWidth, H: randHeight})
		x += randWidth + randGAP
	}
	return bars
}
