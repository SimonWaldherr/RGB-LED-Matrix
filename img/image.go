package main

import (
	"flag"
	"image/color"
	"image/png"
	"log"
	"math/rand"
	"os"
	"simonwaldherr.de/go/golibs/gcurses"
	rgbmatrix "simonwaldherr.de/go/rpirgbled"
	"time"
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

var config = &rgbmatrix.DefaultConfig
var c *rgbmatrix.Canvas

func randomUint() uint8 {
	return uint8(rand.Intn(255))
}

func smallRandomUint() uint8 {
	return uint8(rand.Intn(16))
}

func newXY(x, y int) (int, int) {
	var xh, yh int
	y1 := y
	x1 := x
	if y < 64 {
		yh = y
		if x < 32 {
			xh = x
			x1 = 192 + yh
			y1 = 31 - xh
		} else if x < 64 {
			xh = x - 32
			x1 = 191 - yh
			y1 = xh
		} else if x < 96 {
			xh = x - 64
			x1 = 64 + yh
			y1 = 31 - xh
		} else if x < 128 {
			xh = x - 96
			x1 = 63 - yh
			y1 = xh
		} else if x < 192 {
			x1 = 0
			y1 = 0
		}
	} else if y < 128 {
		yh = y - 64
		if x < 32 {
			xh = x
			x1 = 256 + yh
			y1 = 31 - xh
		} else if x < 64 {
			xh = x - 32
			x1 = 383 - yh
			y1 = xh
		} else if x < 96 {
			xh = x - 64
			x1 = 384 + yh
			y1 = 31 - xh
		} else if x < 128 {
			xh = x - 96
			x1 = 511 - yh
			y1 = xh
		} else if x < 192 {
			x1 = 0
			y1 = 0
		}
	} else {
		x1 = 0
		y1 = 0
	}
	return x1, y1
}

func resizeImage(src image.Image, newWidth, newHeight int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.NearestNeighbor.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)
	return dst
}

func (field *Field) printField() string {
	file, err := os.Open("./png.png")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	img, err := png.Decode(file)
	
	resizedImg := resizeImage(img, 128, 128)
	
	if err != nil {
		log.Fatal(os.Stderr, "%s: %v\n", "./png.png", err)
	}

	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {

			x1, y1 := newXY(x, y)

			oldPixel := resizedImg.At(x, y)
			r, g, b, _ := oldPixel.RGBA()
			c.Set(x1, y1, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		}
	}
	c.Render()
	time.Sleep(time.Minute * 15)
	return ""
}

func main() {
	writer := gcurses.New()
	writer.Start()

	flag.IntVar(&setwidth, "w", 80, "terminal width")
	flag.IntVar(&setheight, "h", 20, "terminal height")
	flag.IntVar(&setduration, "d", -1, "game of life duration")
	flag.IntVar(&setfps, "f", 20, "frames per second")
	flag.StringVar(&setfilename, "o", "", "open file")

	flag.IntVar(&outputlength, "l", 200, "frames")

	flag.Parse()

	config.Rows = *rows
	config.Parallel = *parallel
	config.ChainLength = *chain
	config.Brightness = *brightness

	m, err := rgbmatrix.NewRGBLedMatrix(config)
	fatal(err)

	c = rgbmatrix.NewCanvas(m)
	defer c.Close()

	for i := 0; i != setduration; i++ {
		time.Sleep(time.Millisecond * 25)
		field.printField()
	}
}
