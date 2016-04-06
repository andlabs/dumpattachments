// 4 april 2016
package main

import (
	"bytes"
	"fmt"
	"mime"
	"net/mail"
	"strconv"
	"strings"

	"github.com/mxk/go-imap/imap"
)

var ErrNoHeader = fmt.Errorf("no header in message")

// TOOD get relevant stackoverflow links back

type MsgTuple struct {
	Folder      string
	UIDValidity uint32
	UID         uint32
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
	ContentType string
	From        string
	Subject     string
	Parts       []*MessagePart
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

	bodyStructure := info.Attrs["BODYSTRUCTURE"]
	b, err := ParseBodyStructure(bodyStructure)
	if err != nil {
		return nil, err
	}
	m.Parts = BodyStructureToPartList(b)

	return m, nil
}

func (m *Message) CanHaveAttachments() bool {
	// this is valid; thanks to kurushiyama in irc.freenode.net/#go-nuts
	// TODO find the stackoverflow that led to this confirmation
	return strings.HasPrefix(m.ContentType, "multipart/")
}

type MessagePart struct {
	ContentType string
	Filename    string
}

func BodyStructureToPartList(bs *BodyStructure) (pl []*MessagePart) {
	if !bs.Multipart {
		p := &MessagePart{
			ContentType: bs.ContentType,
			Filename:    bs.Name,
		}
		return []*MessagePart{p}
	}
	list := make([]*MessagePart, 0, len(bs.Parts))
	for _, part := range bs.Parts {
		list = append(list, BodyStructureToPartList(part)...)
	}
	return list
}
