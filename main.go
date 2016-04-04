// 3 april 2016
package main

import (
	"fmt"
	"os"
)

// TODOs:
// - options: -f to limit to a folder, -s for SSL

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

	list, err := ListLines(c)
	if err != nil {
		panic(err)
	}
	fmt.Print(list)
}
