package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"math"
	"math/rand"
	"net"
	"time"

	perlin "github.com/aquilax/go-perlin"
	colorful "github.com/lucasb-eyer/go-colorful"
)

const (
	FLAG_SYNC = 1
	FLAG_RAW  = 2
)

const (
	WIDTH  = 15
	HEIGHT = 15
	XMID   = float64(WIDTH) / 2
	YMID   = float64(HEIGHT) / 2
)

type Color struct {
	r uint8
	g uint8
	b uint8
}

type ColorGradient func(gradient float64) Color

func rainbow(gradient float64) Color {
	c := colorful.Hsv(359.0*gradient, 1.0, 1.0)
	r, g, b := c.RGB255()
	return Color{r, g, b}
}

func sky(gradient float64) Color {
	v := uint8(gradient * 255)
	return Color{255 - v, 255 - v, 255}
}

func lerp(a Color, b Color) ColorGradient {
	dr := float64(b.r) - float64(a.r)
	dg := float64(b.g) - float64(a.g)
	db := float64(b.b) - float64(a.b)
	return func(v float64) Color {
		v = math.Pow(v, 1.5)
		if v < 0.01 {
			return a
		} else if v > 0.99 {
			return b
		} else {
			r := float64(a.r) + dr*v
			g := float64(a.g) + dg*v
			b := float64(a.b) + db*v
			return Color{uint8(r), uint8(g), uint8(b)}
		}
	}
}

func blue(gradient float64) Color {
	return Color{0, 0, uint8(gradient * 255)}
}

func makeFrame(image []Color) []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(FLAG_SYNC)

	var offset uint16 = 0
	binary.Write(buf, binary.BigEndian, offset)
	var count = uint16(len(image))
	binary.Write(buf, binary.BigEndian, count)

	for i := 0; i < len(image); i++ {
		color := image[i]
		buf.WriteByte(color.r)
		buf.WriteByte(color.g)
		buf.WriteByte(color.b)
	}

	return buf.Bytes()
}

func imageFromGradient(gradient []float64, color ColorGradient) []Color {
	image := make([]Color, len(gradient))
	for i := 0; i < len(gradient); i++ {
		image[i] = color(gradient[i])
	}
	return image
}

func normalize(gradient []float64) {
	min := math.Inf(+1)
	max := math.Inf(-1)
	for i := 0; i < len(gradient); i++ {
		min = math.Min(min, gradient[i])
		max = math.Max(max, gradient[i])
	}
	for i := 0; i < len(gradient); i++ {
		gradient[i] = (gradient[i] - min) / (max - min)
	}
}

func perlinNoise(conn net.Conn, color ColorGradient) {
	randSource := rand.NewSource(time.Now().UnixNano())
	n := 2
	p := perlin.NewPerlinRandSource(2, 2, n, randSource)
	min := math.Inf(+1)
	max := math.Inf(-1)

	for i := 0; true; i++ {
		gradient := make([]float64, WIDTH*HEIGHT)

		for y := 0; y < HEIGHT; y++ {
			for x := 0; x < WIDTH; x++ {
				val := p.Noise2D((float64(x)+float64(i)/8)/WIDTH, (float64(y)+float64(i)/5)/HEIGHT)
				min = math.Min(min, val)
				max = math.Max(max, val)
				gradient[y*WIDTH+x] = val
			}
		}

		// Normalize over whole run time.
		for i := 0; i < len(gradient); i++ {
			gradient[i] = (gradient[i] - min) / (max - min)
		}

		conn.Write(makeFrame(imageFromGradient(gradient, color)))
		time.Sleep(16 * time.Millisecond)
	}
}

// https://github.com/adonovan/gopl.io/blob/master/ch1/lissajous/main.go
func lissajous(conn net.Conn) {
	const (
		cycles = 2    // number of complete x oscillator revolutions
		res    = 0.01 // angular resolution
	)
	freq := rand.Float64() * 3.0
	phase := 0.0 // phase difference
	for i := 0; i < 10000; i++ {
		gradient := make([]float64, WIDTH*HEIGHT)
		for t := 0.0; t < cycles*2*math.Pi; t += res {
			x := math.Sin(t)
			y := math.Sin(t*freq + phase)
			ix := int(WIDTH * (1.0 + x) / 2.0)
			iy := int(HEIGHT * (1.0 + y) / 2.0)
			gradient[iy*WIDTH+ix]++
		}
		normalize(gradient)
		conn.Write(makeFrame(imageFromGradient(gradient, rainbow)))
		time.Sleep(10 * time.Millisecond)
		phase += 0.05
	}
}

func wavy(conn net.Conn, color ColorGradient) {
	for i := 0; true; i++ {
		gradient := make([]float64, WIDTH*HEIGHT)

		for y := 0; y < HEIGHT; y++ {
			for x := 0; x < WIDTH; x++ {
				t := float64(i) / 60.0
				v := math.Sin(t + float64(x)/WIDTH + math.Cos(float64(y)/(HEIGHT/3)))
				gradient[y*WIDTH+x] = (1.0 + v) / 2.0
			}
		}

		conn.Write(makeFrame(imageFromGradient(gradient, color)))
		time.Sleep(16 * time.Millisecond)
	}
}

type coord struct {
	x int
	y int
}

func rain(conn net.Conn, color ColorGradient) {
	w := 1.5
	k := -2.0

	wave := func(x int, y int, t float64) float64 {
		r := math.Sqrt(float64(x*x + y*y))
		v := math.Sin(k*r+w*t) / (1.0 + 0.5*r)
		return (1.0 + v) / (2.0 + math.Pow(t, 1.25))
	}

	drops := map[coord]int{}
	drops[coord{1, 1}] = 1
	delete(drops, coord{1, 1})

	for i := 0; true; i++ {
		gradient := make([]float64, WIDTH*HEIGHT)

		if rand.Intn(45) == 0 {
			c := coord{rand.Intn(WIDTH), rand.Intn(HEIGHT)}
			if _, ok := drops[c]; !ok {
				drops[c] = 1
			}
		}

		for y := 0; y < HEIGHT; y++ {
			for x := 0; x < WIDTH; x++ {
				v := 0.0
				count := 1
				for c, t := range drops {
					v += wave(x-c.x, y-c.y, float64(t)/10.0)
					count += 1
				}
				gradient[y*WIDTH+x] = math.Min(v, 1.0)
			}
		}

		for c, d := range drops {
			if d > 300 {
				delete(drops, c)
			} else {
				drops[c] = d + 1
			}
		}

		conn.Write(makeFrame(imageFromGradient(gradient, color)))
		time.Sleep(16 * time.Millisecond)
	}
}

func main() {
	conn, err := net.Dial("udp4", "ledtable.dhcp:1337")
	if err != nil {
		log.Fatalln("Udp dial:", err)
	}

	// rain(conn, lerp(Color{0, 0, 0}, Color{0, 0, 192}))
	rain(conn, lerp(Color{0, 0, 0}, Color{0, 0, 255}))
}
