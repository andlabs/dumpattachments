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
// - replace panics in all files

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
	errf("  dump")
	errf("	dump all the attachments listed on stdin to the current directory")
	errf("  rawdump mailbox uidverify uid")
	errf("	hex dump the raw header and body of the given email")
	errf("  read mailbox uidverify uid")
	errf("	writes the first text/plain or text/html part of the email to stdout;")
	errf("	does not mark as read (useful for seeing if this is the email you want)")
}

func notenoughargs(command string) {
	errf("%s: not enough arguments for command %s", os.Args[0], command)
	usage()
	os.Exit(2)
}

func doRaw(f func(header []byte, body []byte), server string, user string, pass string, command string, cmdargs []string) {
	if len(cmdargs) != 3 {
		notenoughargs(command)
	}
	tuple, err := MsgTupleFromArgs(cmdargs)
	if err != nil {
		errf("%s: %s: invalid message tuple: %v", os.Args[0], command, err)
		os.Exit(1)
	}
	do(func(c *Conn) {
		header, body, err := c.RawMessage(tuple)
		if err != nil {
			errf("%s: %s: error getting message: %v", os.Args[0], command, err)
			os.Exit(1)
		}
		f(header, body)
	}, server, user, pass)
}

func do(f func(c *Conn), server string, user string, pass string) {
	c, err := NewConn(server, user, pass, true)
	if err != nil {
		panic(err)
	}
	defer c.Close()
	f(c)
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

	switch command {
	case "list":
		do(func(c *Conn) {
			err := ListLines(c)
			if err != nil {
				panic(err)
			}
		}, server, user, pass)
	case "dump":
		do(doDump, server, user, pass)
	case "rawdump":
		doRaw(func(header []byte, body []byte) {
			fmt.Printf("header:\n")
			fmt.Print(hex.Dump(header))
			fmt.Printf("body:\n")
			fmt.Print(hex.Dump(body))
		}, server, user, pass, command, cmdargs)
	case "read":
		doRaw(func(header []byte, body []byte) {
			m, err := MultipartFromRaw(header, body)
			if err != nil {
				panic(err)
			}
			for m.Next() {
				p := m.Part()
				if p.ContentType == "text/plain" || p.ContentType == "text/html" {
					os.Stdout.Write(p.Contents)
					fmt.Println()			// just in case
					break
				}
			}
			if err := m.Err(); err != nil {
				panic(err)
			}
			// TODO alert that nothing was found
		}, server, user, pass, command, cmdargs)
	default:
		errf("%s: unknown command %q", os.Args[0], command)
		usage()
		os.Exit(2)
	}
}
