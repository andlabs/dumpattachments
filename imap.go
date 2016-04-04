// 4 april 2016
package main

import (
	"fmt"
	"strconv"

	"github.com/mxk/go-imap/imap"
)

// TOOD get relevant stackoverflow links back

type MsgTuple struct {
	Mailbox			string
	UIDValidity		uint32
	UID				uint32
}

func parseUint32(str string) (uint32, error) {
	n, err := strconv.ParseUint(str, 10, 32)
	return uint32(n), err
}

func MsgTupleFromLog(split []string) (m *MsgTuple, err error) {
	if len(split) < 3 {
		return nil, fmt.Errorf("MsgTupleFromLog(): invalid log line format")
	}
	m := new(MsgTuple)
	m.Mailbox, err = StringFromLog(split[0])
	if err != nil {
		return nil, err
	}
	m.UIDValidity, err = parseUint32(split[1])
	if err != nil {
		return nil, err
	}
	m.UID, err = parseUint32(split[2])
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (m *MsgTuple) ToLog() string {
	return fmt.Sprintf("%s %d %d", StringToLog(m.Mailbox),
		m.UIDValidity, m.UID)
}

func (m *MsgTuple) String() string {
	return fmt.Sprintf("%s %d %d", m.Mailbox,
		m.UIDValidity, m.UID)
}

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

	if c.Caps["STARTTLS"] {
		err = c.handle(c.c.StartTLS(nil))
		if err != nil {
			return nil, err
		}
	}
	if c.Caps["ID"] {
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
	err := c.handle(c.c.Select(folder, true)
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
		err:		error,
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
		m.c.Data = nil
	}
	if !m.cmd.InProgress() {
		return false
	}
	err := m.c.Recv(-1)
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
		Path:			m.path,
		UIDValidity:	m.validity,
		UID:			info.UID,
	}
	msg, err := ProcessMessage(info)
	return tuple, msg, err
}

func (m *MessageIter) Error() error {
	return m.err
}

func (m *MessageIter) Close() error {
	_, err := m.cmd.Result(imap.OK)
	if err != nil {
		return err
	}
	return handle(c.c.Close(false))
}
