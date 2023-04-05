# Emoji List Generator

This package downloads the latest list of emojis from Unicode and parses them
into a Go map to be used my the emoji validator in the emoji package.

By default, this list is downloaded from the latest
[emoji-test.txt](https://unicode.org/Public/emoji/latest/emoji-test.txt)
provided by Unicode. According to
[UTS #51](https://www.unicode.org/reports/tr51/), this list contains all emoji
characters that should be supported by keyboards and fonts. So this list was
chosen to maximize compatibility of emojis across different systems and fonts.

## When to Update

This generator should be run for each new Unicode release, which happens
[once a year](https://unicode.org/versions/#schedule).

## Generating List
To run the generator in default mode from the repository root, run

```shell
go run ./emoji/generate/
```

## Options

The utility also supports a number of customisations that can be found by
running with the `-h` flag.

```text
go run ./emoji/generate/ -h
Downloads the emoji file (from Unicode) and parses them into a map that can be saved as a Go file or JSON file.

Usage:
  generateEmojiMap [flags]

Flags:
  -d, --delim string    The separator used between codepoints. (default " ")
  -h, --help            help for generateEmojiMap
  -j, --json string     Output file path for JSON file. Set to empty for no output.
  -l, --log string      Log output path. By default, logs are printed to stdout. To disable logging, set this to empty (""). (default "-")
  -v, --logLevel int    Verbosity level of logging. 0 = TRACE, 1 = DEBUG, 2 = INFO, 3 = WARN, 4 = ERROR, 5 = CRITICAL, 6 = FATAL (default 4)
  -o, --output string   Output file path for Go file. Set to empty for no output. (default "./emoji/data.go")
  -u, --url string      URL to download emojis from. (default "https://unicode.org/Public/emoji/latest/emoji-test.txt")

```