// 5 april 2016
package main

import (
	"fmt"
	"os"
	"bufio"
	"strings"
)

func tryOpen(filename string) (f *os.File, realFilename string, err error) {
	realFilename = filename
	suffix := 0
	for {
		f, err = os.OpenFile(realFilename, os.O_WRONLY | os.O_CREATE | os.O_EXCL, 0644)
		if err == nil {
			break
		}
		if !os.IsExist(err) {
			return nil, "", err
		}
		realFilename = fmt.Sprintf("%s%d", filename, suffix)
		suffix++
	}
	return f, realFilename, nil
}

func writeOut(filename string, body []byte) (realFilename string, err error) {
	f, realFilename, err := tryOpen(filename)
	if err != nil {
		return "", err
	}
errf("%s\n", realFilename)
f.Close()
	return realFilename, nil
}

func doDump(c *Conn) {
	someError := false
	stdin := bufio.NewScanner(os.Stdin)
	for lineno := 1; stdin.Scan(); lineno++ {
		line := stdin.Text()
		split := strings.SplitN(line, " ", 5)
		if len(split) < 4 {
			// TODO elaborate this
			errf("line %d invalid (not enough fields); skipping", lineno)
			someError = true
			continue
		}
		tuple, err := MsgTupleFromList(split)
		if err != nil {
			errf("line %d has invalid message tuple (%v); skipping", lineno, err)
			someError = true
			continue
		}
		filename, err := StringFromList(split[3])
		if err != nil {
			errf("line %d has invalid attachment filename (%v); skipping", lineno, err)
			someError = true
			continue
		}
		_, body, err := c.RawMessage(tuple)
		if err != nil {
			errf("failed to retrieve message for line %d (%v); skipping", lineno, err)
			someError = true
			continue
		}
errf("%s",filename)
		filename, err = writeOut(filename, body)
		if err != nil {
			errf("failed to write %s (%v) for line %d; skipping", filename, err, lineno)
			someError = true
			continue
		}
	}
	if err := stdin.Err(); err != nil {
		panic(err)
	}
	if someError {
		os.Exit(4)
	}
}
