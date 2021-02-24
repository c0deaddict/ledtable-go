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
)

type Color struct {
	r uint8
	g uint8
	b uint8
}

func rainbow(gradient float64) Color {
	c := colorful.Hsv(359.0*gradient, 1.0, 1.0)
	r, g, b := c.RGB255()
	return Color{r, g, b}
}

func sky(gradient float64) Color {
	v := uint8(gradient * 255)
	return Color{255 - v, 255 - v, 255}
}

func makeFrame(image []Color) []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(FLAG_SYNC)

	var offset uint16 = 0
	binary.Write(buf, binary.LittleEndian, offset)
	var count = uint16(len(image))
	binary.Write(buf, binary.LittleEndian, count)

	for i := 0; i < len(image); i++ {
		color := image[i]
		buf.WriteByte(color.r)
		buf.WriteByte(color.g)
		buf.WriteByte(color.b)
	}

	return buf.Bytes()

}

func perlinNoise(conn net.Conn, color func(gradient float64) Color) {
	randSource := rand.NewSource(time.Now().UnixNano())
	n := 2
	p := perlin.NewPerlinRandSource(2, 2, n, randSource)
	min := math.Inf(+1)
	max := math.Inf(-1)

	for i := 0; true; i++ {
		var image [WIDTH * HEIGHT]float64

		for y := 0; y < HEIGHT; y++ {
			for x := 0; x < WIDTH; x++ {
				val := p.Noise2D((float64(x)+float64(i)/8)/WIDTH, (float64(y)+float64(i)/5)/HEIGHT)
				min = math.Min(min, val)
				max = math.Max(max, val)
				image[y*WIDTH+x] = val
			}
		}

		pixels := make([]Color, len(image))
		for i := 0; i < len(image); i++ {
			gradient := (image[i] - min) / (max - min)
			pixels[i] = color(gradient)
		}

		conn.Write(makeFrame(pixels))
		time.Sleep(10 * time.Millisecond)
	}
}

// https://github.com/adonovan/gopl.io/blob/master/ch1/lissajous/main.go
func lissajous(conn net.Conn) {
	const (
		cycles = 1    // number of complete x oscillator revolutions
		res    = 0.01 // angular resolution
	)
	freq := rand.Float64() * 3.0
	phase := 0.0 // phase difference
	for i := 0; i < 10000; i++ {
		image := make([]Color, WIDTH*HEIGHT)
		for t := 0.0; t < cycles*2*math.Pi; t += res {
			x := math.Sin(t)
			y := math.Sin(t*freq + phase)
			ix := int(WIDTH * (1.0 + x) / 2.0)
			iy := int(HEIGHT * (1.0 + y) / 2.0)
			image[iy*WIDTH+ix] = Color{0, 255, 0}
		}
		conn.Write(makeFrame(image))
		time.Sleep(10 * time.Millisecond)
		phase += 0.05
	}
}

func main() {
	conn, err := net.Dial("udp4", "ledtable.dhcp:1337")
	if err != nil {
		log.Fatalln("Udp dial:", err)
	}

	lissajous(conn)
}
