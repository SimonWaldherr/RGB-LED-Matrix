package main

import (
	"flag"
	"fmt"
	"image/color"
	"image/gif"
	"log"
	"math/rand"
	"os"
	"time"

	rgbmatrix "simonwaldherr.de/go/rpirgbled"
)

var (
	rows       = flag.Int("led-rows", 32, "number of rows supported")
	parallel   = flag.Int("led-parallel", 1, "number of daisy-chained panels")
	chain      = flag.Int("led-chain", 16, "number of displays daisy-chained")
	brightness = flag.Int("brightness", 99, "brightness (0-100)")
)

type Field struct {
	cells  [][]int
	width  int
	height int
}

var field *Field

var (
	setfps       int
	setwidth     int
	setheight    int
	setduration  int
	outputlength int
	setfilename  string
	outputfile   string
	port         string
)

func fatal(err error) {
	if err != nil {
		panic(err)
	}
}

func newField(width, height int) *Field {
	cells := make([][]int, height)
	for cols := range cells {
		cells[cols] = make([]int, width)
	}
	return &Field{cells: cells, width: width, height: height}
}

func randomUint() uint8 {
	return uint8(rand.Intn(255))
}

func smallRandomUint() uint8 {
	return uint8(rand.Intn(16))
}

func newXY(x, y int) (int, int) {
	switch {
	case y < 64:
		switch {
		case x < 32:
			return 192 + y, 31 - x
		case x < 64:
			return 191 - y, x - 32
		case x < 96:
			return 64 + y, 31 - (x - 64)
		case x < 128:
			return 63 - y, x - 96
		case x < 192:
			return 0, 0
		}
	case y < 128:
		yh := y - 64
		switch {
		case x < 32:
			return 256 + yh, 31 - x
		case x < 64:
			return 383 - yh, x - 32
		case x < 96:
			return 384 + yh, 31 - (x - 64)
		case x < 128:
			return 511 - yh, x - 96
		}
	}
	return 0, 0
}

func (field *Field) printField(setfilename string) string {
	file, err := os.Open(setfilename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	c.Clear()

	gif, _ := gif.DecodeAll(file)

	for i, srcImg := range gif.Image {
		start := time.Now() // Start time measurement

		for y := 0; y < field.height; y++ {
			for x := 0; x < field.width; x++ {
				x1, y1 := newXY(x, y)
				pixel := srcImg.At(x, y)
				r, g, b, _ := pixel.RGBA()
				c.Set(x1, y1, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
			}
		}
		c.Render()

		elapsed := time.Since(start) // Calculate elapsed time

		if i < len(gif.Delay) {
			delay := time.Duration(gif.Delay[i]*10) * time.Millisecond
			if delay > elapsed {
				time.Sleep(delay - elapsed) // Adjusted sleep time
			}
		}
	}
	return ""
}

var config = &rgbmatrix.DefaultConfig
var c *rgbmatrix.Canvas

func main() {
	flag.IntVar(&setwidth, "w", 128, "terminal width")
	flag.IntVar(&setheight, "h", 128, "terminal height")
	flag.IntVar(&setduration, "d", -1, "game of life duration")
	flag.IntVar(&setfps, "f", 20, "frames per second")
	flag.StringVar(&setfilename, "o", "./data.gif", "open file")
	flag.IntVar(&outputlength, "l", 200, "frames")

	flag.Parse()

	config.Rows = *rows
	config.ChainLength = *chain
	config.Parallel = *parallel
	config.PWMBits = 6
	config.PWMLSBNanoseconds = 95
	config.Brightness = *brightness
	config.ScanMode = rgbmatrix.Interlaced
	config.DisableHardwarePulsing = false

	m, err := rgbmatrix.NewRGBLedMatrix(config)
	fatal(err)
	c = rgbmatrix.NewCanvas(m)
	defer c.Close()

	field = newField(setwidth, setheight)

	for i := 0; i != setduration; i++ {
		field.printField(setfilename)
		fmt.Println("Reset Matrix")

		c.Clear()
		c.Close()

		m, _ = rgbmatrix.NewRGBLedMatrix(config)
		c = rgbmatrix.NewCanvas(m)
	}
}
