// 4 april 2016
package main

import (
	"fmt"
	"bytes"
	"net/mail"
	"mime"
	"strings"
	"strconv"

	"github.com/mxk/go-imap/imap"
)

var ErrNoHeader = fmt.Errorf("no header in message")
var ErrInvalidBodyStructure = fmt.Errorf("invalid body structure in message")
var ErrInvalidMessagePart = fmt.Errorf("invalid message part in message")

// TOOD get relevant stackoverflow links back

type MsgTuple struct {
	Folder			string
	UIDValidity		uint32
	UID				uint32
}

func parseUint32(str string) (uint32, error) {
	n, err := strconv.ParseUint(str, 10, 32)
	return uint32(n), err
}

func mktuple(split []string, list bool) (m *MsgTuple, err error) {
	m = new(MsgTuple)
	if len(split) < 3 {
		// TODO
	}
	m.Folder = split[0]
	if list {
		m.Folder, err = StringFromList(m.Folder)
		if err != nil {
			return nil, err
		}
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

func MsgTupleFromArgs(split []string) (m *MsgTuple, err error) {
	return mktuple(split, false)
}

func MsgTupleFromList(split []string) (m *MsgTuple, err error) {
	return mktuple(split, true)
}

func (m *MsgTuple) ToList() string {
	return fmt.Sprintf("%s %d %d", StringToList(m.Folder),
		m.UIDValidity, m.UID)
}

func (m *MsgTuple) String() string {
	return fmt.Sprintf("%q %d %d", m.Folder,
		m.UIDValidity, m.UID)
}

type Message struct {
	ContentType	string
	From		string
	Subject		string
	Parts			[]*MessagePart
}

func ParseMessage(info *imap.MessageInfo) (m *Message, err error) {
	m = new(Message)

	headerBytes := imap.AsBytes(info.Attrs["RFC822.HEADER"])
	if len(headerBytes) == 0 {
		return nil, ErrNoHeader
	}
	mailmsg, err := mail.ReadMessage(bytes.NewReader(headerBytes))
	if err != nil {
		return nil, err
	}
	header := mailmsg.Header

	contentType := header.Get("Content-Type")
	if contentType == "" {
		// assume emails without a Content-Type are text/plain
		// yeah, I have one from 2009 and I *think* it's text/plain...
		contentType = "text/plain"
	}
	m.ContentType, _, err = mime.ParseMediaType(contentType)
	if err != nil {
		return nil, err
	}

	m.From = header.Get("From")
	m.Subject = header.Get("Subject")

	parts := imap.AsList(info.Attrs["BODYSTRUCTURE"])
	if len(parts) == 0 {
		return nil, ErrInvalidBodyStructure
	}
	for _, part := range parts {
		// a string will always be after the last multipart file
		if imap.TypeOf(part) & imap.List == 0 {
			break
		}
		p, err := ParsePart(imap.AsList(part))
		if err != nil {
			return nil, err
		}
		m.Parts = append(m.Parts, p)
	}
	if len(m.Parts) == 0 {			// only one part
		p, err := ParsePart(parts)
		if err != nil {
			return nil, err
		}
		m.Parts = append(m.Parts, p)
	}

	return m, nil
}

func (m *Message) CanHaveAttachments() bool {
	// this is valid; thanks to kurushiyama in irc.freenode.net/#go-nuts
	// TODO find the stackoverflow that led to this confirmation
	return strings.HasPrefix(m.ContentType, "multipart/")
}

type MessagePart struct {
	ContentType	string
	Filename		string
}

func ParsePart(part []imap.Field) (p *MessagePart, err error) {
	p = new(MessagePart)
	if len(part) < 3 {
		return nil, ErrInvalidMessagePart
	}

	// these will always be at positions 0 and 1
	// TODO error check
	p.ContentType = imap.AsString(part[0]) + "/" +
		imap.AsString(part[1])
	// and just to look good
	p.ContentType = strings.ToLower(p.ContentType)

	// we can't use imap.AsFieldMap() here because the key type here is not Atom
	// TODO will it always be at position 2?
	ext := imap.AsList(part[2])
// TODO there seem to be even deeper trees of multiparts
/*	if len(ext) % 2 == 1 {
		return nil, ErrInvalidMessagePart
	}
*/	p.Filename = ""
	for i := 0; i < len(ext); i += 2 {
		// TODO wil it always be capitalized?
		// TODO error check
		if imap.AsString(ext[i]) == "NAME" {
			p.Filename = imap.AsString(ext[i + 1])
			break
		}
	}

	return p, nil
}
