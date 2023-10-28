package main

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

type editorConfig struct {
	screenRows int
	screenCols int
	termios    *term.State
}

var E editorConfig

// Sets the terminal to raw mode and returns the previous state of the terminal
// This allows us to restore the terminal to its previous state when the program exits
// Visit term package docs for more info: https://pkg.go.dev/golang.org/x/term
func enableRawMode() (*term.State, error) {
	fd := int(os.Stdin.Fd())

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}

	E.termios = oldState
	return oldState, nil
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
		disableRawMode(E.termios)
		os.Exit(0)
	}
}

// Draw tilde characters to fill the screen similar to vim
func editorDrawRows() {
	for i := 0; i < E.screenRows; i++ {
		fmt.Println("~")
	}
}

// \x1b is the escape character which is 27 in decimal
// [2J is the escape sequence to clear the screen
// [H is the escape sequence to position the cursor at the top left of the screen
// This is based on the VT100 terminal escape sequences
// Visit https://vt100.net/docs/vt100-ug/chapter3.html#ED for more info
func editorRefreshScreen() {
	fmt.Print("\x1b[2J")
	fmt.Print("\x1b[H")

	editorDrawRows()

	fmt.Print("\x1b[H")
}

// Get the size of the terminal window using the term package
// Visit term package docs for more info: https://pkg.go.dev/golang.org/x/term
func getWindowSize() (int, int, error) {
	return term.GetSize(int(os.Stdin.Fd()))
}

func initEditor() error {
	var err error
	E.screenRows, E.screenCols, err = getWindowSize()
	fmt.Print(E.screenRows, E.screenCols)
	if err != nil {
		return fmt.Errorf("error getting window size: %v", err)
	}
	return nil
}

func main() {
	oldState, err := enableRawMode()
	if err != nil {
		fmt.Println("Error setting raw mode:", err)
		return
	}

	initEditor()

	defer disableRawMode(oldState)

	for {
		editorRefreshScreen()
		editorProcessKeypress()
	}
}
