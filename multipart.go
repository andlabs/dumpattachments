// 5 april 2016
package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
)

type Multipart struct {
	// This is a stack of readers, to handle recursive multipart reads.
	m []*multipart.Reader
	// This, on the other hand, only stores the innermost.
	p   *Part
	err error
}

func MultipartFromRaw(header []byte, body []byte) (m *Multipart, err error) {
	mailmsg, err := mail.ReadMessage(bytes.NewReader(header))
	if err != nil {
		return nil, err
	}
	contentType := mailmsg.Header.Get("Content-Type")
	_, extra, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, err
	}
	boundary := extra["boundary"]

	m = new(Multipart)
	m.recurse(body, boundary)
	return m, nil
}

func (m *Multipart) nextPart() (*multipart.Part, error) {
	return m.m[len(m.m)-1].NextPart()
}

func (m *Multipart) recurse(body []byte, boundary string) {
	m2 := multipart.NewReader(bytes.NewReader(body), boundary)
	m.m = append(m.m, m2)
}

func (m *Multipart) unrecurse() {
	m.m = m.m[:len(m.m)-1]
}

func (m *Multipart) Next() bool {
	if m.err != nil { // error
		return false
	}
	if len(m.m) == 0 { // no more at all
		return false
	}
	mp, err := m.nextPart()
	m.err = err
	if m.err == io.EOF { // no more at this level
		m.err = nil
		m.unrecurse()
		return m.Next()
	} else if m.err != nil { // error at this level
		return false
	}
	m.p, m.err = extractPart(mp)
	if m.err != nil {
		return false
	}
	if strings.HasPrefix(m.p.ContentType, "multipart/") {
		m.recurse(m.p.Contents, m.p.Boundary)
		return m.Next()
	}
	return true
}

type Part struct {
	Filename    string
	ContentType string
	Boundary    string
	Contents    []byte
	Encoding    string
}

func extractPart(mp *multipart.Part) (p *Part, err error) {
	p = new(Part)
	defer mp.Close()
	p.Filename = mp.FileName()
	contentType := mp.Header.Get("Content-Type")
	contentType, extra, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, err
	}
	p.ContentType = contentType
	p.Boundary = extra["boundary"]
	p.Contents, err = ioutil.ReadAll(mp)
	if err != nil {
		return nil, err
	}
	p.Encoding = mp.Header.Get("Content-Transfer-Encoding")
	// I've seen this in all caps, so...
	p.Encoding = strings.ToLower(p.Encoding)
	return p, nil
}

func (m *Multipart) Part() *Part {
	return m.p
}

func (m *Multipart) Err() error {
	return m.err
}
