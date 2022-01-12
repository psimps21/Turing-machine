package main

import (
	"bufio"
	"canvas"
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"
	"strings"
)

type ColorID int
type State int
type Direction int

const (
	ForwardDir = iota
	RightDir
	BackwardDir
	LeftDir
)

const (
	NorthDir = ForwardDir
	SouthDir = BackwardDir
)

type Signal struct {
	state State
	color ColorID
}

type Action struct {
	state State
	color ColorID
	turn  Direction
}

type Turmite struct {
	rules      map[Signal]Action
	x, y       int
	currentDir Direction
	state      State
}

type Field [][]ColorID

// NewField creates a new square field of the given edge size.
func NewField(size int) Field {
	f := make([][]ColorID, size)
	for i := range f {
		f[i] = make([]ColorID, size)
	}
	return f
}

// DrawField draws the field to a PNG in the given filename. Assumes that
// field[y][x] is at the cell (y,x), where the origin is at the top-right
// corner.
func (f Field) DrawField(filename string) {
	const scale = 5
	n := len(f)
	out := canvas.CreateNewCanvas(n*scale, n*scale)

	for x := 0; x < n; x++ {
		for y := 0; y < n; y++ {
			out.SetFillColor(f[y][x].ToColor())
			out.ClearRect(x*scale, y*scale, (x+1)*scale, (y+1)*scale)
		}
	}

	out.SaveToPNG(filename)
}

// ToRGB returns the red, green, blue values for a given color id.
func (c ColorID) ToColor() color.Color {
	colors := [][]uint8{
		{0, 0, 0},
		{254, 209, 0},
		{0, 125, 0},
		{0, 0, 125},
		{125, 0, 125},
		{255, 255, 255},
		{254, 208, 0}, // 6 special center x color {124, 0, 0},
		{0, 155, 58},  // 7 jamica green
	}
	return canvas.MakeColor(colors[c][0], colors[c][1], colors[c][2])
}

// DirFromString returns a direction constant given an English string.
func DirFromString(s string) (Direction, error) {
	switch strings.ToLower(s) {
	case "forward", "f":
		return ForwardDir, nil
	case "backward", "back":
		return BackwardDir, nil
	case "left", "l":
		return LeftDir, nil
	case "right", "r":
		return RightDir, nil
	default:
		return 0, fmt.Errorf("unknown direction type: %s", s)
	}
}

// PositiveMod computes n % m, returning a number in [0,m-1].
func PositiveMod(n, m int) int {
	return ((n % m) + m) % m
}

// Left returns the direction turing 90 degrees left of d.
func (d Direction) Left() Direction {
	return Direction(PositiveMod(int(d)-1, 4))
}

// Right returns the direction turning 90 degrees right of d.
func (d Direction) Right() Direction {
	return Direction(PositiveMod(int(d)+1, 4))
}

// ReadTurmite reads a file that specifies the turmite rules. The file should
// have lines of the format:
//
//  state color -> state color direction
//
// where state is a lowercase letter a-z; color is an integer;  direction is a
// direction understood by DirFromString. The returned Turmite will be
// positioned at the center of the field and facing north (aka ForwardDir).
func ReadTurmite(filename string, size int) (*Turmite, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	tur := Turmite{
		x:          size / 2,
		y:          size / 2,
		currentDir: NorthDir,
		state:      0,
		rules:      make(map[Signal]Action),
	}

	scanner := bufio.NewScanner(file)
	for lineno := 1; scanner.Scan(); lineno++ {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		var color_in, color_out ColorID
		var dirString string
		var state_in_char, state_out_char rune

		n, err := fmt.Sscanf(line, "%c %d -> %c %d %s",
			&state_in_char,
			&color_in,
			&state_out_char,
			&color_out,
			&dirString)
		if err != nil || n != 5 {
			return nil, fmt.Errorf("Badly formatted line: %d", lineno)
		}
		state_in := State(state_in_char - 'a')
		state_out := State(state_out_char - 'a')
		dir, err := DirFromString(dirString)
		if err != nil {
			return nil, err
		}
		tur.rules[Signal{state: state_in, color: color_in}] = Action{
			state: state_out,
			color: color_out,
			turn:  dir,
		}
	}
	fmt.Printf("Read turmite with %d rules\n", len(tur.rules))
	return &tur, nil
}

// Returns the new x and y after moving one step
func (t *Turmite) MoveOneStep() (int, int) {
	var newX, newY int
	switch t.currentDir {
	case 0: // North
		newX, newY = t.x-1, t.y
	case 1: // East
		newX, newY = t.x, t.y+1
	case 2: // South
		newX, newY = t.x+1, t.y
	case 3: // West
		newX, newY = t.x, t.y-1
	}
	return newX, newY
}

// Step moves the turmite one step using the given field. Return an error if the
// turmite gets stuck with no rule to apply.
func (t *Turmite) Step(field Field) error {
	change := false
	for sig, act := range t.rules {
		if t.state == sig.state && field[t.x][t.y] == sig.color {
			change = true
			t.state = act.state
			field[t.x][t.y] = act.color
			switch act.turn {
			case 1:
				t.currentDir = t.currentDir.Right()
			case 2:
				t.currentDir = t.currentDir.Right().Right()
			case 3:
				t.currentDir = t.currentDir.Left()
			}
			newX, newY := t.MoveOneStep()
			if newX < 0 {
				newX = len(field) - 1
			} else if newX == len(field) {
				newX = 0
			} else if newY < 0 {
				newY = len(field) - 1
			} else if newY == len(field) {
				newY = 0
			}
			t.x, t.y = newX, newY
			break
		}
	}

	if !change {
		return fmt.Errorf("No rule for current state: %d and current color: %d", t.state, field[t.x][t.y])
	}
	// FILL IN THIS FUNCTION
	return nil
}

func main() {
	var program, pngfile string
	var fieldSize, iters int

	flag.StringVar(&program, "prog", "", "File containing the turmite program")
	flag.IntVar(&fieldSize, "s", 101, "Size of the field")
	flag.IntVar(&iters, "steps", 6220, "Number of steps")
	flag.StringVar(&pngfile, "o", "output.png", "Filename to draw output")
	flag.Parse()

	if program == "" {
		log.Fatal("Must supply a program file with -prog.")
	}

	mite, err := ReadTurmite(program, fieldSize)
	if err != nil {
		log.Fatal(err)
	}
	field := NewField(fieldSize)
	for i := 0; i < iters; i++ {
		err = mite.Step(field)
		if err != nil {
			fmt.Printf("State: %d, Color: %d, Dir:%d, Pos:(%d,%d)\n", mite.state, field[mite.x][mite.y], mite.currentDir, mite.x, mite.y)
			log.Fatal(err)
		}
	}
	fmt.Printf("State: %d, Color: %d, Dir:%d, Pos:(%d,%d)\n", mite.state, field[mite.x][mite.y], mite.currentDir, mite.x, mite.y)
	field.DrawField(pngfile)
}
