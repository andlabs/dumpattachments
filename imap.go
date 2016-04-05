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

func (c *Conn) gatherFolders(root string) ([]string, error) {
	cmd, err := imap.Wait(c.c.List(root, "%"))
	if err != nil {
		return nil, err
	}
	// scale it up just to be safe for complex trees
	folders := make([]string, 0, len(cmd.Data) * 4)
	for _, folder := range cmd.Data {
		name := folder.MailboxInfo().Name
		folders = append(folders, name)
		subs, err := c.gatherFolders(name)
		if err != nil {
			return nil, err
		}
		folders = append(folders, subs...)
	}
	return folders, nil
}

func (c *Conn) AllFolders() ([]string, error) {
	return c.gatherFolders("")
}

type MessageIter struct {
	c		*Conn
	first		uint64		// uint64 to prevent overflow
	n		uint64
	folder	string
	validity	uint32
	err		error

	// When cmd is nil, first:first+messagesPerCmd messages are extracted into cmd and first is advanced.
	cmd		*imap.Command
	cur		int
}

const messagesPerCmd = 100

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

	return &MessageIter{
		c:		c,
		first:		1,
		n:		uint64(c.c.Mailbox.Messages),
		folder:	folder,
		validity:	c.c.Mailbox.UIDValidity,
	}, nil
}

func (m *MessageIter) nextNextCmd() bool {
	if m.first > m.n {		// finished?
		return false
	}

	set := &imap.SeqSet{}
	// TODO comment these calculations
	last := m.first + messagesPerCmd - 1
	if last > m.n {
		last = m.n
	}
	set.AddRange(uint32(m.first), uint32(last))
	m.first = last + 1
	m.cmd, m.err = m.c.c.Fetch(set, "UID RFC822.HEADER BODYSTRUCTURE")
	// just to be safe
	if m.err != nil && m.cmd != nil {
		m.cmd.Result(imap.OK)
		m.cmd = nil
	}
	return m.cmd != nil
}

// TODO comment this
func (m *MessageIter) Next() bool {
	// treat a nil MessageIter as meaning no messages, no error
	if m == nil {
		return false
	}
	if m.err != nil {
		return false
	}
	if m.cmd == nil {
		if !m.nextNextCmd() {
			// either we're finished or there's an error
			return false
		}
	}
	if len(m.cmd.Data) != 0 {
		// try the next data element
		m.cur++
		if m.cur < len(m.cmd.Data) {
			return true
		}
		// otherwise we're done; get ready for the next receipt
		m.cmd.Data = nil
		// TODO why do we need this?
		m.c.c.Data = nil
	}
	// Sometimes we can get an empty m.cmd.Data; in that case, we need to run these steps again.
	for {
		if !m.cmd.InProgress() {
			_, m.err = m.cmd.Result(imap.OK)
			m.cmd = nil
			return m.Next()
		}
		m.err = m.c.c.Recv(-1)
		if m.err != nil {
			return false
		}
		if len(m.cmd.Data) == 0 {
			// just to be safe
			m.cmd.Data = nil
			// TODO why do we need this?
			m.c.c.Data = nil
			continue
		}
		// otherwise we made it
		break
	}
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
	if m == nil {
		return nil
	}
	return m.err
}

func (m *MessageIter) Close() error {
	if m == nil {
		return nil
	}
	if m.cmd != nil {
		_, err := m.cmd.Result(imap.OK)
		if err != nil {
			return err
		}
	}
	return m.c.handle(m.c.c.Close(false))
}
