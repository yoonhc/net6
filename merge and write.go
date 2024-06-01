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
	"sync"
)

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

func putFile(fileName string) {
	// Open File
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatalf("Failed to get file info: %v", err)
	}
	fileSize := fileInfo.Size()

	// Connect to server1
	conn1, err := net.Dial("tcp", server1)
	if err != nil {
		log.Fatalf("Failed to connect to server1: %v", err)
	}
	defer conn1.Close()

	// Connect to server2
	conn2, err := net.Dial("tcp", server2)
	if err != nil {
		log.Fatalf("Failed to connect to server2: %v", err)
	}
	defer conn2.Close()

	// filename.xxx -> filename-part1.xxx / filename-part2.xxx
	part1FileName := addSuffixToFileName(fileName, "part1")
	part2FileName := addSuffixToFileName(fileName, "part2")

	// 0: put command, 1: get command
	sendCommand(conn1, "0:"+part1FileName)
	sendCommand(conn2, "0:"+part2FileName)

	var wg sync.WaitGroup
	wg.Add(2)

	// Create SectionReaders for each part
	sectionReader1 := io.NewSectionReader(file, 0, fileSize)
	sectionReader2 := io.NewSectionReader(file, 0, fileSize)

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

func getFile(fileName string) {
	mergedFileName := addSuffixToFileName(fileName, "merged")

	// connect to server1
	conn1, err := net.Dial("tcp", server1)
	if err != nil {
		log.Fatalf("Failed to connect to server1: %v", err)
	}
	defer conn1.Close()

	// connect to server 2
	conn2, err := net.Dial("tcp", server2)
	if err != nil {
		log.Fatalf("Failed to connect to server2: %v", err)
	}
	defer conn2.Close()

	part1FileName := addSuffixToFileName(fileName, "part1")
	part2FileName := addSuffixToFileName(fileName, "part2")

	// 0: put command, 1: get command
	sendCommand(conn1, "1:"+part1FileName)
	sendCommand(conn2, "1:"+part2FileName)

	// Create merged file
	mergedFile, err := os.Create(mergedFileName)
	if err != nil {
		log.Fatalf("Failed to create merged file: %v", err)
	}
	defer mergedFile.Close()

	// Channels for alternate byte writing
	ch1 := make(chan byte)
	ch2 := make(chan byte)
	done := make(chan bool)

	// Receive and send data to channels
	go receiveAlternateBytes(conn1, ch1)
	go receiveAlternateBytes(conn2, ch2)

	// Merge the bytes from both channels and write to merged file
	go mergeAndWriteBytes(ch1, ch2, mergedFile, done)

	<-done
	log.Println("File received and merged successfully")
}
func sendCommand(conn net.Conn, action string) {
	_, err := conn.Write([]byte(action + "\n"))
	if err != nil {
		log.Fatalf("Failed to send action: %v", err)
	}
}

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

func receiveAlternateBytes(conn net.Conn, ch chan<- byte) {
	buffer := make([]byte, 1)
	for {
		n, err := conn.Read(buffer)
		if err != nil && err != io.EOF {
			log.Fatalf("Failed to read data: %v", err)
		}
		if n == 0 {
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

func mergeAndWriteBytes(ch1, ch2 <-chan byte, file *os.File, done chan<- bool) {
	var b1, b2 byte
	var ok1, ok2 bool
	readFromCh1 := true

	for {
		if readFromCh1 {
			b1, ok1 = <-ch1
			if ok1 {
				_, err := file.Write([]byte{b1})
				if err != nil {
					log.Fatalf("Failed to write to merged file: %v", err)
				}
				readFromCh1 = false
			} else {
				b2, ok2 = <-ch2
				if !ok2 {
					break
				} else {
					readFromCh1 = false
				}
			}
		} else {
			b2, ok2 = <-ch2
			if ok2 {
				_, err := file.Write([]byte{b2})
				if err != nil {
					log.Fatalf("Failed to write to merged file: %v", err)
				}
				readFromCh1 = true
			} else {
				b1, ok1 = <-ch1
				if !ok1 {
					break
				} else {
					readFromCh1 = true
				}
			}
		}
	}
	done <- true
}
