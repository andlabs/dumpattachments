// 3 april 2016
package main

import (
	"fmt"
	"os"
	"flag"
	"encoding/hex"

	"github.com/mxk/go-imap/imap"
)

// TODOs:
// - options: -f to limit to a folder, -s for SSL
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
	errf("  rawdump mailbox uidverify uid")
	errf("	hex dump the raw header and body of the given email")
}

func notenoughargs(command string) {
	errf("%s: not enough arguments for command %s", os.Args[0], command)
	usage()
	os.Exit(2)
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
	cmdargs := args[4:]

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
	case "rawdump":
		if len(cmdargs) != 3 {
			notenoughargs(command)
		}
		tuple, err := MsgTupleFromArgs(cmdargs)
		if err != nil {
			errf("%s: %s: invalid message tuple: %v", os.Args[0], command, err)
			os.Exit(1)
		}
		header, body, err := c.RawMessage(tuple)
		if err != nil {
			errf("%s: %s: error getting message: %v", os.Args[0], command, err)
			os.Exit(1)
		}
		fmt.Printf("header:\n")
		fmt.Print(hex.Dump(header))
		fmt.Printf("body:\n")
		fmt.Print(hex.Dump(body))
	default:
		errf("%s: unknown command %q", os.Args[0], command)
		usage()
		os.Exit(2)
	}
}
