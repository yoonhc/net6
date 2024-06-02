/**
 * 20192253 Hongchan Yoon
 **/

package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: go run SplitFileServer.go <port>")
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

	reader := bufio.NewReader(conn)

	// command format: <1:filename>
	command, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			log.Println("Connection closed by client")
			return
		}
		log.Printf("Failed to read command: %v", err)
		return
	}
	// command format: <1:fileName:fileSize>
	command = strings.TrimSpace(command)
	parts := strings.Split(command, ":")
	if len(parts) != 3 {
		log.Printf("Invalid command: %s", command)
		return
	}

	// action 1: <put>	action 2: <get>
	action, partFileName := parts[0], parts[1]
	partFileSize, _ := strconv.ParseInt(parts[2], 10, 64)

	if action == "1" { // put command
		receiveFile(reader, partFileName, partFileSize)
	} else if action == "2" { // get command
		sendFile(conn, partFileName)
	} else {
		log.Printf("Unknown action: %s", action)
	}
}

// receive split file from the client
func receiveFile(reader *bufio.Reader, partFileName string, partFileSize int64) {
	file, err := os.Create(partFileName)
	if err != nil {
		log.Printf("Failed to create file: %v", err)
		return
	}
	defer file.Close()

	log.Printf("Receiving file <%s>...", partFileName)

	_, err = io.Copy(file, reader)
	if err != nil {
		log.Printf("Failed to write data to file: %v", err)
		os.Remove(partFileName)
		log.Printf("Removed file <%s> due to error", partFileName)
		return
	}
	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		log.Printf("Failed to get file info: %v", err)
		return
	}
	receivedFileSize := fileInfo.Size()

	if receivedFileSize != partFileSize {
		log.Printf("Error occured while receiving %s", partFileName)
		os.Remove(partFileName)
		log.Printf("Removed file <%s> due to error", partFileName)
		return
	}
	log.Printf("file <%s> received successfully", partFileName)
}

// send split file to the client
func sendFile(conn net.Conn, partFileName string) {
	file, err := os.Open(partFileName)
	if err != nil {
		log.Printf("Failed to open file: %v", err)
		sendErrorMessage(conn, partFileName)
		return
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		log.Printf("Failed to get file info: %v", err)
		return
	}
	partFileSize := fileInfo.Size()

	// file does exist. send ok message
	sendOKMessage(conn, partFileName, partFileSize)

	log.Printf("Sending file <%s>...", partFileName)
	_, err = io.Copy(conn, file)
	if err != nil {
		log.Printf("Failed to send data: %v", err)
		return
	}

	log.Printf("file <%s> sent successfully", partFileName)
}

// add suffix to file name
// e.g. addSuffixToFileName(hello.txt, part1) -> hello-part1.txt
func addSuffixToFileName(fileName, suffix string) (filename string) {
	extension := filepath.Ext(fileName)
	nameWithoutExtension := strings.TrimSuffix(fileName, extension)

	addedSuffixName := nameWithoutExtension + "-" + suffix + extension
	return addedSuffixName
}

// send response message to the client that requested file does/doesn't exist in the server
func sendErrorMessage(conn net.Conn, fileName string) {
	_, err := conn.Write([]byte("ERROR: <" + fileName + "> file does not exist!\n"))
	if err != nil {
		log.Printf("Failed to send error message: %v", err)
	}
}

// send response message to the client that requested file does/doesn't exist in the server
func sendOKMessage(conn net.Conn, fileName string, partFileSize int64) {
	_, err := conn.Write([]byte("OK:" + strconv.FormatInt(partFileSize, 10) + "\n"))
	if err != nil {
		log.Printf("Failed to send error message: %v", err)
	}
}
