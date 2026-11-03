package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"

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

	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Println("New client connected!")

		// buffer := make([]byte, 1024)

		// n, err := conn.Read(buffer)
		// if err != nil {
		// 	fmt.Println(err)
		// 	conn.Close()
		// 	continue
		// }

		// command := strings.TrimSpace(string(buffer[:n]))
		// fmt.Println("Received:", command)

		// response :=handleCommand(command, store)

		// _, err = conn.Write([]byte(response + "\n"))
		// if err != nil {
		// 	fmt.Println(err)
		// }

		// conn.Close()
		go handleClient(conn, store)
	}
}

func handleClient(conn net.Conn, store *storage.Storage) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		command, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		command = strings.TrimSpace(command)

		if command == "" {
			continue
		}

		fmt.Println("Received:", command)

		response := handleCommand(command, store)

		_, err = conn.Write([]byte(response + "\n"))
		if err != nil {
			break
		}
	}
}

func handleCommand(command string, store *storage.Storage) string {
	parts := strings.Fields(command)

	if len(parts) == 0 {
		return "-ERR empty command"
	}

	switch strings.ToUpper(parts[0]) {
	case "PING":
		return "PONG"

	case "SET":
		if len(parts) != 3 {
			return "-ERR wrong number of arguments"
		}

		key := parts[1]
		value := parts[2]

		store.Set(key, value)

		return "OK"
	case "GET":
		if len(parts) != 2 {
			return "-ERR wrong number of arguments"
		}

		key := parts[1]

		value, exists := store.Get(key)

		if !exists {
			return "(nil)"
		}

		return value

	case "DEL":
		if len(parts) != 2 {
			return "-ERR wrong number of arguments"
		}

		key := parts[1]

		store.Delete(key)

		return "OK"

	default:
		return "-ERR unknown command"
	}

}
