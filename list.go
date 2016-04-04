// 4 april 2016
package main

import (
	"fmt"
	"os"
	"encoding/base64"
	"strings"
)

var ErrInvalidListLine = fmt.Errorf("invalid list line passed to list line parser")

func StringToList(what string) string {
	return base64.StdEncoding.EncodeToString([]byte(what))
}

func StringFromList(word string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(word)
	return string(b), err
}

func MessageToList(tuple *MsgTuple, m *Message, part int) string {
	ts := tuple.ToList()
	filename := m.Parts[part].Filename
	fs := StringToList(filename)
	return fmt.Sprintf("%s %s | folder:%q filename:%q from:%q subject:%q contentType:%q\n",
		ts, fs,
		tuple.Folder, filename,
		m.From, m.Subject,
		m.Parts[part].ContentType)
}

func TupleFilenameFromList(line string) (tuple *MsgTuple, filename string, err error) {
	split := strings.SplitN(line, " ", 5)
	if len(split) < 4 {
		return nil, "", ErrInvalidListLine
	}
	tuple, err = MsgTupleFromList(split)
	if err != nil {
		return nil, "", err
	}
	filename, err = StringFromList(split[3])
	if err != nil {
		return nil, "", err
	}
	return tuple, filename, nil
}

func ListLinesForMessage(tuple *MsgTuple, m *Message) string {
	if !m.CanHaveAttachments() {
		return ""
	}
	list := ""
	for i, part := range m.Parts {
		if part.Filename != "" {
			list += MessageToList(tuple, m, i)
		}
	}
	return list
}

func ListLinesForFolder(c *Conn, folder string) (string, error) {
	m, err := c.ListMessages(folder)
	if err != nil {
		return "", err
	}
	defer m.Close()

	list := ""
	for m.Next() {
		tuple, msg, err := m.Message()
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping invalid message %s: %v\n", tuple, err)
			continue
		}
		list += ListLinesForMessage(tuple, msg)
	}
	if m.Err() != nil {
		return "", m.Err()
	}

	return list, nil
}

func ListLines(c *Conn) (list string, err error) {
	folders, err := c.AllFolders()
	if err != nil {
		return "", err
	}

	list = ""
	for _, f := range folders {
		s, err := ListLinesForFolder(c, f)
		if err != nil {
			return "", err
		}
		list += s
	}

	return list, nil
}
