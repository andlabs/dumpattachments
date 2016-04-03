// 3 april 2016
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/mxk/go-imap/imap"
)

func start(cmd *imap.Command, err error) *imap.Command {
	if err != nil {
		panic(err)
	}
	return cmd
}

func finish(cmd *imap.Command) *imap.Command {
	_, err := cmd.Result(imap.OK)
	if err != nil {
		panic(err)
	}
	// TODO figure out why we need this
	cmd.Client().Data = nil
	return cmd
}

func handle(cmd *imap.Command, err error) *imap.Command {
	cmd, err = imap.Wait(cmd, err)
	if err != nil {
		panic(err)
	}
	return finish(cmd)
}

func search(c *imap.Client, path string, indent int) {
	handle(c.Select(path, true))
	defer func() {		// closure needed as otherwise c.Close() will be run immediately
		handle(c.Close(false))
	}()

	// TODO why is this needed?
	c.Data = nil

	// If there are no messages, the fetch will fail
	if c.Mailbox.Messages == 0 {
		return
	}

	seq, err := imap.NewSeqSet("1:*")
	if err != nil {
		panic(err)
	}
	list := start(c.Fetch(seq, "RFC822.HEADER"))
	for list.InProgress() {
		err = c.Recv(-1)
		if err != nil {
			panic(err)
		}
/*		for _, msg := range list.Data {
			fmt.Printf("%s%s\n", strings.Repeat(" ", indent),
				msg.Type)
		}
*/		list.Data = nil
		// TODO why is this needed?
		c.Data = nil
	}
	finish(list)
}

// TODO does this actually handle multiple directory structures correctly? it seems to given how name works
func runList(c *imap.Client, list *imap.Command, indent int) {
	for _, r := range list.Data {
		name := r.MailboxInfo().Name
		fmt.Printf("[DIR] %s%s\n", strings.Repeat(" ", indent), name)
		search(c, name, indent + 1)
		list2 := handle(imap.Wait(c.List(name, "%")))
		runList(c, list2, indent + 1)
	}
}

func tree(c *imap.Client) {
	list := handle(imap.Wait(c.List("", "%")))
	runList(c, list, 0)
}

func main() {
//	imap.DefaultLogMask = imap.LogAll

	server := os.Args[1]
	user := os.Args[2]
	pass := os.Args[3]

	c, err := imap.DialTLS(server, nil)
	if err != nil {
		panic(err)
	}
	if c.Caps["STARTTLS"] {
		handle(c.StartTLS(nil))
	}
	if c.Caps["ID"] {
		handle(c.ID("name", "dumpattachments"))
	}
	handle(c.Noop())
	handle(c.Login(user, pass))
	defer c.Logout(-1)

	tree(c)
}
