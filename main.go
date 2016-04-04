// 3 april 2016
package main

import (
	"fmt"
	"os"
//	"io"
	"bytes"
	"strings"
	"net/mail"
	"mime"
//	"mime/multipart"
	"encoding/base64"

	"github.com/mxk/go-imap/imap"
"encoding/hex"
"github.com/davecgh/go-spew/spew"
)

// TODOs:
// - go from panic to errors
// - componentize all this
// - options: -f to limit to a folder, -s for SSL

var TODO_remove_this = spew.Config
var TODO_remove_this_too = hex.Dump

func toword(what string) string {
	return base64.StdEncoding.EncodeToString([]byte(what))
}

func fromword(word string) string {
	b, err := base64.StdEncoding.DecodeString(word)
	if err != nil {
		panic(err)
	}
	return string(b)
}

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

func extract(raw *imap.Response) (uid uint32, msg *mail.Message, bodyStructure []imap.Field) {
	info := raw.MessageInfo()
	headerbytes := imap.AsBytes(info.Attrs["RFC822.HEADER"])
	if len(headerbytes) == 0 {
		panic("no header in message")
	}
	msg, err := mail.ReadMessage(bytes.NewReader(headerbytes))
	if err != nil {
// TODO debug
// TODO show tuple here
fmt.Fprintf(os.Stderr, "skipping invalid message\n")
return 0, nil, nil
		panic(err)
	}
	bodyStructure = imap.AsList(info.Attrs["BODYSTRUCTURE"])
	return info.UID, msg, bodyStructure
}

func canHaveAttachment(msg *mail.Message) bool {
	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		// assume emails without a Content-Type are text/plain
		// yeah, I have one from 2009 and I *think* it's text/plain...
		return false
	}
	mimetype, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		panic(err)
	}
	// this is valid; thanks to kurushiyama in irc.freenode.net/#go-nuts
	return strings.HasPrefix(mimetype, "multipart/")
}

func process(path string, raw *imap.Response) {
	uid, msg, bodyStructure := extract(raw)
	if msg == nil {
		return
	}
fmt.Fprintf(os.Stderr, "%s %q %q\n", path, msg.Header.Get("From"), msg.Header.Get("Subject"))
	if !canHaveAttachment(msg) {
		return
	}
	for _, part := range bodyStructure {
		// a string will always be after the last multipart file
		if imap.TypeOf(part) & imap.List == 0 {
			break
		}
		mime := imap.AsList(part)
		// TODO will it always be at position 2?
		ext := imap.AsFieldMap(mime[2])
		// TODO wil it always be capitalized?
		filename := ext["NAME"]
_=filename
	}
_=uid
/*
		if filename != "" {
			fmt.Printf("%s %d %s | folder:%q filename:%q from:%s subject:%q contentType:%q\n",
				// actual fields
				toword(path),
				uid,
				toword(filename),
				// comments
				path,
				filename,
				msg.Header.Get("From"),
				msg.Header.Get("Subject"),
				part.Header.Get("Content-Type"))
*/
}

const fetchSize = 100

func fetch(c *imap.Client, path string, first uint32, last uint32) {
	seq, err := imap.NewSeqSet(fmt.Sprintf("%d:%d", first, last))
	if err != nil {
		panic(err)
	}
	list := start(c.Fetch(seq, "UID RFC822.HEADER BODYSTRUCTURE"))
	for list.InProgress() {
		err = c.Recv(-1)
		if err != nil {
			panic(err)
		}
		for _, msg := range list.Data {
			process(path, msg)
		}
		list.Data = nil
		// TODO why is this needed?
		c.Data = nil
	}
	finish(list)
}

func search(c *imap.Client, path string, indent int) {
	handle(c.Select(path, true))
	defer func() {		// closure needed as otherwise c.Close() will be run immediately
//TODO		handle(c.Close(false))
		cmd, _ := c.Close(false)
		cmd.Result(imap.OK)
	}()

	// TODO why is this needed?
	c.Data = nil

	n := c.Mailbox.Messages

	// If there are no messages, the fetch will fail
	if n == 0 {
		return
	}

	for first := uint32(1); first <= n; first += fetchSize {
		last := first + fetchSize
		if last > n {
			last = n
		}
		fetch(c, path, first, last)
	}
}

// TODO does this actually handle multiple directory structures correctly? it seems to given how name works
func runList(c *imap.Client, list *imap.Command, indent int) {
	for _, r := range list.Data {
		name := r.MailboxInfo().Name
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

	tree(c)

	// DO NOT DEFER THIS
	// doing so will cause any panics to cascade
	// we can defer this if we move away from panic() though
	c.Logout(-1)
}
