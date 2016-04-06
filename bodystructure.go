// 6 april 2016
package main

import (
	"fmt"
	"strings"

	"github.com/mxk/go-imap/imap"
)

var ErrInvalidBodyStructure = fmt.Errorf("invalid body structure")

type BodyStructure struct {
	Multipart		bool
	ContentType	string
	Name		string
	Parts			[]*BodyStructure
}

// TODO differentiate errors?
// TODO stricter type checking?
func ParseBodyStructure(bodyStructure imap.Field) (b *BodyStructure, err error) {
	b = new(BodyStructure)

	elem := imap.AsList(bodyStructure)
	if len(elem) < 2 {
		return nil, ErrInvalidBodyStructure
	}
	firstType := imap.TypeOf(elem[0])
	b.Multipart = firstType & imap.List != 0
	if !b.Multipart {
		b.ContentType = imap.AsString(elem[0])
	} else {
		b.ContentType = "multipart"
	}
	b.ContentType += "/" + imap.AsString(elem[1])
	// these fields seem to be all uppercase; make them lowercase to make things easier
	b.ContentType = strings.ToLower(b.ContentType)

	if b.Multipart {
		parts := imap.AsList(elem[0])
		// TODO is zero parts invalid?
		b.Parts = make([]*BodyStructure, 0, len(parts))
		for _, part := range parts {
			pb, err := ParseBodyStructure(part)
			if err != nil {
				return nil, err
			}
			b.Parts = append(b.Parts, pb)
		}
	} else if len(elem) > 2 {		// TODO is omitting this valid?
		if imap.TypeOf(elem[2]) & imap.List != 0 {
			return nil, ErrInvalidBodyStructure
		}
		kv := imap.AsList(elem[2])
		if len(kv) % 2 != 0 {
			return nil, ErrInvalidBodyStructure
		}
		for i := 0; i < len(kv); i += 2 {
			// TODO will it also be capitalized?
			if imap.AsString(kv[i]) == "NAME" {
				b.Name = imap.AsString(kv[i + 1])
				break
			}
		}
	}

	return b, nil
}
