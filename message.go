// 4 april 2016
package main

import (
	"fmt"
	"bytes"
	"net/mail"
	"mime"
	"strings"

	"github.com/mxk/go-imap/imap"
)

var ErrNoHeader = fmt.Errorf("no header in message")
var ErrInvalidBodyStructure = fmt.Errorf("invalid body structure in message")
var ErrInvalidMessagePart = fmt.Errorf("invalid message part in message")

type Message struct {
	ContentType	string
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

	contentType = header.Get("Content-Type")
	if contentType == "" {
		// assume emails without a Content-Type are text/plain
		// yeah, I have one from 2009 and I *think* it's text/plain...
		contentType = "text/plain"
	}
	m.ContentType, err = mime.ParseMediaType(contentType)
	if err != nil {
		return nil, err
	}

	parts := imap.AsList(info.Attrs["BODYSTRUCTURE"])
	if len(bodyStructure) == 0 {
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
		p, err := ParsePart(bodyStructure)
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
	p.ContentType := imap.AsString(part[0]) + "/" +
		imap.AsString(part[1])
	// and just to look good
	p.ContentType = strings.ToLower(p.ContentType)

	// we can't use imap.AsFieldMap() here because the key type here is not Atom
	// TODO will it always be at position 2?
	ext := imap.AsList(mime[2])
	if len(ext) == 0 || len(ext) % 2 == 1 {
		return nil, ErrInvalidMessagePart
	}
	p.Filename = ""
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