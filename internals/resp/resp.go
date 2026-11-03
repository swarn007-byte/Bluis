package resp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

const (
	maxArrayLength = 1 << 20
	maxBulkLength  = 512 << 20
)

type ProtocolError struct {
	Reason string
}

func (e *ProtocolError) Error() string {
	return "protocol error: " + e.Reason
}

func protocolError(format string, args ...any) error {
	return &ProtocolError{Reason: fmt.Sprintf(format, args...)}
}

func ReadCommand(r *bufio.Reader) ([]string, error) {
	line, err := readLine(r)
	if err != nil {
		return nil, err
	}

	if len(line) == 0 || line[0] != '*' {
		return nil, protocolError("expected array header, got %q", line)
	}

	count, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, protocolError("invalid array length %q", line[1:])
	}
	if count < 0 || count > maxArrayLength {
		return nil, protocolError("invalid array length %d", count)
	}

	var args []string
	for i := 0; i < count; i++ {
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}

		if len(line) == 0 || line[0] != '$' {
			return nil, protocolError("expected bulk string header for argument %d, got %q", i, line)
		}

		length, err := strconv.Atoi(line[1:])
		if err != nil {
			return nil, protocolError("invalid bulk length %q", line[1:])
		}
		if length < 0 || length > maxBulkLength {
			return nil, protocolError("invalid bulk length %d", length)
		}

		buf := make([]byte, length+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil, protocolError("incomplete bulk string for argument %d", i)
			}
			return nil, err
		}
		if buf[length] != '\r' || buf[length+1] != '\n' {
			return nil, protocolError("bulk string for argument %d missing CRLF terminator", i)
		}

		args = append(args, string(buf[:length]))
	}

	return args, nil
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		if err == io.EOF && len(line) > 0 {
			return "", protocolError("incomplete line %q", line)
		}
		return "", err
	}

	if len(line) < 2 || line[len(line)-2:] != "\r\n" {
		return "", protocolError("line missing CRLF terminator: %q", line)
	}

	return line[:len(line)-2], nil
}

func SimpleString(msg string) string {
	return "+" + msg + "\r\n"
}

func Error(msg string) string {
	return "-" + msg + "\r\n"
}

func Integer(n int64) string {
	return ":" + strconv.FormatInt(n, 10) + "\r\n"
}

func BulkString(s string) string {
	return "$" + strconv.Itoa(len(s)) + "\r\n" + s + "\r\n"
}

func NullBulkString() string {
	return "$-1\r\n"
}
