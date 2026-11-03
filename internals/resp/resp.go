// Package resp implements RESP (Redis Serialization Protocol).
// It decodes commands sent by clients and encodes server replies.
package resp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

// Limits mirroring Redis defaults, so malformed or hostile input cannot
// make the server allocate arbitrary amounts of memory.
const (
	maxArrayLength = 1 << 20   // maximum number of elements in a command array
	maxBulkLength  = 512 << 20 // maximum declared bulk string size (512MB)
)

// ProtocolError reports malformed RESP input. When a reader returns it the
// connection is unusable and should be closed.
type ProtocolError struct {
	Reason string
}

func (e *ProtocolError) Error() string {
	return "protocol error: " + e.Reason
}

func protocolError(format string, args ...any) error {
	return &ProtocolError{Reason: fmt.Sprintf(format, args...)}
}

// ReadCommand reads one RESP array of bulk strings and returns its elements.
// For example "*3\r\n$3\r\nSET\r\n$4\r\nname\r\n$5\r\nSwarn\r\n" becomes
// []string{"SET", "name", "Swarn"}.
//
// A clean end of stream is reported as io.EOF. Anything else that is not a
// ProtocolError is a network error.
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

		// Read exactly the declared number of bytes plus the CRLF terminator,
		// so values containing spaces or embedded newlines stay intact.
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

// readLine reads one CRLF-terminated line, without the terminator.
func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		if err == io.EOF && len(line) > 0 {
			return "", protocolError("incomplete line %q", line)
		}
		return "", err // io.EOF on a clean disconnect
	}

	if len(line) < 2 || line[len(line)-2:] != "\r\n" {
		return "", protocolError("line missing CRLF terminator: %q", line)
	}

	return line[:len(line)-2], nil
}

// SimpleString encodes a simple string reply, e.g. "+OK\r\n".
func SimpleString(msg string) string {
	return "+" + msg + "\r\n"
}

// Error encodes an error reply, e.g. "-ERR message\r\n".
func Error(msg string) string {
	return "-" + msg + "\r\n"
}

// Integer encodes an integer reply, e.g. ":1\r\n".
func Integer(n int64) string {
	return ":" + strconv.FormatInt(n, 10) + "\r\n"
}

// BulkString encodes a bulk string reply using its byte length,
// e.g. "$5\r\nSwarn\r\n".
func BulkString(s string) string {
	return "$" + strconv.Itoa(len(s)) + "\r\n" + s + "\r\n"
}

// NullBulkString encodes a missing value reply, e.g. "$-1\r\n".
func NullBulkString() string {
	return "$-1\r\n"
}
