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
)

const (
	FLAG_SYNC = 1
	FLAG_RAW  = 2
)

const (
	WIDTH  = 15
	HEIGHT = 15
)

func main() {
	conn, err := net.Dial("udp4", "ledtable.dhcp:1337")
	if err != nil {
		log.Fatalln("Udp dial:", err)
	}

	randSource := rand.NewSource(time.Now().UnixNano())

	for i := 0; i < 100; i++ {
		buf := new(bytes.Buffer)
		buf.WriteByte(FLAG_SYNC)

		var offset uint16 = 0
		binary.Write(buf, binary.LittleEndian, offset)
		var count uint16 = WIDTH * HEIGHT
		binary.Write(buf, binary.LittleEndian, count)

		n := 3
		max := math.Sqrt(float64(n)) / 2
		log.Println(max)
		p := perlin.NewPerlinRandSource(3, 4, n, randSource)

		for y := 0; y < HEIGHT; y++ {
			for x := 0; x < WIDTH; x++ {
				c := p.Noise2D(float64(x)/WIDTH, float64(y)/HEIGHT)
				c = (max + c) / (2 * max)
				log.Println(c)
				buf.WriteByte(byte(c * 255))
				buf.WriteByte(0)
				buf.WriteByte(byte((1.0 - c) * 255))
			}
		}

		conn.Write(buf.Bytes())

		time.Sleep(1000 * time.Millisecond)
	}
}
