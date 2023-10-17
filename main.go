package main

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

func enableRawMode() (*term.State, error) {
	fd := int(os.Stdin.Fd())

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}

	return oldState, nil
}

func main() {
	oldState, err := enableRawMode()
	if err != nil {
		fmt.Println("Error setting raw mode:", err)
		return
	}

	defer term.Restore(int(os.Stdin.Fd()), oldState)

	fmt.Println("Terminal in raw mode. Press 'q' to quit.")

	var c [1]byte
	for {
		_, err := os.Stdin.Read(c[:])
		if err != nil || c[0] == 'q' {
			break
		}
		fmt.Println("You pressed:", c[0])
	}
}
