package main

import (
	"flag"
	"image"
	"image/color"
	"image/draw"
	"math"
	"math/rand"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

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

var config = &rgbmatrix.DefaultConfig
var c *rgbmatrix.Canvas

func randomUint() uint8 {
	return uint8(rand.Intn(255))
}

func smallRandomUint() uint8 {
	return uint8(rand.Intn(16))
}

func genClock() *image.RGBA {
	const size = 128
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	draw.Draw(img, img.Bounds(), &image.Uniform{color.Black}, image.ZP, draw.Src)

	now := time.Now()
	hour, min, sec := now.Clock()
	dateStr := now.Format("02.01.2006")
	timeStr := now.Format("15:04:05")

	drawHand(img, size/2, size/2, int(float64(hour)/12*360), size/3, color.RGBA{200, 200, 200, 200}, 3)   // Stundenzeiger
	drawHand(img, size/2, size/2, int(float64(min)/60*360), size/2-10, color.RGBA{200, 200, 200, 200}, 2) // Minutenzeiger
	drawHand(img, size/2, size/2, int(float64(sec)/60*360), size/2-5, color.RGBA{200, 0, 0, 200}, 1)      // Sekundenzeiger, rot

	for i := 0; i < 12; i++ {
		angle := float64(i) / 12 * 2 * math.Pi
		x1 := size/2 + int(math.Cos(angle)*(float64(size)/2-1))
		y1 := size/2 + int(math.Sin(angle)*(float64(size)/2-1))
		x2 := size/2 + int(math.Cos(angle)*(float64(size)/2-10))
		y2 := size/2 + int(math.Sin(angle)*(float64(size)/2-10))
		drawLine(img, x1, y1, x2, y2, color.RGBA{200, 200, 200, 200})
	}

	addLabel(img, size/2, 100, dateStr)
	addLabel(img, size/2, 115, timeStr)

	return img
}

func drawHand(img *image.RGBA, x, y, angle, length int, col color.Color, width int) {
	rad := float64(angle-90) * math.Pi / 180

	endX := x + int(float64(length)*math.Cos(rad))
	endY := y + int(float64(length)*math.Sin(rad))

	drawThickLine(img, x, y, endX, endY, col, width)
}

// drawLine zeichnet eine einfache Linie von (x1, y1) zu (x2, y2)
func drawLine(img *image.RGBA, x1, y1, x2, y2 int, col color.Color) {
	dx := math.Abs(float64(x2 - x1))
	dy := math.Abs(float64(y2 - y1))
	sx := -1.0
	sy := -1.0
	if x1 < x2 {
		sx = 1.0
	}
	if y1 < y2 {
		sy = 1.0
	}
	err := dx - dy

	for {
		img.Set(int(x1), int(y1), col)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += int(sx)
		}
		if e2 < dx {
			err += dx
			y1 += int(sy)
		}
	}
}

func drawThickLine(img *image.RGBA, x1, y1, x2, y2 int, col color.Color, width int) {
	if width <= 1 {
		drawLine(img, x1, y1, x2, y2, col)
		return
	}

	offsets := make([]image.Point, 0, width*width)
	for dy := -width / 2; dy <= width/2; dy++ {
		for dx := -width / 2; dx <= width/2; dx++ {
			offsets = append(offsets, image.Point{dx, dy})
		}
	}

	for _, offset := range offsets {
		drawLine(img, x1+offset.X, y1+offset.Y, x2+offset.X, y2+offset.Y, col)
	}
}

// drawCircle zeichnet einen Kreis mit Mittelpunkt (x, y) und Radius r
func drawCircle(img *image.RGBA, x, y, r int, col color.Color) {
	putPixel := func(xc, yc, x, y int) {
		img.Set(xc+x, yc+y, col)
		img.Set(xc-x, yc+y, col)
		img.Set(xc+x, yc-y, col)
		img.Set(xc-x, yc-y, col)
		img.Set(xc+y, yc+x, col)
		img.Set(xc-y, yc+x, col)
		img.Set(xc+y, yc-x, col)
		img.Set(xc-y, yc-x, col)
	}

	x = 0
	y = r
	d := 3 - 2*r
	putPixel(x, y, x, y)
	for y >= x {
		x++
		if d > 0 {
			y--
			d = d + 4*(x-y) + 10
		} else {
			d = d + 4*x + 6
		}
		putPixel(x, y, x, y)
	}
}

func addLabel(img *image.RGBA, x, y int, label string) {
	col := color.RGBA{255, 255, 255, 255} // Weiß
	point := fixed.P(x-(len(label)*3), y) // Zentriert das Datum
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
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

func (field *Field) printField() string {
	img := genClock()

	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {

			x1, y1 := newXY(x, y)

			oldPixel := img.At(x, y)
			r, g, b, _ := oldPixel.RGBA()
			c.Set(x1, y1, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		}
	}
	c.Render()
	time.Sleep(time.Millisecond * 100)
	return ""
}

func main() {
	flag.IntVar(&setwidth, "w", 80, "terminal width")
	flag.IntVar(&setheight, "h", 20, "terminal height")
	flag.IntVar(&setduration, "d", -1, "game of life duration")
	flag.IntVar(&setfps, "f", 20, "frames per second")
	flag.StringVar(&setfilename, "o", "./clock.png", "open file")

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
		time.Sleep(time.Millisecond * 50)
		field.printField()
	}
}
