package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"bluis/internals/resp"
	"bluis/internals/storage"
)

func main() {
	listen, err := net.Listen("tcp", "localhost:6380")
	if err != nil {
		log.Fatal(err)
	}
	defer listen.Close()

	store := storage.NewStorage()

	err = store.LoadFromFile("dump.rdb")
	if err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}

	cleanupStop := make(chan struct{})
	go store.StartCleanup(cleanupStop)

	// Shutdown handling
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-signalChannel

		log.Println("Shutdown signal received:", sig)

		close(cleanupStop)

		log.Println("Saving data...")

		if err := store.SaveToFile("dump.rdb"); err != nil {
			log.Println("Error saving data:", err)
		}

		log.Println("Snapshot saved. Shutting down...")
		os.Exit(0)
	}()

	fmt.Println("BLUIS server is running...")

	//for {
	//	conn, err := listen.Accept()
	//	if err != nil {
	//		fmt.Println(err)
	//		continue
	//	}
	//
	//	fmt.Println("New client connected!")
	//	go handleClient(conn, store)
	//}

	serve(listen, store)
}

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
			if err != io.EOF {
				fmt.Println("RESP read error:", err)
				_, _ = conn.Write([]byte(resp.Error("ERR " + err.Error())))
			}
			return
		}

		fmt.Println("Received:", args)

		response := handleCommand(args, store)

		_, err = conn.Write([]byte(response))
		if err != nil {
			fmt.Println("Write error:", err)
			return
		}
	}
}

func handleCommand(args []string, store *storage.Storage) string {
	if len(args) == 0 {
		return resp.Error("ERR empty command")
	}

	switch strings.ToUpper(args[0]) {

	case "PING":
		if len(args) > 2 {
			return resp.Error("ERR wrong number of arguments")
		}
		if len(args) == 2 {
			return resp.BulkString(args[1])
		}
		return resp.SimpleString("PONG")

	case "SET":
		if len(args) != 3 {
			return resp.Error("ERR wrong number of arguments")
		}

		store.Set(args[1], args[2])
		return resp.SimpleString("OK")

	case "GET":
		if len(args) != 2 {
			return resp.Error("ERR wrong number of arguments")
		}

		value, exists := store.Get(args[1])
		if !exists {
			return resp.NullBulkString()
		}

		return resp.BulkString(value)

	case "DEL":
		if len(args) != 2 {
			return resp.Error("ERR wrong number of arguments")
		}

		_, exists := store.Get(args[1])
		store.Delete(args[1])

		if exists {
			return resp.Integer(1)
		}

		return resp.Integer(0)

	case "EXPIRE":
		if len(args) != 3 {
			return resp.Error("ERR wrong number of arguments")
		}

		seconds, err := strconv.Atoi(args[2])
		if err != nil {
			return resp.Error("ERR value is not an integer or out of range")
		}

		if store.Expire(args[1], seconds) {
			return resp.Integer(1)
		}

		return resp.Integer(0)

	case "TTL":
		if len(args) != 2 {
			return resp.Error("ERR wrong number of arguments")
		}

		return resp.Integer(int64(store.TTL(args[1])))

	default:
		return resp.Error("ERR unknown command")
	}
}
