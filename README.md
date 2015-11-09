# Go Replace

Go Replace (gr) is a simple utility which can be used as replacement for grep +
sed combination in one of most popular cases - find files, which contain
something, possibly replace this with something else. Main points:

 - Reads `.hgignore`/`.gitignore` to skip files
 - Skips binaries
 - Familiar PCRE-like regexp syntax
 - Can perform replacements
 - Fast

Bonus:

 - Can search in file names with `-f` (i.e. a simple alternative to `find`)

[![Build Status](https://travis-ci.org/piranha/goreplace.png)](https://travis-ci.org/piranha/goreplace)

[Releases and changelog](https://github.com/piranha/goreplace/releases)

## Why

Why do thing which is done by grep, find, and sed? Well, for one - I grew tired
of typing long commands with pipes and ugly syntax. You want to search? Use
grep. Replace? Use find and sed! Different syntax, context switching,
etc. Switching from searching to replacing with gr is 'up one item in
history and add a replacement string', much simpler!

Besides, it's also faster than grep! Hard to believe, and it's a bit of cheating -
but gr by default ignores everything you have in your `.hgignore` and
`.gitignore` files, skipping binary files and compiled bytecodes (which you
usually don't want to touch anyway).

This is my reason to use it - less latency doing task I'm doing often.

## Installation

Just download a suitable binary from
[release page](https://github.com/piranha/goreplace/releases). Put this file in
your `$PATH` and rename it to `gr` to have easier access.

### Building from source

You can also install it from source, if that's your thing:

    go get github.com/piranha/goreplace

And you should be done. You have to have `$GOPATH` set for this to work (`go`
will put sources and generated binary there). Add `-u` flag there to update your
`gr`.

I prefer name `gr` to `goreplace`, so I link `gr` somewhere in my path (usually
in `~/bin`) to `$GOPATH/bin/goreplace`. **NOTE**: if you use `oh-my-zsh`, it
aliases `gr` to `git remote`, so you either should use another name (I propose
`gor`) or remove `gr` alias:

```
mkdir -p ~/.oh-my-zsh/custom && echo "unalias gr" >> ~/.oh-my-zsh/custom/goreplace.zsh
```

## Usage

Usage is pretty simple, you can just run `gr` to see help on options. Basically
you just supply a regexp (or a simple string - it's a regexp always as well) as
an argument and gr will search for it in all files starting from the
current directory, just like this:

    gr somestring

Some directories and files can be ignored by default (`gr` is looking for your
`.hgignore`/`.gitignore` in parent directories), just run `gr` without any
arguments to see help message - it contains information about them.

And to replace:

    gr somestring -r replacement

It's performed in place and no backups are made (not that you need them, right?
You're using version control, aren't you?). Regular expression submatches
supported via `$1` syntax - see
[re2 documentation](https://code.google.com/p/re2/wiki/Syntax) for more
information about syntax and capabilities.

## About this fork

This fork introduces a couple useful features, and discards one feature that 
arguably isn't that useful. The full list of changes, roughly in decreasing order of usefulness, is:

1. `--ask`, which produces interactive prompts to allow the user to accept or
reject each candidate replacement, where the context is shown (the line containing the match, before and after replacement, with the contained replacement highlighted when the terminal is in color mode).
This feature is Mac/Linux only.
2. `--dry-run`, which gives the same output as a real invocation, but doesn't
actually change any files. 
3. `--show`, which displays the context of each match (specifically the line containing the match). In "grep mode", where no replacement is given, this produces exactly one line per match, with the match highlighted on color termianls. In ordinary replace mode, this shows the line before and after replacement. `--ask` implies `--show`. `--group` is respected by `--show`.
4. `--plain` now works in conjunction with replace mode, previously it was disabled, presumably out of fear that `$1` etc would still be expanded in the replacement string even though the search regexp had all metacharacters escaped.
5. The replacement string no longer has double quotes, etc, automatically stripped off it, which was apparently just a quirk of the flag-parsing library and let to many infuriatingly irreversible replacements in my experience.
6. `--unquote`, which makes the default beavior of applying Golang string unquote rules to the replacement string optional rather than mandatory.
7. `--group`, i.e. grouping behavior is now opt-in rather than opt-out. This more a personal choice of mine. The old code that did grouping 'semi-magically' via Printer is now retired, it only got used for "grep mode" anyway.
8. `--singeline` is gone, because it didn't work in replace mode anyway, and goreplace will never be as rich or fast as "ag" anyway.
9. Replacing with an empty string checks first with the user that they meant to do that -- I've somehow made this mistake a bunch of times before, and it's particularly hard to recover from if you're in the middle of staging changes.

In general the codebase could do with a little simplification, in particular the way that different flags combine and their edge cases is a bit hard to understand and there are probably some bugs there. Also, I could have re-used some existing code but I was in a rush and didn't see it until it was too late.

Anyway, these changes make me much more confident in using `goreplace` on larger codebases, doing more ambitious replacements, especially thanks to `--dry-run` and `--ask` features.

-- @taliesinb