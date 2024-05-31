/**
 * 20192253 Hongchan Yoon
 **/

package main

import (
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <port>", os.Args[0])
	}
	port := os.Args[1]

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to set up listener: %v", err)
	}
	defer listener.Close()
	log.Printf("Server listening on port %s", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil {
		log.Printf("Failed to read command: %v", err)
		return
	}
	command := strings.TrimSpace(string(buf[:n]))

	parts := strings.Split(command, " ")
	if len(parts) != 2 {
		log.Printf("Invalid command: %s", command)
		return
	}

	action, partFileName := parts[0], parts[1]
	log.Print(partFileName)

	if action == "put" {
		receiveFile(conn, partFileName)
	} else if action == "get" {
		sendFile(conn, partFileName)
	} else {
		log.Printf("Unknown action: %s", action)
	}
}

// receive split file from the client
func receiveFile(conn net.Conn, partFileName string) {
	file, err := os.Create(partFileName)
	if err != nil {
		log.Printf("Failed to create file: %v", err)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, conn)
	if err != nil {
		log.Printf("Failed to write data to file: %v", err)
		return
	}

	log.Printf("File %s received successfully", partFileName)
}

// send split file to the client
func sendFile(conn net.Conn, partFileName string) {
	file, err := os.Open(partFileName)
	if err != nil {
		log.Printf("Failed to open file: %v", err)
		return
	}
	defer file.Close()

	_, err = io.Copy(conn, file)
	if err != nil {
		log.Printf("Failed to send data: %v", err)
		return
	}

	log.Printf("File %s sent successfully", partFileName)
}

// add suffix to file name
// e.g. addSuffixToFileName(hello.txt, part1) -> hello-part1.txt
func addSuffixToFileName(fileName, suffix string) (filename string) {
	extension := filepath.Ext(fileName)
	nameWithoutExtension := strings.TrimSuffix(fileName, extension)

	addedSuffixName := nameWithoutExtension + "-" + suffix + extension
	return addedSuffixName
}
