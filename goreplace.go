// (c) 2011-2014 Alexander Solovyov
// under terms of ISC license

package main

import (
	"bytes"
	"fmt"
	flags "github.com/jessevdk/go-flags"
	byten "github.com/pyk/byten"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"unicode"
)

const (
	Author  = "Alexander Solovyov"
	Version = "2.4"
)

var byteNewLine = []byte("\n")
var NoColors = false

var opts struct {
	Replace         *string  `short:"r" long:"replace" description:"replace found substrings with RE" value-name:"RE"`
	Whole           bool     `short:"w" long:"whole" description:"only replace whole-word matches"`
	Dry             bool     `short:"d" long:"dry-run" description:"do a dry run, which gives same output but doesn't change any files"`
	Show            bool     `short:"s" long:"show" description:"shows a before/after for each matching line"`
	UnquoteReplace  bool     `short:"u" long:"unqoute" description:"applies Go unquote rules to replacement string"`
	Ask             bool     `short:"a" long:"ask" description:"ask whether to do each and every replacement"`
	Force           bool     `short:""  long:"force" description:"force replacement in binary files"`
	IgnoreCase      bool     `short:"i" long:"ignore-case" description:"ignore pattern case"`
	PlainText       bool     `short:"p" long:"plain" description:"treat pattern as plain text"`
	IgnoreFiles     []string `short:"x" long:"exclude" description:"exclude filenames that match regexp RE (multi)" value-name:"RE"`
	AcceptFiles     []string `short:"o" long:"only" description:"search only filenames that match regexp RE (multi)" value-name:"RE"`
	NoGlobalIgnores bool     `short:"I" long:"no-autoignore" description:"do not read .git/.hgignore files"`
	FindFiles       bool     `short:"f" long:"find-files" description:"search in file names"`
	OnlyName        bool     `short:"n" long:"filename" description:"print only filenames"`
	Verbose         bool     `short:"v" long:"verbose" description:"show non-fatal errors (like unreadable files)"`
	NoColors        bool     `short:"c" long:"no-colors" description:"do not show colors in output"`
	Group           bool     `short:"g" long:"group" description:"print file name before each line"`
	ShowVersion     bool     `short:"V" long:"version" description:"show version and exit"`
	ShowHelp        bool     `short:"h" long:"help" description:"show this help message"`
}

var argparser = flags.NewParser(&opts, flags.PrintErrors|flags.PassDoubleDash)

var printer *Printer

func main() {
	args, err := argparser.Parse()
	if err != nil {
		os.Exit(1)
	}

	/* this fixes an apparent bug in the flags package which strips off all double quotes that
	appear for a string argument */
	for i, s := range os.Args {
		switch {
		case strings.HasPrefix(s, "-r="):
			r := s[3:]
			opts.Replace = &r
		case strings.HasPrefix(s, "--replace="):
			r := s[10:]
			opts.Replace = &r
		case s == "-r" || s == "--replace":
			opts.Replace = &os.Args[i+1]
		}
	}

	if opts.ShowVersion {
		fmt.Printf("goreplace %s\n", Version)
		return
	}

	NoColors = opts.NoColors || runtime.GOOS == "windows"

	printer = &Printer{NoColors, ""}

	cwd, _ := os.Getwd()
	ignoreFileMatcher := NewMatcher(cwd, opts.NoGlobalIgnores)
	ignoreFileMatcher.Append(opts.IgnoreFiles)

	acceptedFileMatcher := NewGeneralMatcher([]string{}, []string{})
	if len(opts.AcceptFiles) > 0 {
		acceptedFileMatcher.Append(opts.AcceptFiles)
	} else {
		acceptedFileMatcher.Append([]string{".*"})
	}

	argparser.Usage = fmt.Sprintf("[OPTIONS] string-to-search\n\n%s",
		ignoreFileMatcher)

	if opts.ShowHelp || len(args) == 0 {
		argparser.WriteHelp(os.Stdout)
		return
	}

	var plain []byte
	arg := args[0]
	if opts.Whole {
		opts.PlainText = true;
	}
	if opts.PlainText {
		plain = []byte(arg)
		arg = regexp.QuoteMeta(arg)
	}
	if opts.IgnoreCase {
		if opts.PlainText && opts.Replace != nil {
			errhandle(fmt.Errorf("Cannot combine --plain with --ignore-case when replacing"), true)
		}
		arg = "(?i:" + arg + ")"
	}

	pattern, err := regexp.Compile(arg)
	errhandle(err, true)

	if pattern.Match([]byte("")) {
		errhandle(fmt.Errorf("Your pattern matches the empty string"), true)
	}

	var replace []byte

	if opts.Replace != nil {
		if opts.UnquoteReplace {
			s, err := strconv.Unquote(`"` + *opts.Replace + `"`)
			if err != nil {
				errhandle(err, true)
			}
			replace = []byte(s)
		} else {
			replace = []byte(*opts.Replace)
		}
		if len(replace) == 0 {
			if !ask("Replacement is empty, all instances of pattern will be deleted. Are you sure?") {
				os.Exit(0)
			}
		}
	}

	if opts.Ask {
		opts.Show = true
		if opts.Dry {
			printer.Printf("note: --ask is ignored for --dry-run\n")
		}
	}

	if opts.Dry {
		printer.Printf("@RDRY RUN@|\n")
	}

	if opts.Ask && replace == nil {
		errhandle(fmt.Errorf("--ask doesn't make sense unless a replacement is given by --replace"), true)
	}

	if opts.Dry && replace == nil {
		errhandle(fmt.Errorf("--dry-run doesn't make sense unless a replacement is given by --replace"), true)
	}

	searchFiles(pattern, plain, replace, ignoreFileMatcher, acceptedFileMatcher, opts.Whole)
	if opts.Dry {
		printer.Printf("@RDRY RUN@|\n")
	}
}

func ask(prompt string) bool {
	if opts.Dry {
		return true
	}
	for {
		printer.Printf("%s @!y@|es, @!n@|o, @!a@|bort\n", prompt)
		c, _, _ := GetChar()
		switch c {
		case 'y', 13:
			return true
		case 'n':
			return false
		case 'a', 'q', 27, 3:
			os.Exit(0)
		}
	}
}

func errhandle(err error, exit bool) bool {
	if err == nil {
		return false
	}
	fmt.Fprintf(os.Stderr, "%s\n", err)
	if exit {
		os.Exit(1)
	}
	return true
}

func searchFiles(pattern *regexp.Regexp, plainPattern []byte, replace []byte, ignoreFileMatcher Matcher,
	acceptedFileMatcher Matcher, whole bool) {

	v := &GRVisitor{pattern, plainPattern, replace, ignoreFileMatcher, acceptedFileMatcher, whole}

	err := filepath.Walk(".", v.Walk)
	errhandle(err, false)
}

type GRVisitor struct {
	pattern             *regexp.Regexp
	plainPattern        []byte
	replace             []byte
	ignoreFileMatcher   Matcher
	acceptedFileMatcher Matcher
	whole               bool
	// errors              chan error
}

func (v *GRVisitor) Walk(fn string, fi os.FileInfo, err error) error {
	if err != nil {
		if opts.Verbose {
			errhandle(err, false)
		}
		return nil
	}

	if fi.IsDir() {
		if !v.VisitDir(fn, fi) {
			return filepath.SkipDir
		}
		return nil
	}

	v.VisitFile(fn, fi)
	return nil
}

func (v *GRVisitor) VisitDir(fn string, fi os.FileInfo) bool {
	return !v.ignoreFileMatcher.Match(fn, true)
}

func (v *GRVisitor) VisitFile(fn string, fi os.FileInfo) {
	if fi.Size() == 0 && !opts.FindFiles {
		return
	}

	if v.ignoreFileMatcher.Match(fn, false) {
		return
	}

	if !v.acceptedFileMatcher.Match(fn, false) {
		return
	}

	if opts.FindFiles {
		v.SearchFileName(fn)
		return
	}

	if fi.Size() >= 1024*1024*10 {
		errhandle(fmt.Errorf("Skipping %s, too big: %s\n", fn, byten.Size(fi.Size())),
			false)
		return
	}

	// just skip invalid symlinks
	if fi.Mode()&os.ModeSymlink != 0 {
		if _, err := os.Stat(fn); err != nil {
			if opts.Verbose {
				errhandle(err, false)
			}
			return
		}
	}

	f, content := v.GetFileAndContent(fn, fi)
	if f == nil {
		return
	}
	defer f.Close()

	changed, result := v.ReplaceInFile(fn, content)
	if changed {
		f.Seek(0, 0)
		n, err := f.Write(result)
		if err != nil {
			errhandle(fmt.Errorf("Error writing replacement to file '%s': %s",
				fn, err), true)
		}
		if int64(n) < fi.Size() {
			err := f.Truncate(int64(n))
			if err != nil {
				errhandle(fmt.Errorf("Error truncating file '%s' to size %d",
					f, n), true)
			}
		}
	}
}

func (v *GRVisitor) GetFileAndContent(fn string, fi os.FileInfo) (f *os.File, content []byte) {
	var err error

	if v.replace != nil && !opts.Dry {
		f, err = os.OpenFile(fn, os.O_RDWR, 0666)
	} else {
		f, err = os.Open(fn)
	}

	if err != nil {
		if opts.Verbose {
			errhandle(err, false)
		}
		return
	}

	content = make([]byte, fi.Size())
	n, err := f.Read(content)
	if err != nil {
		errhandle(fmt.Errorf("Error %s", err), false)
		return
	}
	if int64(n) != fi.Size() {
		errhandle(fmt.Errorf("Not whole file '%s' was read, only %d from %d",
			fn, n, fi.Size()), true)
	}

	return
}

func (v *GRVisitor) SearchFileName(fn string) {
	if !v.pattern.MatchString(fn) {
		return
	}
	colored := v.pattern.ReplaceAllStringFunc(fn,
		func(wrap string) string {
			return printer.Sprintf("@Y%s", wrap)
		})
	fmt.Println(colored)
}

func getSuffix(num int) string {
	if num != 1 {
		return "s"
	}
	return ""
}

func wordchar(c byte) bool {
	return unicode.IsLetter(rune(c)) || unicode.IsNumber(rune(c)) || c == '$';
}

func findWholeSkips(content []byte, pos [][]int) []bool {
	ln := len(content)
	skips := make([]bool, len(pos))
	for i, p := range pos {
		pl := p[0]
		pr := p[1]
		skip := (pl > 0 && wordchar(content[pl-1])) || (pr < ln && wordchar(content[pr]))
		skips[i] = skip
	}
	return skips
}

func (v *GRVisitor) ReplaceInFile(fn string, content []byte) (changed bool, result []byte) {
	norep := v.replace == nil
	changenum := 0
	skippednum := 0
	binary := bytes.IndexByte(content, 0) != -1

	var has_match bool
	if v.plainPattern != nil {
		has_match = bytes.Index(content, v.plainPattern) != -1
	} else {
		has_match = v.pattern.Find(content) != nil
	}

	if !has_match {
		return false, nil
	}

	if binary && !opts.Force {
		errhandle(
			fmt.Errorf("%s: at least one match detected in binary file, supply --force to force change of binary file", fn),
			false)
		return false, nil
	}

	show := opts.Show && !binary

	var matchl, matchr, linel, liner []int
	var skips []bool
	var linenum []int
	if show || norep {
		pos := v.pattern.FindAllIndex(content, -1)
		ln := len(content)
		matchl = make([]int, len(pos))
		matchr = make([]int, len(pos))
		linel = make([]int, len(pos))
		liner = make([]int, len(pos))
		linenum = make([]int, len(pos))
		for i, p := range pos {
			pl := p[0]
			pr := p[1]
			matchl[i] = pl
			matchr[i] = pr
			linenum[i] = bytes.Count(content[:pl], []byte{'\n'}) + 1
			for content[pl] != '\n' && pl > 0 {
				pl--
			}
			for content[pr] != '\n' && pr < (ln-1) {
				pr++
			}
			linel[i] = pl
			liner[i] = pr
		}
		if v.whole {
			skips = findWholeSkips(content, pos)
		}
	} else if v.whole {
		pos := v.pattern.FindAllIndex(content, -1)
		skips = findWholeSkips(content, pos)
	}
	if skips != nil {
		any := false
		for _, b := range skips {
			if !b { 
				any = true
				break
			}
		}
		if !any {
			return false, nil
		}
	}
	if opts.Group && (show || norep) {
		printer.Printf("@g%s@|\n", fn)
	}
	if norep {
		for i, line := range linenum {
			if skips != nil && skips[i] {
				continue
			}
			prefix := bytes.TrimLeft(content[linel[i]:matchl[i]], "\n")
			postfix := bytes.TrimRight(content[matchr[i]:liner[i]], "\n")
			str := content[matchl[i]:matchr[i]]
			if opts.Group {
				printer.Printf("@y%d:@| ", line)
			} else {
				printer.Printf("@g%s:%d:@| ", fn, line)
			}
			printer.Printf("%s@{Yk}%s@|%s\n", prefix, str, postfix)
		}
		return false, nil
	}
	i := 0
	result = v.pattern.ReplaceAllFunc(content, func(str []byte) (res []byte) {
		if skips != nil && skips[i] {
			i += 1
			return str
		}
		if v.plainPattern != nil {
			res = bytes.Replace(str, v.plainPattern, v.replace, 1)
		} else {
			res = v.pattern.ReplaceAll(str, v.replace)
		}
		if show {
			prefix := bytes.TrimLeft(content[linel[i]:matchl[i]], "\n")
			postfix := bytes.TrimRight(content[matchr[i]:liner[i]], "\n")
			var loc0, loc1, loc2 string
			if opts.Group {
				loc0 = ""
				loc1 = fmt.Sprintf("%d: ", linenum[i])
				loc2 = strings.Repeat(" ", len(loc1))
			} else {
				loc0 = fmt.Sprintf("%s:%d\n", fn, linenum[i])
				loc1 = ""
				loc2 = ""
			}
			printer.Printf("@g%s@y%s@|%s@{Yk}%s@|%s\n@y%s@|%s@{Ck}%s@|%s\n",
				loc0, loc1, prefix, str, postfix, loc2, prefix, res, postfix)
			i += 1
			if opts.Ask {
				if ask("apply replacement?") {
					changenum += 1
					return
				} else {
					skippednum += 1
					return str
				}
			}
		}
		changenum += 1
		return
	})

	if binary && opts.Ask && !ask(fmt.Sprintf("%s: replace %d matches in binary file?", fn, changenum)) {
		return false, nil
	}

	if changenum > 0 || skippednum > 0 {
		if !show {
			printer.Printf("@g%s@|: ", fn)
		}
		if skippednum > 0 {
			printer.Printf("@m%d change%s made, %d skipped\n", changenum, getSuffix(changenum), skippednum)
		} else {
			printer.Printf("@m%d change%s made\n", changenum, getSuffix(changenum))
		}
		if show {
			println()
		}
	}

	if opts.Dry {
		return false, nil
	}
	return true, result
}
