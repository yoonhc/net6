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
	 "os/signal"
	 "path/filepath"
	 "strconv"
	 "strings"
	 "sync"
	 "syscall"
 )
 
 // MODIFY SERVER ADDRESS HERE :)
 const (
	 server1 = "nsl2.cau.ac.kr:42253"
	 server2 = "nsl5.cau.ac.kr:52253"
 )
 
 func main() {
	 if len(os.Args) != 3 {
		 log.Fatalf("Usage: go run SplitFileClient.go <put/get> <filename>")
	 }
 
	 action := os.Args[1]   // <put/get> command
	 fileName := os.Args[2] // <filename>
 
	 if action == "put" {
		 putFile(fileName)
	 } else if action == "get" {
		 getFile(fileName)
	 } else {
		 log.Fatalf("Unknown action: %s. Use 'put' or 'get'!", action)
	 }
 }
 
 // this function is called when types put as command line argument
 func putFile(fileName string) {
	 // Open File
	 file, err := os.Open(fileName)
	 if err != nil {
		 log.Printf("Failed to open file: %v", err)
		 return
	 }
	 defer file.Close()
 
	 // Get file size
	 fileInfo, err := file.Stat()
	 if err != nil {
		 log.Printf("Failed to get file info: %v", err)
		 return
	 }
	 fileSize := fileInfo.Size()
 
	 // Connect to server1
	 conn1, err := net.Dial("tcp", server1)
	 if err != nil {
		 log.Printf("Failed to connect to server1: %v", err)
		 return
	 }
	 defer conn1.Close()
 
	 // Connect to server2
	 conn2, err := net.Dial("tcp", server2)
	 if err != nil {
		 log.Printf("Failed to connect to server2: %v", err)
		 return
	 }
	 defer conn2.Close()
 
	 // filename.xxx -> filename-part1.xxx / filename-part2.xxx
	 part1FileName := addSuffixToFileName(fileName, "part1")
	 part2FileName := addSuffixToFileName(fileName, "part2")
 
	 part1FileSize := (fileSize / 2) + 1
	 part2FileSize := (fileSize / 2)
 
	 // 1: put command, 2: get command
	 // sendCommand format: <1:fileName:fileSize>
	 if err := sendCommand(conn1, "1:"+part1FileName+":"+
		 strconv.FormatInt(part1FileSize, 10)); err != nil {
		 return
	 }
	 if err := sendCommand(conn2, "1:"+part2FileName+":"+
		 strconv.FormatInt(part2FileSize, 10)); err != nil {
		 return
	 }
 
	 var wg sync.WaitGroup
	 wg.Add(2)
 
	 // Create SectionReaders for each part
	 sectionReader1 := io.NewSectionReader(file, 0, fileSize)
	 sectionReader2 := io.NewSectionReader(file, 0, fileSize)
 
	 log.Println("Sending files to both servers...")
 
	 go func() {
		 defer wg.Done()
		 sendAlternateBytes(sectionReader1, conn1, 0)
	 }()
 
	 go func() {
		 defer wg.Done()
		 sendAlternateBytes(sectionReader2, conn2, 1)
	 }()
 
	 wg.Wait()
	 log.Println("File sent successfully to both servers")
 }
 
 // this function is called when types get as command line argument
 func getFile(fileName string) {
	 mergedFileName := addSuffixToFileName(fileName, "merged")
 
	 // connect to server1
	 conn1, err := net.Dial("tcp", server1)
	 if err != nil {
		 log.Printf("Failed to connect to server1: %v", err)
		 return
	 }
	 reader1 := bufio.NewReader(conn1)
	 defer conn1.Close()
 
	 // connect to server 2
	 conn2, err := net.Dial("tcp", server2)
	 if err != nil {
		 log.Printf("Failed to connect to server2: %v", err)
		 return
	 }
	 reader2 := bufio.NewReader(conn2)
	 defer conn2.Close()
 
	 part1FileName := addSuffixToFileName(fileName, "part1")
	 part2FileName := addSuffixToFileName(fileName, "part2")
 
	 // 1: put command, 2: get command
	 if err := sendCommand(conn1, "2:"+part1FileName+":"); err != nil {
		 return
	 }
	 if err := sendCommand(conn2, "2:"+part2FileName+":"); err != nil {
		 return
	 }
 
	 // Check whether both files exist in servers
	 part1FileSize, ok1 := checkServerResponse(reader1, "server1")
	 part2FileSize, ok2 := checkServerResponse(reader2, "server2")
 
	 if !ok1 || !ok2 {
		 log.Println("file retrieval failed due to missing parts on servers")
		 return
	 }
 
	 // Create merged file
	 mergedFile, err := os.Create(mergedFileName)
	 if err != nil {
		 log.Printf("Failed to create merged file: %v", err)
		 return
	 }
	 defer mergedFile.Close()
 
	 // sigint handler(when sigint, deletes in porgress file)
	 removeFileWhenInterrupt(mergedFileName)
 
	 // Channels for alternate byte writing
	 ch1 := make(chan byte)
	 ch2 := make(chan byte)
	 done := make(chan bool)
 
	 log.Println("Receiving files from both servers...")
 
	 // Receive and send data to channels
	 go receiveAlternateBytes(reader1, ch1)
	 go receiveAlternateBytes(reader2, ch2)
 
	 // Merge the bytes from both channels and write to merged file
	 go mergeAndWriteBytes(ch1, ch2, mergedFile, done)
 
	 <-done
	 // Get file size
	 fileInfo, err := mergedFile.Stat()
	 if err != nil {
		 log.Printf("Failed to get file info: %v", err)
		 return
	 }
	 mergedFileSize := fileInfo.Size()
 
	 if mergedFileSize != (part1FileSize + part2FileSize) {
		 log.Printf("Error occured while receiving <%s>", mergedFileName)
		 os.Remove(mergedFileName)
		 log.Printf("Removed file %s due to error", mergedFileName)
		 return
	 }
	 log.Printf("file <%s> received successfully", mergedFileName)
 }
 
 // send command(put/get with filename) to the server
 func sendCommand(conn net.Conn, action string) error {
	 _, err := conn.Write([]byte(action + "\n"))
	 if err != nil {
		 log.Printf("Failed to send action: %v", err)
		 return err
	 }
	 return nil
 }
 
 // send odd bytes(1,3,5...) when offset is 0, send even bytes(2,4,6...) otherwise
 func sendAlternateBytes(reader io.Reader, writer io.Writer, offset int) {
	 buffer := make([]byte, 1024) // chunk size: 1024 byte
 
	 for {
		 n, err := reader.Read(buffer)    // read 1024 bytes(A chunk) of file
		 if err != nil && err != io.EOF { // failed to read file
			 log.Fatalf("Failed to read file: %v", err)
		 }
		 if n == 0 { // number of bytes read from chunk is 0 which means EOF
			 break
		 }
 
		 for i := offset; i < n; i += 2 {
			 _, err := writer.Write(buffer[i : i+1])
			 if err != nil {
				 log.Fatalf("Failed to write data: %v", err)
			 }
		 }
	 }
 }
 
 // receive byte from server, save it in channel
 func receiveAlternateBytes(reader *bufio.Reader, ch chan<- byte) {
	 buffer := make([]byte, 1)
	 for {
		 n, err := reader.Read(buffer)
		 if err != nil && err != io.EOF {
			 log.Printf("Failed to receive data: %v", err)
			 break
		 }
		 if n == 0 { // successfully read all the file
			 break
		 }
		 ch <- buffer[0]
	 }
	 close(ch)
 }
 
 // add suffix to file name
 // e.g. addSuffixToFileName(hello.txt, part1) -> hello-part1.txt
 func addSuffixToFileName(fileName, suffix string) (filename string) {
	 extension := filepath.Ext(fileName)
	 nameWithoutExtension := strings.TrimSuffix(fileName, extension)
 
	 addedSuffixName := nameWithoutExtension + "-" + suffix + extension
	 return addedSuffixName
 }
 
 // Merge alternating byte in ch1 and ch2 to file
 func mergeAndWriteBytes(ch1, ch2 <-chan byte, file *os.File, done chan<- bool) {
	 var b1, b2 byte
	 var ok1, ok2 bool
	 readFromCh1 := true
 
	 for {
		 if readFromCh1 { // Time to get byte from channel1
			 b1, ok1 = <-ch1
			 if ok1 {
				 _, err := file.Write([]byte{b1})
				 if err != nil {
					 log.Printf("Failed to write to merged file: %v", err)
				 }
				 readFromCh1 = false
			 } else {
				 break
			 }
		 } else { // Time to get byte from channel2
			 b2, ok2 = <-ch2
			 if ok2 { // channel 2 good to go
				 _, err := file.Write([]byte{b2})
				 if err != nil {
					 log.Printf("Failed to write to merged file: %v", err)
				 }
				 readFromCh1 = true
			 } else {
				 break
			 }
		 }
	 }
	 done <- true
 }
 
 // returns false when there is an error
 func checkServerResponse(reader *bufio.Reader, serverName string) (int64, bool) {
	 response, err := reader.ReadString('\n')
	 if err != nil {
		 log.Printf("Failed to read response from %s: %v", serverName, err)
		 return 0, false
	 }
	 response = strings.TrimSpace(response)
	 if strings.HasPrefix(response, "ERROR:") {
		 log.Printf("Error from %s: %s", serverName, response)
		 return 0, false
	 }
	 if strings.HasPrefix(response, "OK:") {
		 // Extract file size from the response
		 fileSizeString := strings.TrimPrefix(response, "OK:")
		 fileSizeString = strings.TrimSpace(fileSizeString)
		 fileSize, _ := strconv.ParseInt(fileSizeString, 10, 64)
		 return fileSize, true
	 }
	 return 0, false
 }
 
 // Remove in-progress(receiving file) when interrupt
 func removeFileWhenInterrupt(mergedFileName string) {
	 sigChan := make(chan os.Signal, 1)
	 signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	 go func() {
		 <-sigChan
		 err := os.Remove(mergedFileName)
		 if err != nil {
			 log.Printf("Failed to delete in-progress file %s: %v", mergedFileName, err)
		 }
		 log.Printf("In-progress file <%s> deleted successfully", mergedFileName)
		 os.Exit(0)
	 }()
 }
 