package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/term"
)

var version = "0.0.1"

type editorConfig struct {
	cx          int
	cy          int
	screenRows  int
	screenCols  int
	origTermios *term.State
}

var E editorConfig

type abuf struct {
	b   []byte
	len int
}

var ABUF_INIT = abuf{b: nil, len: 0}

// Append a string to the buffer
func abAppend(ab *abuf, s []byte) {
	ab.b = append(ab.b, s...)
	ab.len += len(s)
}

// Sets the terminal to raw mode and returns the previous state of the terminal
// This allows us to restore the terminal to its previous state when the program exits
// Visit term package docs for more info: https://pkg.go.dev/golang.org/x/term
func enableRawMode() error {
	fd := int(os.Stdin.Fd())

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}

	E.origTermios = oldState
	return nil
}

// Use this to restore the terminal to its previous state when the program exits
func disableRawMode(oldState *term.State) error {
	return term.Restore(int(os.Stdin.Fd()), oldState)
}

// CTRL_KEY macro bitwise ANDs a key with 00011111 in binary (0x1f in hex)
// This sets the upper 3 bits of the byte to 0
// This is the equivalent of pressing CTRL and a key at the same time
func CTRL_KEY(k byte) byte {
	return k & 0x1f
}

const (
	ARROW_UP    = 'w'
	ARROW_DOWN  = 's'
	ARROW_RIGHT = 'd'
	ARROW_LEFT  = 'a'
	DEL_KEY     = '\x7f'
	PAGE_UP     = 'k'
	PAGE_DOWN   = 'j'
	HOME_KEY    = 'h'
	END_KEY     = 'l'
)

// Read a keypress and return the byte representation of the key
func editorReadKey() (byte, error) {
	var c [1]byte
	_, err := os.Stdin.Read(c[:])
	if err != nil {
		return 0, err
	}
	if c[0] == '\x1b' {
		seq := make([]byte, 3)
		_, err := os.Stdin.Read(seq)
		if err != nil || seq[0] != '[' {
			return '\x1b', err
		}
		if seq[0] == '[' {
			if seq[1] >= '0' && seq[1] <= '9' {
				_, err := os.Stdin.Read(seq[2:])
				if err != nil {
					return '\x1b', err
				}
				if seq[2] == '~' {
					switch seq[1] {
					case '1':
						return HOME_KEY, nil
					case '3':
						return DEL_KEY, nil
					case '4':
						return END_KEY, nil
					case '5':
						return PAGE_UP, nil
					case '6':
						return PAGE_DOWN, nil
					case '7':
						return HOME_KEY, nil
					case '8':
						return END_KEY, nil
					}
				}
			} else {
				switch seq[1] {
				case 'A':
					return ARROW_UP, nil
				case 'B':
					return ARROW_DOWN, nil
				case 'C':
					return ARROW_RIGHT, nil
				case 'D':
					return ARROW_LEFT, nil
				case 'H':
					return HOME_KEY, nil
				case 'F':
					return END_KEY, nil
				}
			}
		} else if seq[0] == 'O' {
			switch seq[1] {
			case 'H':
				return HOME_KEY, nil
			case 'F':
				return END_KEY, nil
			}
		}
		return '\x1b', nil
	}

	return c[0], nil
}

func editorMoveCursor(key byte) {
	switch key {
	case ARROW_LEFT:
		if E.cx != 0 {
			E.cx--
		}
	case ARROW_RIGHT:
		if E.cx != E.screenCols-1 {
			E.cx++
		}
	case ARROW_UP:
		if E.cy != 0 {
			E.cy--
		}
	case ARROW_DOWN:
		if E.cy != E.screenRows-1 {
			E.cy++
		}
	}
}

// Logic for processing keypresses
func editorProcessKeypress() {
	c, err := editorReadKey()

	if err != nil {
		fmt.Println("Error reading key:", err)
		return
	}

	switch c {
	case CTRL_KEY('q'):
		fmt.Print("\x1b[2J")
		fmt.Print("\x1b[H")
		disableRawMode(E.origTermios)
		os.Exit(0)
	case HOME_KEY:
		E.cx = 0
	case END_KEY:
		E.cx = E.screenCols - 1
	case PAGE_UP, PAGE_DOWN:
		times := E.screenRows
		for times > 0 {
			if c == PAGE_UP {
				editorMoveCursor(ARROW_UP)
			} else if c == PAGE_DOWN {
				editorMoveCursor(ARROW_DOWN)
			}
			times--
		}
	case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT:
		editorMoveCursor(c)
	}
}

// Draw tilde characters to fill the screen similar to vim
// Welcome message is displayed in the middle of the screen using padding
// \x1b[K is the escape sequence to clear the line
// \r\n is the escape sequence to move to the next line
func editorDrawRows() {
	for i := 0; i < E.screenRows; i++ {
		if i == E.screenRows/3 {
			welcome := fmt.Sprintf("GoEd -- version %s", version)
			if len(welcome) > E.screenCols {
				welcome = welcome[:E.screenCols]
			}
			padding := (E.screenCols - len(welcome)) / 2
			if padding > 0 {
				abAppend(&ABUF_INIT, []byte("~"))
				padding--
			}
			for ; padding > 0; padding-- {
				abAppend(&ABUF_INIT, []byte(" "))
			}
			abAppend(&ABUF_INIT, []byte(welcome))
		} else {
			abAppend(&ABUF_INIT, []byte("~"))
		}
		abAppend(&ABUF_INIT, []byte("\x1b[K"))
		if i < E.screenRows-1 {
			abAppend(&ABUF_INIT, []byte("\r\n"))
		}
	}
}

// \x1b is the escape character which is 27 in decimal
// [?25l is the escape sequence to hide the cursor
// [H is the escape sequence to position the cursor at the top left of the screen
// [%d;%dH is the escape sequence to position the cursor at a specific row and column
// [?25h is the escape sequence to show the cursor]
// This is based on the VT100 terminal escape sequences
// Visit https://vt100.net/docs/vt100-ug/chapter3.html#ED for more info
func editorRefreshScreen() {
	abAppend(&ABUF_INIT, []byte("\x1b[?25l"))
	abAppend(&ABUF_INIT, []byte("\x1b[H"))

	editorDrawRows()

	// Ensure cursor position is within the screen boundaries
	if E.cy >= E.screenRows {
		E.cy = E.screenRows - 1
	}
	if E.cx >= E.screenCols {
		E.cx = E.screenCols - 1
	}

	buf := fmt.Sprintf("\x1b[%d;%dH", E.cy+1, E.cx+1)
	abAppend(&ABUF_INIT, []byte(buf))

	abAppend(&ABUF_INIT, []byte("\x1b[?25h"))

	fmt.Print(string(ABUF_INIT.b))
}

// Get the size of the terminal window using the term package
// Visit term package docs for more info: https://pkg.go.dev/golang.org/x/term
func getWindowSize() (int, int, error) {
	return term.GetSize(int(os.Stdin.Fd()))
}

// Initialize the editor by getting the window size and setting the cursor position to 0,0
func initEditor() error {
	E.cx = 0
	E.cy = 0
	var err error
	E.screenCols, E.screenRows, err = getWindowSize()
	if err != nil {
		return fmt.Errorf("error getting window size: %v", err)
	}
	return nil
}

func main() {
	err := enableRawMode()
	if err != nil {
		log.Fatalf("Error setting raw mode: %v", err)
		return
	}

	initEditor()

	defer disableRawMode(E.origTermios)

	for {
		editorRefreshScreen()
		editorProcessKeypress()
	}
}
