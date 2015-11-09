// (c) 2011-2014 Alexander Solovyov
// under terms of ISC License

package main

import (
	"fmt"
	"github.com/pkg/term"
	"github.com/wsxiaoys/terminal/color"
	"regexp"
)

type Printer struct {
	NoColors bool
	previous string
}

var removeMeta *regexp.Regexp = regexp.MustCompile(`@(\{[^}]+\}|.)`)

func (p *Printer) Printf(fmtstr string, args ...interface{}) {
	if p.NoColors {
		fmtstr = removeMeta.ReplaceAllLiteralString(fmtstr, "")
		fmt.Printf(fmtstr, args...)
	} else {
		color.Printf(fmtstr, args...)
	}
}

func (p *Printer) Sprintf(fmtstr string, args ...interface{}) string {
	if p.NoColors {
		fmtstr = removeMeta.ReplaceAllLiteralString(fmtstr, "")
		return fmt.Sprintf(fmtstr, args...)
	} else {
		return color.Sprintf(fmtstr, args...)
	}
}

// from https://github.com/paulrademacher/climenu/blob/master/getchar.go
// Returns either an ascii code, or (if input is an arrow) a Javascript key code.
func GetChar() (ascii int, keyCode int, err error) {
	t, _ := term.Open("/dev/tty")
	term.RawMode(t)
	bytes := make([]byte, 3)

	var numRead int
	numRead, err = t.Read(bytes)
	if err != nil {
		return
	}
	if numRead == 3 && bytes[0] == 27 && bytes[1] == 91 {
		// Three-character control sequence, beginning with "ESC-[".

		// Since there are no ASCII codes for arrow keys, we use
		// Javascript key codes.
		if bytes[2] == 65 {
			// Up
			keyCode = 38
		} else if bytes[2] == 66 {
			// Down
			keyCode = 40
		} else if bytes[2] == 67 {
			// Right
			keyCode = 39
		} else if bytes[2] == 68 {
			// Left
			keyCode = 37
		}
	} else if numRead == 1 {
		ascii = int(bytes[0])
	} else {
		// Two characters read??
	}
	t.Restore()
	t.Close()
	return
}
