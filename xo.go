/*
Command xo is a command line utility that takes an input string from stdin and
formats the regexp matches.
*/
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"unicode/utf8"

	"github.com/xplshn/a-utils/pkg/ccmd"
)

func main() {
cmdInfo := &ccmd.CmdInfo{
		Name:        "xo",
		Authors:     []string{"ezekg", "xplshn"},
		Repository:  "https://github.com/xplshn/xo",
		Description: "Utility that composes regular expression matches",
		Synopsis:    "'/<pattern>/<formatter>/[flags]'",
		CustomFields: map[string]interface{}{
			"1_Examples": `Let's start off a little simple, and then we'll ramp it up and get crazy. xo, in its simplest form, does things like this,
	  \echo 'Hello! My name is C3PO, human cyborg relations.' | xo '/^(\w+)?! my name is (\w+)/$1, $2!/i'
	  \# =>
	  \#  Hello, C3PO!
    Here's a quick breakdown of what each piece of the puzzle is,
	  \echo 'Hello! My name is C3PO.' | xo '/^(\w+)?! my name is (\w+)/$1, $2!/i'
	  \^                              ^     ^^                       ^ ^     ^ ^
	  \|______________________________|     ||_______________________| |_____| |
	  \                |                    + Delimiter |                 |    + Flag
	  \                + Piped output                   + Pattern         + Formatter
    When you create a regular expression, wrapping a subexpression in parenthesis (...) creates a new capturing group, numbered from left to right in order of opening parenthesis. Submatch $0 is the match of the entire expression, submatch $1 the match of the first parenthesized subexpression, and so on. These capturing groups are what xo works with.
    What about the question mark? The question mark makes the preceding token in the regular expression optional. colou?r matches both colour and color. You can make several tokens optional by grouping them together using parentheses, and placing the question mark after the closing parenthesis, e.g. Nov(ember)? matches Nov and November.
    With that, what if the input string forgot to specify a greeting, but we, desiring to be polite, still wanted to say "Hello"? Well, that sounds like a great job for a fallback value! Let's update the example a little bit,
	  \echo 'Hello! My name is C3PO.' | xo '/^(?:(\w+)! )?my name is (\w+)/$1?:Greetings, $2!/i'
	  \# =>
	  \#  Hello, C3PO!
	  \
	  \echo 'My name is Chewbacca, uuuuuur ahhhhhrrr uhrrr ahhhrrr aaargh.' | xo '/^(?:(\w+)! )?my name is (\w+)/$1?:Greetings, $2!/i'
	  \# =>
	  \#  Greetings, Chewbacca!
    As you can see, we've taken the matches and created a new string out of them. We also supplied a fallback value for the first match ($1) that gets used if no match is found, using the elvis ?: operator.
    (The ?: inside of the regex pattern is called a non-capturing group, which is different from the elvis ?: operator in the formatter; a non-capturing group allows you to create optional character groups without capturing them into a match $i variable.)
    Now that we have the basics of xo out of the way, let's pick up the pace a little bit. Suppose we had a text file called starwars.txt containing some Star Wars quotes,
	  \Vader: If only you knew the power of the Dark Side. Obi-Wan never told you what happened to your father.
	  \Luke: He told me enough! He told me you killed him!
	  \Vader: No, I am your father.
	  \Luke: [shocked] No. No! That's not true! That's impossible!
    and we wanted to do a little formatting, as if we're telling it as a story. Easy!
	  \xo '/^(\w+):(\s*\[(.*?)\]\s*)?\s*([^\n]+)/$1 said, "$4" in a $3?:normal voice./mi' < starwars.txt
	  \# =>
	  \#   Vader said, "If only you knew the power of the Dark Side. Obi-Wan never told you what happened to your father." in a normal voice.
	  \#   Luke said, "He told me enough! He told me you killed him!" in a normal voice.
	  \#   Vader said, "No, I am your father." in a normal voice.
	  \#   Luke said, "No. No! That's not true! That's impossible!" in a shocked voice.
    Okay, okay. Let's move away from Star Wars references and on to something a little more useful. Suppose we had a configuration file called servers.yml containing some project information. Maybe it looks like this,
	  \stages:
	  \  production:
	  \    server: 192.168.1.1:1234
	  \    user: user-1
	  \  staging:
	  \    server: 192.168.1.1
	  \    user: user-2
    Now, let's say we have one of these configuration files for every project we've ever worked on. Our day to day requires us to SSH into
    these projects a lot, and having to read the config file for the IP address of the server, 
    the SSH user, as well as any potential port number gets pretty repetitive. Let's automate!
	  \xo '/.*?(production):\s*server:\s+([^:\n]+):?(\d+)?.*?user:\s+([^\n]+).*/$4@$2 -p $3?:22/mis' < servers.yml
	  \# =>
	  \#  user-1@192.168.1.1 -p 1234
	  \
	  \# Now let's actually use the output,
	  \ssh $(xo '/.*?(staging):\s*server:\s+([^:\n]+):?(\d+)?.*?user:\s+([^\n]+).*/$4@$2 -p $3?:22/mis' < servers.yml)
	  \# =>
	  \#  ssh user-2@192.168.1.1 -p 22
    Set that up as a nice ~/.shrc function, and then you're good to go:
	  \function shh() {
	  \  ssh $(xo "/.*?($1):\s*server:\s+([^:\n]+):?(\d+)?.*?user:\s+([^\n]+).*/\$4@\$2 -p \$3?:22/mis" < servers.yml)
	  \}
	  \
	  \# And then we can use it like,
	  \shh production
	  \# =>
	  \#  ssh user-1@192.168.1.1 -p 1234
    Lastly, what about reading sensitive credentials from an ignored configuration file to pass to a process, say, rails s? Let's use Stripe
    keys as an example of something we might not want to log to our terminal history,
	  \cat secrets/*.yml | xo '/test_secret_key:\s([\w]+).*?test_publishable_key:\s([\w]+)/PUBLISHABLE_KEY=$1 SECRET_KEY=$2 rails s/mis' | sh
    Pretty cool, huh?
`,
			"2_Fallback values": `You may specify fallback values for matches using the elvis operator,
    $i?:value, where i is the index that you want to assign the fallback value to.
    The fallback value may contain any sequence of characters, though anything other than letters,
    numbers, dashes and underscores must be escaped; it may also contain other match group indices
    if they are in descending order e.g. $2?:$1, not $1?:$2.`,
			"3_Delimiters": `You may substitute / for any delimiter. If the delimiter is found within your pattern or formatter, it must be escaped.
    If it would normally be escaped in your pattern or formatter, it must be escaped again. For example,
	\# Using the delimiter '|',
	\echo 'Hello! My name is C3PO, human cyborg relations.' | xo '|^(\w+)?! my name is (\w+)|$1, $2!|i'
	\
	\# Using the delimiter 'w',
	\echo 'Hello! My name is C3PO, human cyborg relations.' | xo 'w^(\\w+)?! my name is (\\w+)w$1, $2!wi'
`,
			"4_Notes":    "![Go Regular Expressions reference sheet](https://golang.org/pkg/regexp/syntax)",
		},
	}

	helpPage, err := cmdInfo.GenerateHelpPage()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error generating help page:", err)
		os.Exit(1)
	}
	flag.Usage = func() {
		fmt.Print(helpPage)
	}

	flag.Parse()

	// Check if no arguments were provided
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(0)
	}

	arg := flag.Arg(0) // Get the first argument after parsing
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		exitWithError("Nothing passed to stdin")
	}

	parts, err := split(arg)
	if err != nil {
		exitWithError("Invalid argument string")
	}
	if len(parts) <= 1 {
		exitWithError("No pattern or formatter specified")
	}
	if len(parts) > 3 {
		exitWithError("Extra delimiter detected (maybe try one other than `/`)")
	}

	pattern, format, flags := parts[0], parts[1], ""
	if len(parts) > 2 {
		flags = parts[2]
		pattern = fmt.Sprintf(`(?%s)%s`, flags, pattern)
	}

	rx, err := regexp.Compile(pattern)
	if err != nil {
		exitWithError("Invalid regular expression")
	}

	in, _ := io.ReadAll(os.Stdin)
	matches := rx.FindAllSubmatch(in, -1)
	if matches == nil {
		exitWithError("No matches found")
	}

	fallbacks := make(map[int]string)

	for _, group := range matches {
		result := format

		for i, match := range group {
			value := string(match)

			rxFallback, err := regexp.Compile(fmt.Sprintf(`(\$%d)\?:(([-_A-Za-z0-9]((\\.)+)?)+)`, i))
			if err != nil {
				exitWithError("Failed to parse default arguments", err.Error())
			}

			// Remove extraneous escapes. This is done because Go doesn't support
			// lookbehinds, i.e. `(\$%d)\?:(([-_A-za-z0-9]|(?<=\\).)+)`, so we have
			// to match escaped fallback characters using the regexp above, which
			// matches backslashes as well as the escaped character.
			rxEsc, _ := regexp.Compile(`\\(.)`)

			fallback := rxFallback.FindStringSubmatch(result)
			if len(fallback) > 1 {
				// Store fallback values if key does not already exist
				if _, ok := fallbacks[i]; !ok {
					fallbacks[i] = rxEsc.ReplaceAllString(fallback[2], "$1")
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

		fmt.Println(result)
	}
}

// split slices str into all substrings separated by non-escaped values of the
// first rune and returns a slice of those substrings.
// It removes one backslash escape from any escaped delimiters.
func split(str string) ([]string, error) {
	if !utf8.ValidString(str) {
		return nil, errors.New("Invalid string")
	}

	// Grab the first rune.
	delim, size := utf8.DecodeRuneInString(str)
	str = str[size:]

	var (
		subs   []string
		buffer bytes.Buffer
	)
	for len(str) > 0 {
		r, size := utf8.DecodeRuneInString(str)
		str = str[size:]

		if r == '\\' {
			peek, peekSize := utf8.DecodeRuneInString(str)
			if peek == delim {
				buffer.WriteRune(peek)
				str = str[peekSize:]
				continue
			}
		}

		if r == delim {
			if buffer.Len() > 0 {
				subs = append(subs, buffer.String())
				buffer.Reset()
			}
			continue
		}

		buffer.WriteRune(r)
	}
	if buffer.Len() > 0 {
		subs = append(subs, buffer.String())
	}
	return subs, nil
}

// exitWithError prints a bunch of strings and then exits with a non-zero exit code.
func exitWithError(errs ...string) {
	for _, err := range errs {
		fmt.Println(err)
	}
	os.Exit(1)
}
