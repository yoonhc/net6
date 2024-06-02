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
	 command = strings.TrimSpace(command)
	 parts := strings.Split(command, ":")
	 if len(parts) != 2 {
		 log.Printf("Invalid command: %s", command)
		 return
	 }
 
	 // action 1: <put>	action 2: <get>
	 action, partFileName := parts[0], parts[1]
	 log.Print(partFileName)
 
	 if action == "1" { // put command
		 receiveFile(reader, partFileName)
	 } else if action == "2" { // get command
		 sendFile(conn, partFileName)
	 } else {
		 log.Printf("Unknown action: %s", action)
	 }
 }
 
 // receive split file from the client
 func receiveFile(reader *bufio.Reader, partFileName string) {
	 file, err := os.Create(partFileName)
	 if err != nil {
		 log.Printf("Failed to create file: %v", err)
		 return
	 }
	 defer func() {
		 if file != nil {
			 if closeErr := file.Close(); closeErr != nil {
				 log.Printf("Error closing file %s: %v", partFileName, closeErr)
			 }
			 // If an error occurred and the file was created, delete the file
			 if err != nil {
				 if removeErr := os.Remove(partFileName); removeErr != nil {
					 log.Printf("Error deleting file %s: %v", partFileName, removeErr)
				 }
			 }
		 }
	 }()
 
	 _, err = io.Copy(file, reader)
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
		 sendFileExistenceMessage(conn, partFileName, false)
		 return
	 }
	 defer file.Close()
 
	 // file does exist. send ok message
	 sendFileExistenceMessage(conn, partFileName, true)
 
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
 
 // send response message to the client that requested file does/doesn't exist in the server
 func sendFileExistenceMessage(conn net.Conn, fileName string, exists bool) {
	 if !exists {
		 _, err := conn.Write([]byte("ERROR: <" + fileName + "> file does not exist!\n"))
		 if err != nil {
			 log.Printf("Failed to send error message: %v", err)
		 }
	 } else {
		 _, err := conn.Write([]byte("OK: <" + fileName + "> file does exist\n"))
		 if err != nil {
			 log.Printf("Failed to send ok message: %v", err)
		 }
	 }
 }
 