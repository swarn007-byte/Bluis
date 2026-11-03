package main

import (
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"bluis/internals/storage"
)

func TestHandleCommandPing(t *testing.T) {
	store := storage.NewStorage()

	if got := handleCommand([]string{"PING"}, store); got != "+PONG\r\n" {
		t.Errorf("PING = %q, want %q", got, "+PONG\r\n")
	}
	if got := handleCommand([]string{"ping", "hi"}, store); got != "$2\r\nhi\r\n" {
		t.Errorf("PING hi = %q, want %q", got, "$2\r\nhi\r\n")
	}
	if got := handleCommand([]string{"PING", "a", "b"}, store); !strings.HasPrefix(got, "-ERR") {
		t.Errorf("PING a b = %q, want an error reply", got)
	}
}

func TestHandleCommandSetGet(t *testing.T) {
	store := storage.NewStorage()

	if got := handleCommand([]string{"SET", "name", "Swarn"}, store); got != "+OK\r\n" {
		t.Errorf("SET = %q, want %q", got, "+OK\r\n")
	}
	if got := handleCommand([]string{"GET", "name"}, store); got != "$5\r\nSwarn\r\n" {
		t.Errorf("GET name = %q, want %q", got, "$5\r\nSwarn\r\n")
	}
}

func TestHandleCommandValueWithSpaces(t *testing.T) {
	store := storage.NewStorage()

	value := "hello world from bluis"
	handleCommand([]string{"SET", "greeting", value}, store)

	want := fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)
	if got := handleCommand([]string{"GET", "greeting"}, store); got != want {
		t.Errorf("GET greeting = %q, want %q", got, want)
	}
}

func TestHandleCommandGetMissingKey(t *testing.T) {
	store := storage.NewStorage()

	if got := handleCommand([]string{"GET", "nope"}, store); got != "$-1\r\n" {
		t.Errorf("GET missing key = %q, want %q", got, "$-1\r\n")
	}
}

func TestHandleCommandDel(t *testing.T) {
	store := storage.NewStorage()

	handleCommand([]string{"SET", "name", "Swarn"}, store)

	if got := handleCommand([]string{"DEL", "name"}, store); got != ":1\r\n" {
		t.Errorf("first DEL = %q, want %q", got, ":1\r\n")
	}
	if got := handleCommand([]string{"DEL", "name"}, store); got != ":0\r\n" {
		t.Errorf("second DEL = %q, want %q", got, ":0\r\n")
	}
	if got := handleCommand([]string{"GET", "name"}, store); got != "$-1\r\n" {
		t.Errorf("GET after DEL = %q, want %q", got, "$-1\r\n")
	}
}

func TestHandleCommandErrors(t *testing.T) {
	store := storage.NewStorage()

	tests := []struct {
		name string
		args []string
	}{
		{"empty command", []string{}},
		{"set missing value", []string{"SET", "key"}},
		{"get extra argument", []string{"GET", "key", "extra"}},
		{"del without key", []string{"DEL"}},
		{"unknown command", []string{"FLUSHALL"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handleCommand(tt.args, store); !strings.HasPrefix(got, "-ERR") {
				t.Errorf("handleCommand(%q) = %q, want an error reply", tt.args, got)
			}
		})
	}
}

func startTestServer(t *testing.T) (string, *storage.Storage) {
	t.Helper()

	listen, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	store := storage.NewStorage()
	go serve(listen, store)

	t.Cleanup(func() { listen.Close() })

	return listen.Addr().String(), store
}

func dial(t *testing.T, addr string) net.Conn {
	t.Helper()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return conn
}

func send(t *testing.T, conn net.Conn, payload string) {
	t.Helper()

	if _, err := conn.Write([]byte(payload)); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func expect(t *testing.T, conn net.Conn, want string) {
	t.Helper()

	got := make([]byte, len(want))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("read reply: %v (want %q)", err, want)
	}
	if string(got) != want {
		t.Fatalf("reply = %q, want %q", got, want)
	}
}

func TestTCMultipleCommandsOneConnection(t *testing.T) {
	addr, _ := startTestServer(t)
	conn := dial(t, addr)

	send(t, conn, "*1\r\n$4\r\nPING\r\n")
	expect(t, conn, "+PONG\r\n")

	send(t, conn, "*3\r\n$3\r\nSET\r\n$4\r\nname\r\n$5\r\nSwarn\r\n")
	expect(t, conn, "+OK\r\n")

	send(t, conn, "*2\r\n$3\r\nGET\r\n$4\r\nname\r\n")
	expect(t, conn, "$5\r\nSwarn\r\n")

	send(t, conn, "*3\r\n$3\r\nSET\r\n$5\r\nabout\r\n$11\r\nhello world\r\n")
	expect(t, conn, "+OK\r\n")

	send(t, conn, "*2\r\n$3\r\nGET\r\n$5\r\nabout\r\n")
	expect(t, conn, "$11\r\nhello world\r\n")

	send(t, conn, "*2\r\n$3\r\nGET\r\n$4\r\nnone\r\n")
	expect(t, conn, "$-1\r\n")

	send(t, conn, "*2\r\n$3\r\nDEL\r\n$4\r\nname\r\n")
	expect(t, conn, ":1\r\n")

	send(t, conn, "*2\r\n$3\r\nDEL\r\n$4\r\nname\r\n")
	expect(t, conn, ":0\r\n")
}

func TestTCPMalformedRESP(t *testing.T) {
	addr, _ := startTestServer(t)
	conn := dial(t, addr)

	send(t, conn, "hello\r\n")

	reply, err := io.ReadAll(conn)
	if err != nil {
		t.Fatalf("read error reply: %v", err)
	}
	if !strings.HasPrefix(string(reply), "-ERR protocol error:") {
		t.Fatalf("reply = %q, want a protocol error reply", reply)
	}
}

func TestTCPConcurrentClients(t *testing.T) {
	addr, _ := startTestServer(t)

	const clients = 8
	const iterations = 25

	errs := make(chan error, clients)
	for i := 0; i < clients; i++ {
		go func(id int) {
			errs <- concurrentClient(addr, id, iterations)
		}(i)
	}

	for i := 0; i < clients; i++ {
		if err := <-errs; err != nil {
			t.Error(err)
		}
	}
}

func concurrentClient(addr string, id, iterations int) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("client %d dial: %w", id, err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return fmt.Errorf("client %d deadline: %w", id, err)
	}

	roundTrip := func(payload, want string) error {
		if _, err := conn.Write([]byte(payload)); err != nil {
			return fmt.Errorf("client %d write: %w", id, err)
		}
		got := make([]byte, len(want))
		if _, err := io.ReadFull(conn, got); err != nil {
			return fmt.Errorf("client %d read: %w", id, err)
		}
		if string(got) != want {
			return fmt.Errorf("client %d reply = %q, want %q", id, got, want)
		}
		return nil
	}

	for i := 0; i < iterations; i++ {
		key := fmt.Sprintf("client-%d-key-%d", id, i)
		value := fmt.Sprintf("value %d with spaces", i)

		steps := []struct {
			payload string
			want    string
		}{
			{fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(key), key, len(value), value), "+OK\r\n"},
			{fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(key), key), fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)},
			{fmt.Sprintf("*2\r\n$3\r\nDEL\r\n$%d\r\n%s\r\n", len(key), key), ":1\r\n"},
			{fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(key), key), "$-1\r\n"},
			{"*3\r\n$3\r\nSET\r\n$6\r\nshared\r\n$7\r\ncontend\r\n", "+OK\r\n"},
		}

		for _, step := range steps {
			if err := roundTrip(step.payload, step.want); err != nil {
				return err
			}
		}
	}

	return nil
}
