package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"

	"bluis/internals/resp"
	"bluis/internals/storage"
)

func main() {
	listen, err := net.Listen("tcp", "localhost:6380")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer listen.Close()

	store := storage.NewStorage()

	fmt.Println("BLUIS server is running...")

	serve(listen, store)
}

// serve accepts connections and handles each one in its own goroutine.
func serve(listen net.Listener, store *storage.Storage) {
	for {
		conn, err := listen.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			fmt.Println(err)
			continue
		}

		fmt.Println("New client connected!")

		go handleClient(conn, store)
	}
}

func handleClient(conn net.Conn, store *storage.Storage) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		args, err := resp.ReadCommand(reader)
		if err != nil {
			var protoErr *resp.ProtocolError
			if errors.As(err, &protoErr) {
				// Malformed RESP: tell the client, then drop the connection.
				fmt.Println("Protocol error:", protoErr)
				conn.Write([]byte(resp.Error("ERR " + protoErr.Error())))
			}
			// Clean disconnect (io.EOF) or network error: close either way.
			return
		}

		fmt.Println("Received:", args)

		response := handleCommand(args, store)

		if _, err := conn.Write([]byte(response)); err != nil {
			return
		}
	}
}

// handleCommand executes one decoded command and returns its RESP-encoded reply.
func handleCommand(args []string, store *storage.Storage) string {
	if len(args) == 0 {
		return resp.Error("ERR empty command")
	}

	switch strings.ToUpper(args[0]) {
	case "PING":
		if len(args) == 1 {
			return resp.SimpleString("PONG")
		}
		if len(args) == 2 {
			return resp.BulkString(args[1])
		}
		return resp.Error("ERR wrong number of arguments for 'ping' command")

	case "SET":
		if len(args) != 3 {
			return resp.Error("ERR wrong number of arguments for 'set' command")
		}

		key := args[1]
		value := args[2]

		store.Set(key, value)

		return resp.SimpleString("OK")

	case "GET":
		if len(args) != 2 {
			return resp.Error("ERR wrong number of arguments for 'get' command")
		}

		key := args[1]

		value, exists := store.Get(key)

		if !exists {
			return resp.NullBulkString()
		}

		return resp.BulkString(value)

	case "DEL":
		if len(args) < 2 {
			return resp.Error("ERR wrong number of arguments for 'del' command")
		}

		var deleted int64
		for _, key := range args[1:] {
			if store.Delete(key) {
				deleted++
			}
		}

		return resp.Integer(deleted)

	default:
		return resp.Error("ERR unknown command '" + args[0] + "'")
	}

}
