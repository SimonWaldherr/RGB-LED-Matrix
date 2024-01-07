package main

import (
	"flag"
	"fmt"
	"image/color"
	"image/png"
	"io/ioutil"
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

type Cell struct {
	col color.RGBA
	vit int
}

type Field struct {
	cells  [][]Cell
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
	cells := make([][]Cell, height)
	for cols := range cells {
		cells[cols] = make([]Cell, width)
	}
	return &Field{cells: cells, width: width, height: height}
}

func (field *Field) setVitality(x, y int, vitality int, c color.RGBA) {
	x += field.width
	x %= field.width
	y += field.height
	y %= field.height
	if vitality < 1 {
		field.cells[y][x] = Cell{vit: 0, col: color.RGBA{0, 0, 0, 0}}
	}
	field.cells[y][x] = Cell{vit: vitality, col: c}
}

func (field *Field) getVitality(x, y int) Cell {
	x += field.width
	x %= field.width
	y += field.height
	y %= field.height
	return field.cells[y][x]
}

func (field *Field) nextVitality(x, y int) Cell {
	var r, g, b uint8
	var alive uint8
	for i := -1; i <= 1; i++ {
		for j := -1; j <= 1; j++ {
			cell := field.getVitality(x+i, y+j)
			if (j != 0 || i != 0) && (cell.vit > 0) {
				alive++
				r += cell.col.R
				g += cell.col.G
				b += cell.col.B
			}
		}
	}

	if alive > 1 {
		r, g, b = r/(alive), g/(alive), b/(alive)
	}

	if int(r)+int(g)+int(b) < 400 {
		if r >= g && r >= b {
			r += 100
		} else if g >= b && g >= r {
			g += 100
		} else if b >= g && b >= r {
			b += 100
		}
	}

	if r > 255 {
		r = 255
	}
	if g > 255 {
		g = 255
	}
	if b > 255 {
		b = 255
	}

	cell := field.getVitality(x, y)
	if alive == 3 || alive == 2 && (cell.vit > 0) {
		if cell.vit < 8 {
			return Cell{vit: cell.vit + 1, col: color.RGBA{r, g, b, 255}}
		} else {
			return Cell{vit: cell.vit, col: color.RGBA{r, g, b, 255}}
		}
	}

	return Cell{vit: 0, col: color.RGBA{0, 0, 0, 255}}
}

func generateFirstRound(width, height int) *Field {
	field := newField(width, height)
	for i := 0; i < (width * height / 4); i++ {
		field.setVitality(rand.Intn(width), rand.Intn(height), 1, color.RGBA{uint8(rand.Intn(255)), uint8(rand.Intn(255)), uint8(rand.Intn(255)), 255})
	}
	return field
}

func loadFirstRound(width, height int, filename string) *Field {
	finfo, err := os.Stat(filename)
	if err != nil {
		fmt.Println(filename + " doesn't exist")
		return generateFirstRound(width, height)
	} else {
		if finfo.IsDir() {
			fmt.Println(filename + " is a directory")
			return generateFirstRound(width, height)
		} else {
			field := newField(width, height)
			if filename[len(filename)-3:len(filename)] == "txt" {
				gofile, _ := ioutil.ReadFile(filename)
				output := []rune(string(gofile))
				x := 0
				y := 0
				for _, char := range output {
					col := color.RGBA{uint8(rand.Intn(255)), uint8(rand.Intn(255)), uint8(rand.Intn(255)), 255}
					switch char {
					case 10:
						y++
						x = 0
					case 49:
						field.setVitality(x, y, 1, col)
					case 50:
						field.setVitality(x, y, 2, col)
					case 51:
						field.setVitality(x, y, 3, col)
					case 52:
						field.setVitality(x, y, 4, col)
					case 53:
						field.setVitality(x, y, 5, col)
					case 54:
						field.setVitality(x, y, 6, col)
					case 55:
						field.setVitality(x, y, 7, col)
					case 56:
						field.setVitality(x, y, 8, col)
					case 57:
						field.setVitality(x, y, 9, col)
					default:
						if char != 32 {
							field.setVitality(x, y, 1, col)
						} else {
							field.setVitality(x, y, 0, col)
						}
					}
					x++
				}
			} else if filename[len(filename)-3:len(filename)] == "png" {
				file, _ := os.Open(filename)
				defer file.Close()

				img, _ := png.Decode(file)

				for y := 0; y < 127; y++ {
					for x := 0; x < 127; x++ {
						oldPixel := img.At(x, y)
						r, g, b, _ := oldPixel.RGBA()

						col := color.RGBA{uint8(r), uint8(g), uint8(b), 255}

						if r > 128 || g > 128 || b > 128 {
							field.setVitality(x, y, 9, col)
						} else if r > 16 || g > 16 || b > 16 {
							field.setVitality(x, y, 1, col)
						}
					}
				}
			}

			return field
		}
	}
	return generateFirstRound(width, height)
}

func (field *Field) nextRound() *Field {
	new_field := newField(field.width, field.height)
	for y := 0; y < field.height; y++ {
		for x := 0; x < field.width; x++ {
			cell := field.nextVitality(x, y)
			new_field.setVitality(x, y, cell.vit, cell.col)
		}
	}
	return new_field
}

func randomUint() uint8 {
	return uint8(rand.Intn(250))
}

func smallRandomUint() uint8 {
	return 254
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
	for y := 0; y < field.height; y++ {
		for x := 0; x < field.width; x++ {
			r := randomUint()
			g := randomUint()
			b := randomUint()

			x1, y1 := newXY(x, y)

			cell := field.getVitality(x, y)

			if cell.vit > 0 {

				if cell.vit > 3 {
					c.Set(x1, y1, color.RGBA{cell.col.R, cell.col.G, cell.col.B, 255})
				} else {
					c.Set(x1, y1, color.RGBA{r, g, b, 255})
				}
			} else {

			}
		}
	}
	c.Render()
	return ""
}

var config = &rgbmatrix.DefaultConfig
var c *rgbmatrix.Canvas

func main() {
	writer := gcurses.New()
	writer.Start()

	flag.IntVar(&setwidth, "w", 128, "terminal width")
	flag.IntVar(&setheight, "h", 128, "terminal height")
	flag.IntVar(&setduration, "d", -1, "game of life duration")
	flag.IntVar(&setfps, "f", 20, "frames per second")
	flag.StringVar(&setfilename, "o", "", "open file")
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

	for {
		m, err := rgbmatrix.NewRGBLedMatrix(config)
		fatal(err)
		c = rgbmatrix.NewCanvas(m)
		if setfilename != "" {
			log.Println("set via file")
			field = loadFirstRound(setwidth, setheight, setfilename)
			log.Println("file loaded")
		} else {
			log.Println("random seed")
			field = generateFirstRound(setwidth, setheight)
			log.Println("random seed generated")
		}
		field.printField()
		time.Sleep(3 * time.Second)

		for i := 0; i != setduration; i++ {
			field = field.nextRound()
			time.Sleep(time.Millisecond * 3)
			field.printField()
		}
		c.Close()
	}
}
