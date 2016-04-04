// 4 april 2016
package main

import (
	"github.com/mxk/go-imap/imap"
)

type Conn struct {
	c	*imap.Client
}

func (c *Conn) handle(cmd *imap.Command, err error) error {
	if err != nil {
		return err
	}
	_, err = cmd.Result(imap.OK)
	if err != nil {
		return err
	}
	// TODO why is this needed?
	c.c.Data = nil
	return nil
}

func NewConn(server string, user string, pass string, ssl bool) (c *Conn, err error) {
	c = new(Conn)

	if ssl {
		c.c, err = imap.DialTLS(server, nil)
	} else {
		c.c, err = imap.Dial(server)
	}
	if err != nil {
		return nil, err
	}

	if c.c.Caps["STARTTLS"] {
		err = c.handle(c.c.StartTLS(nil))
		if err != nil {
			return nil, err
		}
	}
	if c.c.Caps["ID"] {
		err = c.handle(c.c.ID("name", "dumpattachments"))
		if err != nil {
			return nil, err
		}
	}

	err = c.handle(c.c.Noop())
	if err != nil {
		return nil, err
	}
	err = c.handle(c.c.Login(user, pass))
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Conn) Close() error {
	return c.handle(c.c.Logout(-1))
}

// TODO if we're using imap.Wait() we can just do away with the caching and go depth first
func (c *Conn) gatherFolders(root string) ([]string, error) {
	// first gather everything at root
	cmd, err := imap.Wait(c.c.List(root, "%"))
	if err != nil {
		return nil, err
	}
	subs := make([]string, 0, len(cmd.Data))
	for _, sub := range cmd.Data {
		subs = append(subs, sub.MailboxInfo().Name)
	}

	// now gather their children
	// preallocate a safe bet to avoid slowdown issues
	children := make([]string, 0, 4 * len(subs))
	for _, sub := range subs {
		child, err := c.gatherFolders(sub)
		if err != nil {
			return nil, err
		}
		children = append(children, child...)
	}

	// and combine the two
	return append(subs, children...), nil
}

func (c *Conn) AllFolders() ([]string, error) {
	return c.gatherFolders("")
}

type MessageIter struct {
	c		*Conn
	cmd		*imap.Command
	started	bool
	cur		int
	folder	string
	validity	uint32
	err		error
}

func (c *Conn) ListMessages(folder string) (*MessageIter, error) {
	err := c.handle(c.c.Select(folder, true))
	if err != nil {
		return nil, err
	}

	// TODO why is this needed?
	c.c.Data = nil

	// If there are no messages, the fetch will fail
	if c.c.Mailbox.Messages == 0 {
		err = c.handle(c.c.Close(false))
		if err != nil {
			return nil, err
		}
		return nil, nil
	}

	// TODO see if we can make this without an error
	set, err := imap.NewSeqSet("1:*")
	if err != nil {
		c.handle(c.c.Close(false))
		return nil, err
	}
	cmd, err := c.c.Fetch(set, "UID RFC822.HEADER BODYSTRUCTURE")
	if err != nil {
		c.handle(c.c.Close(false))
		return nil, err
	}

	return &MessageIter{
		c:		c,
		cmd:		cmd,
		folder:	folder,
		validity:	c.c.Mailbox.UIDValidity,
		err:		nil,
	}, nil
}

func (m *MessageIter) Next() bool {
	if m.err != nil {
		return false
	}
	if m.started {
		// first exhaust the last receipt...
		m.cur++
		if m.cur < len(m.cmd.Data) {
			return true
		}
		// ...we're done; get ready for the next receipt
		m.cmd.Data = nil
		// TODO why do we need this?
		m.c.c.Data = nil
	}
	if !m.cmd.InProgress() {
		return false
	}
	err := m.c.c.Recv(-1)
	if err != nil {
		m.err = err
		return false
	}
	m.started = true
	m.cur = 0
	return true
}

func (m *MessageIter) Message() (*MsgTuple, *Message, error) {
	info := m.cmd.Data[m.cur].MessageInfo()
	tuple := &MsgTuple{
		Folder:		m.folder,
		UIDValidity:	m.validity,
		UID:			info.UID,
	}
	msg, err := ParseMessage(info)
	return tuple, msg, err
}

func (m *MessageIter) Err() error {
	return m.err
}

func (m *MessageIter) Close() error {
	_, err := m.cmd.Result(imap.OK)
	if err != nil {
		return err
	}
	return m.c.handle(m.c.c.Close(false))
}
