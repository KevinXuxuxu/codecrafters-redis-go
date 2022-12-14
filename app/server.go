package main

import (
	"bufio"
	"fmt"

	"net"
	"os"
)

func handleRequest(conn net.Conn, data *SafeMap) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		object, err := ParseRESP(reader)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		if object == nil {
			break
		}
		conn.Write([]byte(object.response(data)))
	}
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	data := newSafeMap()
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleRequest(conn, data)
	}
}
