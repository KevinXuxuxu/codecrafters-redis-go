package main

import (
	"fmt"
	// Uncomment this block to pass the first stage
	"net"
	"os"
)

func handleRequest(conn net.Conn) {
	defer conn.Close()
	for {
		if _, err := conn.Read([]byte{}); err != nil {
			fmt.Println("Error reading data from connection: ", err.Error())
			os.Exit(1)
		}
		// if _, err := conn.Write([]byte("+PONG\r\n")); err != nil {
		// 	fmt.Println("Error sending data to connection: ", err.Error())
		// 	os.Exit(1)
		// }
		conn.Write([]byte("+PONG\r\n"))
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
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleRequest(conn)
	}
}
