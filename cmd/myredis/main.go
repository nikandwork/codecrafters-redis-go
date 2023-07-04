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

	_, err = l.Accept()
	if err != nil {
		return errors.Wrap(err, "accept")
	}

	return nil
}

func closeIt(c io.Closer, errp *error, msg string) {
	err := c.Close()
	if *errp == nil {
		*errp = errors.Wrap(err, "%v", msg)
	}
}
