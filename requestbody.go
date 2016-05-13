package main

import (
  "io"
)

type requestBody struct {
  req *Request
  started bool
  curBuf []byte
}

func (b *requestBody) Read(buf []byte) (int, error) {
  if !b.started {
    // First tell the caller that we need some data.
    b.req.cmds <- command{id: CmdGetBody}
    b.started = true
  }

  cb := b.curBuf
  if cb == nil {
    // Will return nil at end of channel.
    cb = <- b.req.bodies
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
    drained := <- b.req.bodies
    for drained != nil {
      drained = <- b.req.bodies
    }
    b.started = false
  }
  return nil
}
