package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/nikandfor/errors"
)

var (
	listen = flag.String("listen", ":6379", "address to listen to")
)

func main() {
	flag.Parse()

	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() (err error) {
	l, err := net.Listen("tcp", *listen)
	if err != nil {
		return errors.Wrap(err, "listen")
	}

	defer closeIt(l, &err, "close listener")

	log.Printf("listening %v", l.Addr())

	for {
		c, err := l.Accept()
		if err != nil {
			return errors.Wrap(err, "accept")
		}

		go handleConn(c)
	}

	return nil
}

func handleConn(c net.Conn) (err error) {
	defer closeIt(c, &err, "close connection")

	buf := make([]byte, 128)

	for {
		n, err := c.Read(buf)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return errors.Wrap(err, "read command")
		}

		log.Printf("command:\n%s", buf[:n])

		_, err = c.Write([]byte("+PONG\r\n"))
		if err != nil {
			return errors.Wrap(err, "write response")
		}
	}

	return nil
}

func closeIt(c io.Closer, errp *error, msg string) {
	err := c.Close()
	if *errp == nil {
		*errp = errors.Wrap(err, "%v", msg)
	}
}
