// 6 april 2016
package main

import (
	"fmt"
	"strings"

	"github.com/mxk/go-imap/imap"
)

var ErrIncompleteBodyStructure = fmt.Errorf("incomplete body structure")
var ErrOddBodyParameters = fmt.Errorf("odd number of body parameters in non-multipart body structure entry")

type BodyStructure struct {
	Multipart   bool
	ContentType string
	Name        string
	Parts       []*BodyStructure
}

// TODO stricter type checking?
func ParseBodyStructure(bodyStructure imap.Field) (b *BodyStructure, err error) {
	b = new(BodyStructure)

	elem := imap.AsList(bodyStructure)
	if len(elem) < 2 {
		return nil, ErrIncompleteBodyStructure
	}
	firstType := imap.TypeOf(elem[0])
	b.Multipart = firstType&imap.List != 0

	i := 1 // determines placement of subtype; set to non-multipart first
	if b.Multipart {
		var e imap.Field

		b.ContentType = "multipart"
		// huh: the spec says the first item should be a list of lists,
		// but in practice this is just items until the first string?
		// TODO
		for i, e = range elem {
			// first non-list (string) item ends it
			if imap.TypeOf(e)&imap.List == 0 {
				break
			}
			part, err := ParseBodyStructure(e)
			if err != nil {
				return nil, err
			}
			b.Parts = append(b.Parts, part)
		}
	} else if len(elem) > 2 { // TODO is omitting this valid?
		b.ContentType = imap.AsString(elem[0])
		kv := imap.AsList(elem[2])
		if len(kv) != 0 { // only if there are parameters
			if len(kv)%2 != 0 {
				return nil, ErrOddBodyParameters
			}
			for i := 0; i < len(kv); i += 2 {
				// TODO will it always be capitalized?
				if imap.AsString(kv[i]) == "NAME" {
					b.Name = imap.AsString(kv[i+1])
					break
				}
			}
		}
	}

	b.ContentType += "/" + imap.AsString(elem[i])
	// these fields seem to be all uppercase; make them lowercase to make things easier
	b.ContentType = strings.ToLower(b.ContentType)

	return b, nil
}
