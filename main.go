// 3 april 2016
package main

import (
	"fmt"
	"os"
	"flag"

	"github.com/mxk/go-imap/imap"
)

// TODOs:
// - options: -f to limit to a folder, -s for SSL
// - command to raw dump message header and body for diagnosing bad messages
// - command to dump text or html part of message to verify this is what you want

var (
	imapdebug = flag.Bool("imapdebug", false, "debug IMAP session")
)

// TODO change other files to use this
func errf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	fmt.Fprintf(os.Stderr, "\n")
}

func usage() {
	errf("usage: %s [options] server user pass command [args...]", os.Args[0])
	errf("options:")
	flag.PrintDefaults()
	errf("commands:")
	errf("  list")
	errf("	print a list of attachments to stdout")
}

func main() {
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) < 4 {
		usage()
		os.Exit(2)
	}

	server := args[0]
	user := args[1]
	pass := args[2]
	command := args[3]
//	cmdargs := args[4:]

	if *imapdebug {
		imap.DefaultLogMask = imap.LogAll
	}

	// TODO move this after the command check somehow
	c, err := NewConn(server, user, pass, true)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	switch command {
	case "list":
		err = ListLines(c)
		if err != nil {
			panic(err)
		}
	default:
		errf("%s: unknown command %q", os.Args[0], command)
		usage()
		os.Exit(2)
	}
}
