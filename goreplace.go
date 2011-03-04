package main

import (
	goopt "github.com/droundy/goopt"
	"os"
	"path"
	"fmt"
	"regexp"
	"bytes"
)

var Pattern *regexp.Regexp
var byteNewLine []byte = []byte("\n")

func main() {
	goopt.Description = func() string {
		return "Go search and replace in files"
	}
	goopt.Version = "0.1"
	goopt.Parse(nil)

	if len(goopt.Args) == 0 {
		println(goopt.Usage())
		return
	}

	var err os.Error
	Pattern, err = regexp.Compile(goopt.Args[0])
	errhandle(err)

	searchFiles()
}

func errhandle(err os.Error) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "ERR %s\n", err)
	os.Exit(1)
}

type Visitor struct{}

func (v *Visitor) VisitDir(p string, fi *os.FileInfo) bool {
	if fi.Name == ".hg" {
		return false
	}
	return true
}

func (v *Visitor) VisitFile(p string, fi *os.FileInfo) {
	if fi.Size >= 1024*1024*10 {
		fmt.Fprintf(os.Stderr, "Skipping %s, too big: %d\n", p, fi.Size)
		return
	}
	f, err := os.Open(p, os.O_RDONLY, 0666)
	errhandle(err)

	content := make([]byte, fi.Size)
	n, err := f.Read(content)
	errhandle(err)
	if int64(n) != fi.Size {
		panic(fmt.Sprintf("Not whole file was read, only %d from %d",
			n, fi.Size))
	}

	searchFile(p, content)

	f.Close()
}

func searchFile(p string, content []byte) {
	var linenum int
	last := 0
	hadOutput := false
	binary := false

	if bytes.IndexByte(content, 0) != -1 {
		binary = true
	}

	for _, bounds := range Pattern.FindAllIndex(content, -1) {
		if binary {
			fmt.Printf("Binary file %s matches\n", p)
			hadOutput = true
			break
		}

		if !hadOutput {
			fmt.Printf("%s:\n", p)
			hadOutput = true
		}

		linenum = bytes.Count(content[last:bounds[0]], byteNewLine)
		last = bounds[0]
		begin, end := beginend(content, bounds[0], bounds[1])

		fmt.Printf("%d:%s\n", linenum, content[begin:end])
	}

	if hadOutput {
		println()
	}
}

// Given a []byte, start and finish of some inner slice, will find nearest
// newlines on both ends of this slice
func beginend(s []byte, start int, finish int) (begin int, end int) {
	begin = 0
	end = len(s)

	for i := start; i >= 0; i-- {
		if s[i] == byteNewLine[0] {
			// skip newline itself
			begin = i + 1
			break
		}
	}

	for i := finish; i < len(s); i++ {
		if s[i] == byteNewLine[0] {
			end = i
			break
		}
	}

	return
}

func searchFiles() {
	v := &Visitor{}

	errors := make(chan os.Error, 64)

	path.Walk(".", v, errors)

	select {
	case err := <-errors:
		errhandle(err)
	default:
	}
}
