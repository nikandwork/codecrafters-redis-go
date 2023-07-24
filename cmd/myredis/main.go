package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/nikandfor/errors"
)

var (
	listen = flag.String("listen", ":6379", "address to listen to")
)

const (
	Array      = '*'
	BulkString = '$'
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

		go func() {
			err := handleConn(c)
			if err != nil {
				log.Printf("handle conn: %v", err)
			}
		}()
	}

	return nil
}

func handleConn(c net.Conn) (err error) {
	defer closeIt(c, &err, "close connection")

	buf := make([]byte, 128)

	for {
		buf = buf[:cap(buf)]

		n, err := c.Read(buf)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return errors.Wrap(err, "read command")
		}

		buf = buf[:n]

		log.Printf("command:\n%s", buf)

		cmd, i, err := parseCommand(buf)
		if err != nil {
			return errors.Wrap(err, "parse command (pos %d)", i)
		}

		buf = buf[:0]

		switch strings.ToLower(cmd[0]) {
		case "command":
			buf = appendSimpleString(buf, "")
		case "ping":
			buf = appendSimpleString(buf, "PONG")
		case "echo":
			buf = appendSimpleString(buf, strings.Join(cmd[1:], " "))
		default:
			panic(cmd[0])
		}

		_, err = c.Write(buf)
		if err != nil {
			return errors.Wrap(err, "write response")
		}
	}

	return nil
}

func appendSimpleString(b []byte, s string) []byte {
	return fmt.Appendf(b, "+%s\r\n", s)
}

func parseCommand(b []byte) ([]string, int, error) {
	i := 0

	if i == len(b) {
		return nil, i, io.ErrUnexpectedEOF
	}

	if b[i] != Array {
		return nil, i, errors.New("array expected")
	}

	i++

	n, i, err := parseNumber(b, i)
	if err != nil {
		return nil, i, errors.Wrap(err, "array length")
	}

	var args []string
	var arg string

	for a := 0; a < n; a++ {
		arg, i, err = parseString(b, i)
		if err != nil {
			return nil, i, err
		}

		args = append(args, arg)
	}

	if i != len(b) {
		return nil, i, errors.New("partial read")
	}

	return args, i, nil
}

func parseString(b []byte, st int) (string, int, error) {
	i := st

	if i == len(b) {
		return "", i, io.ErrUnexpectedEOF
	}

	if b[i] != BulkString {
		return "", i, errors.New("string expected")
	}

	i++

	n, i, err := parseNumber(b, i)
	if err != nil {
		return "", i, errors.Wrap(err, "string length")
	}

	if i+n > len(b) {
		return "", st, io.ErrUnexpectedEOF
	}

	s := string(b[i : i+n])
	i += n

	i, err = expect(b, i, "\r\n")
	if err != nil {
		return "", i, err
	}

	return s, i, nil
}

func parseNumber(b []byte, st int) (n, i int, err error) {
	i = st

	for i < len(b) && b[i] >= '0' && b[i] <= '9' {
		n = n*10 + int(b[i]-'0')
		i++
	}

	i, err = expect(b, i, "\r\n")
	if err != nil {
		return 0, i, err
	}

	return n, i, nil
}

func expect(b []byte, i int, exp string) (int, error) {
	if i+len(exp) <= len(b) && string(b[i:i+len(exp)]) == exp {
		return i + len(exp), nil
	}

	return i, errors.New("expected: %q", exp)
}

func closeIt(c io.Closer, errp *error, msg string) {
	err := c.Close()
	if *errp == nil {
		*errp = errors.Wrap(err, "%v", msg)
	}
}
