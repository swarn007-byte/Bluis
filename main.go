
package main

import (
	"fmt"
	"net"
	"strings"

	"bluis/internal/storage"
)

func main() {
	listen, err := net.Listen("tcp", "localhost:6379")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer listen.Close()

	store := storage.NewStore()

	fmt.Println("BLUIS server is running...")

	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Println("New client connected!")

		buffer := make([]byte, 1024)

		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println(err)
			conn.Close()
			continue
		}

		command := strings.TrimSpace(string(buffer[:n]))
		fmt.Println("Received:", command)

		response := handleCommand(command, store)

		_, err = conn.Write([]byte(response + "\n"))
		if err != nil {
			fmt.Println(err)
		}

		conn.Close()
	}
}

func handleCommand(command string, store *storage.Store) string {
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
    
        store.Del(key)
    
        return "OK"

	default:
		return "-ERR unknown command"
	}
	
}
