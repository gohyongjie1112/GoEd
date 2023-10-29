package main

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

type editorConfig struct {
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

// Read a keypress and return the byte representation of the key
func editorReadKey() (byte, error) {
	var c [1]byte
	_, err := os.Stdin.Read(c[:])
	if err != nil {
		return 0, err
	}
	return c[0], nil
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
	}
}

// Draw tilde characters to fill the screen similar to vim
// \x1b[K is the escape sequence to clear the line
// \r\n is the escape sequence to move to the next line
func editorDrawRows() {
	for i := 0; i < E.screenRows; i++ {
		abAppend(&ABUF_INIT, []byte("~"))
		abAppend(&ABUF_INIT, []byte("\x1b[K"))
		if i < E.screenRows-1 {
			abAppend(&ABUF_INIT, []byte("\r\n"))
		}
	}
}

// \x1b is the escape character which is 27 in decimal
// [?25l is the escape sequence to hide the cursor
// [2J is the escape sequence to clear the screen
// [H is the escape sequence to position the cursor at the top left of the screen
// This is based on the VT100 terminal escape sequences
// Visit https://vt100.net/docs/vt100-ug/chapter3.html#ED for more info
func editorRefreshScreen() {
	abAppend(&ABUF_INIT, []byte("\x1b[?25l"))
	abAppend(&ABUF_INIT, []byte("\x1b[2J"))
	abAppend(&ABUF_INIT, []byte("\x1b[H"))

	editorDrawRows()

	abAppend(&ABUF_INIT, []byte("\x1b[H"))
	abAppend(&ABUF_INIT, []byte("\x1b[?25h"))

	fmt.Print(string(ABUF_INIT.b))
}

// Get the size of the terminal window using the term package
// Visit term package docs for more info: https://pkg.go.dev/golang.org/x/term
func getWindowSize() (int, int, error) {
	return term.GetSize(int(os.Stdin.Fd()))
}

// Initialize the editor by getting the window size
func initEditor() error {
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
		fmt.Println("Error setting raw mode:", err)
		return
	}

	initEditor()

	defer disableRawMode(E.origTermios)

	for {
		editorRefreshScreen()
		editorProcessKeypress()
	}
}
