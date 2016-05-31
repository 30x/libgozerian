package main

import (
	"io"
)

/*
#include <stdlib.h>
*/
import "C"

type requestBody struct {
	handler commandHandler
	started bool
	curBuf  []byte
}

func (b *requestBody) Read(buf []byte) (int, error) {
	if !b.started {
		b.handler.StartRead()
		// First tell the caller that we need some data.
		b.handler.Commands() <- command{id: RBOD}
		b.started = true
	}

	cb := b.curBuf
	if cb == nil {
		// Will return nil at end of channel.
		cb = <-b.handler.Bodies()
	}

	if cb == nil {
		return 0, io.EOF
	}

	if len(cb) <= len(buf) {
		copy(buf, cb)
		//copy((*[1<<30]byte)(buf)[:], cb)
		b.curBuf = nil
		return len(cb), nil
	}

	copy(buf, cb[:len(buf)])
	b.curBuf = cb[len(buf):]
	return len(buf), nil
}

func (b *requestBody) Close() error {
	if b.started {
		// Need to clear the channel.
		b.curBuf = nil
		drained := <-b.handler.Bodies()
		for drained != nil {
			drained = <-b.handler.Bodies()
		}
		b.started = false
	}
	return nil
}

// Utility used in testing

func readChunk(id int32, release bool) []byte {
	c := getChunk(id)
	ret := make([]byte, c.len)
	copy(ret[:], (*[1 << 30]byte)(c.data)[:])
	if release {
		C.free(c.data)
		releaseChunk(id)
	}
	return ret
}
