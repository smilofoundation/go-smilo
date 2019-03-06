package core

import (
	"errors"
	"io/ioutil"
	"net"
)

var (
	waitingErr = errors.New("unix socket dial failed")
	upcheckErr = errors.New("http upcheck failed")
	doneErr    = errors.New("done")
)

func checkFunc(tmIPCFile string) error {
	conn, err := net.Dial("unix", tmIPCFile)
	if err != nil {
		return waitingErr
	}
	if _, err := conn.Write([]byte("GET /upcheck HTTP/1.0\r\n\r\n")); err != nil {
		return upcheckErr
	}
	result, err := ioutil.ReadAll(conn)
	if err != nil || string(result) == "I'm up!" {
		return doneErr
	}
	return upcheckErr
}
