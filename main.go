/*
Command xo is a command line utility similar to `sed`, except it has one job: take an
input string from `stdin` and format the matches.
*/
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) == 1 {
		throwError("No arguments were passed")
	}

	arg := os.Args[1]
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		throwError("Nothing passed to stdin")
	}

	delimiter := string(arg[len(arg)-1])
	parts := delEmpty(strings.Split(arg, delimiter))
	if len(parts) <= 1 {
		throwError("No pattern or formatter specified")
	}

	var (
		mods    string
		pattern string
		format  string
	)

	if len(parts) > 2 {
		mods = parts[0]
		pattern = parts[1]
		format = parts[2]
	} else {
		pattern = parts[0]
		format = parts[1]
	}

	rx, err := regexp.Compile(fmt.Sprintf(`(?%s)%s`, mods, pattern))
	if err != nil {
		throwError("Invalid regular expression", err.Error())
	}

	stdin, _ := ioutil.ReadAll(os.Stdin)
	matches := rx.FindAllSubmatch(stdin, -1)
	if matches == nil {
		throwError("No matches found")
	}

	var (
		fallbacks = make(map[int]string)
	)

	for _, group := range matches {
		result := format

		for i, match := range group {
			value := string(match)

			rxFallback, err := regexp.Compile(fmt.Sprintf(`(\$%d)\?:([^\s\n]+)`, i))
			if err != nil {
				throwError("Failed to parse default arguments", err.Error())
			}

			fallback := rxFallback.FindStringSubmatch(result)
			if len(fallback) > 1 {
				// Store fallback values if key does not already exist
				if _, ok := fallbacks[i]; !ok {
					fallbacks[i] = string(fallback[2])
				}
				result = rxFallback.ReplaceAllString(result, "$1")
			}

			// Set default for empty values
			if value == "" {
				value = fallbacks[i]
			}

			// Replace values
			rxRepl, _ := regexp.Compile(fmt.Sprintf(`\$%d`, i))
			result = rxRepl.ReplaceAllString(result, value)
		}

		fmt.Printf("%s\n", result)
	}
}

func delEmpty(strs []string) []string {
	var result []string

	for _, str := range strs {
		if str != "" {
			result = append(result, str)
		}
	}

	return result
}

func throwError(errs ...string) {
	for _, err := range errs {
		fmt.Printf("%s\n", err)
	}
	os.Exit(1)
}
