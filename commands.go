package main

const (
  CmdError = iota
  CmdGetBody
  CmdWriteBody
  CmdDone
)

const (
  ErrorVal = "ERRR"
  GetBodyVal = "RBOD"
  WriteBodyVal = "WBOD"
  DoneVal = "DONE"
)

type command struct {
  id int
  msg string
}

func createErrorCommand(err error) command {
  return command{
    id: CmdError,
    msg: err.Error(),
  }
}

func (c command) String() string {
  var pfx string
  switch c.id {
  case CmdError:
    pfx = ErrorVal
  case CmdGetBody:
    pfx = GetBodyVal
  case CmdWriteBody:
    pfx = WriteBodyVal
  case CmdDone:
    pfx = DoneVal
  default:
    panic("Internal command mismatch")
  }

  return pfx + c.msg
}
