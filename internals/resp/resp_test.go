package resp

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestReadCommandValid(t *testing.T) {
	input := "*3\r\n$3\r\nSET\r\n$4\r\nname\r\n$5\r\nSwarn\r\n"

	args, err := ReadCommand(bufio.NewReader(strings.NewReader(input)))
	if err != nil {
		t.Fatalf("ReadCommand returned error: %v", err)
	}

	want := []string{"SET", "name", "Swarn"}
	if len(args) != len(want) {
		t.Fatalf("got %d args, want %d: %q", len(args), len(want), args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("arg %d = %q, want %q", i, args[i], want[i])
		}
	}
}

func TestReadCommandValueWithSpaces(t *testing.T) {
	// "my key" is 6 bytes, "hello world" is 11 bytes: lengths are respected
	// even though the values contain spaces.
	input := "*3\r\n$3\r\nSET\r\n$6\r\nmy key\r\n$11\r\nhello world\r\n"

	args, err := ReadCommand(bufio.NewReader(strings.NewReader(input)))
	if err != nil {
		t.Fatalf("ReadCommand returned error: %v", err)
	}

	want := []string{"SET", "my key", "hello world"}
	if len(args) != len(want) {
		t.Fatalf("got %d args, want %d: %q", len(args), len(want), args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("arg %d = %q, want %q", i, args[i], want[i])
		}
	}
}

func TestReadCommandMultipleFromOneReader(t *testing.T) {
	// Two commands on one stream: the first read must not consume bytes
	// belonging to the second command.
	input := "*1\r\n$4\r\nPING\r\n*2\r\n$3\r\nGET\r\n$4\r\nname\r\n"
	reader := bufio.NewReader(strings.NewReader(input))

	first, err := ReadCommand(reader)
	if err != nil {
		t.Fatalf("first ReadCommand error: %v", err)
	}
	if len(first) != 1 || first[0] != "PING" {
		t.Fatalf("first command = %q, want [PING]", first)
	}

	second, err := ReadCommand(reader)
	if err != nil {
		t.Fatalf("second ReadCommand error: %v", err)
	}
	want := []string{"GET", "name"}
	if len(second) != len(want) {
		t.Fatalf("second command = %q, want %q", second, want)
	}
	for i := range want {
		if second[i] != want[i] {
			t.Errorf("second command arg %d = %q, want %q", i, second[i], want[i])
		}
	}
}

func TestReadCommandEmptyArray(t *testing.T) {
	args, err := ReadCommand(bufio.NewReader(strings.NewReader("*0\r\n")))
	if err != nil {
		t.Fatalf("ReadCommand returned error: %v", err)
	}
	if len(args) != 0 {
		t.Fatalf("got %d args, want 0", len(args))
	}
}

func TestReadCommandCleanEOF(t *testing.T) {
	_, err := ReadCommand(bufio.NewReader(strings.NewReader("")))
	if !errors.Is(err, io.EOF) {
		t.Fatalf("error = %v, want io.EOF", err)
	}
}

func TestReadCommandMalformed(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"plain text instead of array", "PING\r\n"},
		{"missing array header", "+OK\r\n"},
		{"non numeric array length", "*x\r\n"},
		{"negative array length", "*-1\r\n"},
		{"argument not a bulk string", "*1\r\n+OK\r\n"},
		{"non numeric bulk length", "*1\r\n$abc\r\n"},
		{"negative bulk length", "*1\r\n$-2\r\n"},
		{"truncated bulk string", "*1\r\n$5\r\nabc"},
		{"bulk string missing CRLF", "*1\r\n$3\r\nabcXY\r\n"},
		{"header line missing CRLF", "*2"},
		{"incomplete line then EOF", "*3\r\n$3\r\nSE"},
		{"array longer than payload", "*3\r\n$3\r\nSET\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadCommand(bufio.NewReader(strings.NewReader(tt.input)))
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if errors.Is(err, io.EOF) {
				return // truncated stream is rejected too, just as io.EOF
			}
			var protoErr *ProtocolError
			if !errors.As(err, &protoErr) {
				t.Fatalf("error = %v (%T), want *ProtocolError", err, err)
			}
		})
	}
}

func TestEncoders(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"simple string", SimpleString("OK"), "+OK\r\n"},
		{"error", Error("ERR message"), "-ERR message\r\n"},
		{"integer one", Integer(1), ":1\r\n"},
		{"integer zero", Integer(0), ":0\r\n"},
		{"bulk string", BulkString("Swarn"), "$5\r\nSwarn\r\n"},
		{"bulk string with space", BulkString("hello world"), "$11\r\nhello world\r\n"},
		{"bulk string utf8 byte length", BulkString("日本語"), "$9\r\n日本語\r\n"},
		{"null bulk string", NullBulkString(), "$-1\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}
