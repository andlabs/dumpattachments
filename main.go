// 3 april 2016
package main

import (
	"os"
)

// TODOs:
// - options: -f to limit to a folder, -s for SSL
// - command to raw dump message header and body for diagnosing bad messages
// - command to dump text or html part of message to verify this is what you want

func main() {
//	imap.DefaultLogMask = imap.LogAll

	server := os.Args[1]
	user := os.Args[2]
	pass := os.Args[3]

	c, err := NewConn(server, user, pass, true)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	err = ListLines(c)
	if err != nil {
		panic(err)
	}
}
