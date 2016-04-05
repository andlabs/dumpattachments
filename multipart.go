// 5 april 2016
package main

import (
	"io"
	"io/ioutil"
	"bytes"
	"net/mail"
	"mime"
	"mime/multipart"
)

// TODO recurse multiparts
type Multipart struct {
	m	*multipart.Reader
	p	*multipart.Part
	err	error
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
	m.m = multipart.NewReader(bytes.NewReader(body), boundary)
	return m, nil
}

func (m *Multipart) Next() bool {
	m.p, m.err = m.m.NextPart()
	if m.err == io.EOF {		// no more
		m.err = nil
		return false
	} else if m.err != nil {		// error
		return false
	}
	return true
}

type Part struct {
	Filename		string
	ContentType	string
	Contents		[]byte
}

func (m *Multipart) Part() (p *Part, err error) {
	p = new(Part)
	p.Filename = m.p.FileName()
	p.ContentType = m.p.Header.Get("Content-Type")
	p.ContentType, _, err = mime.ParseMediaType(p.ContentType)
	if err != nil {
		return nil, err
	}
	p.Contents, err = ioutil.ReadAll(m.p)
	if err != nil {
		return nil, err
	}
	m.p.Close()
	return p, nil
}

func (m *Multipart) Err() error {
	return m.err
}
