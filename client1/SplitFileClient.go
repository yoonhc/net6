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
	server1Address = "nsl2.cau.ac.kr:42253"
	server2Address = "nsl5.cau.ac.kr:52253"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Usage: %s <put/get> <filename>", os.Args[0])
	}

	action := os.Args[1]   // <put/get>
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
	conn1, err := net.Dial("tcp", server1Address)
	if err != nil {
		log.Fatalf("Failed to connect to server1: %v", err)
	}
	defer conn1.Close()

	// Connect to server2
	conn2, err := net.Dial("tcp", server2Address)
	if err != nil {
		log.Fatalf("Failed to connect to server2: %v", err)
	}
	defer conn2.Close()

	part1FileName := addSuffixToFileName(fileName, "part1")
	part2FileName := addSuffixToFileName(fileName, "part2")
	sendCommand(conn1, "put "+part1FileName)
	sendCommand(conn2, "put "+part2FileName)

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
	part1FileName := addSuffixToFileName(fileName, "part1")
	part2FileName := addSuffixToFileName(fileName, "part2")
	mergedFileName := addSuffixToFileName(fileName, "merged")

	conn1, err := net.Dial("tcp", server1Address)
	if err != nil {
		log.Fatalf("Failed to connect to server1: %v", err)
	}
	defer conn1.Close()

	conn2, err := net.Dial("tcp", server2Address)
	if err != nil {
		log.Fatalf("Failed to connect to server2: %v", err)
	}
	defer conn2.Close()

	sendCommand(conn1, "get "+part1FileName)
	sendCommand(conn2, "get "+part2FileName)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		receiveFile(conn1, part1FileName)
	}()

	go func() {
		defer wg.Done()
		receiveFile(conn2, part2FileName)
	}()

	wg.Wait()

	mergeFiles(mergedFileName, part1FileName, part2FileName)
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
		n, err := reader.Read(buffer)    // reads 1024 bytes(chunk) of file
		if err != nil && err != io.EOF { // failed to read file
			log.Fatalf("Failed to read file: %v", err)
		}
		if n == 0 { // number of bytes read from chunk = 0 which means EOF
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

func receiveFile(conn net.Conn, fileName string) {
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	// Copy data from the connection to the file
	_, err = io.Copy(file, conn)
	if err != nil {
		log.Fatalf("Failed to receive data: %v", err)
	}
}

// add suffix to file name
// e.g. addSuffixToFileName(hello.txt, part1) -> hello-part1.txt
func addSuffixToFileName(fileName, suffix string) (filename string) {
	extension := filepath.Ext(fileName)
	nameWithoutExtension := strings.TrimSuffix(fileName, extension)

	addedSuffixName := nameWithoutExtension + "-" + suffix + extension
	return addedSuffixName
}

func mergeFiles(outputFileName, part1FileName, part2FileName string) {
	part1File, err := os.Open(part1FileName)
	if err != nil {
		log.Fatalf("Failed to open part1 file: %v", err)
	}
	defer part1File.Close()

	part2File, err := os.Open(part2FileName)
	if err != nil {
		log.Fatalf("Failed to open part2 file: %v", err)
	}
	defer part2File.Close()

	mergedFile, err := os.Create(outputFileName)
	if err != nil {
		log.Fatalf("Failed to create merged file: %v", err)
	}
	defer mergedFile.Close()

	buf1 := make([]byte, 1)
	buf2 := make([]byte, 1)

	for {
		n1, err1 := part1File.Read(buf1)
		n2, err2 := part2File.Read(buf2)

		if n1 > 0 {
			_, err := mergedFile.Write(buf1[:n1])
			if err != nil {
				log.Fatalf("Failed to write to merged file: %v", err)
			}
		}

		if n2 > 0 {
			_, err := mergedFile.Write(buf2[:n2])
			if err != nil {
				log.Fatalf("Failed to write to merged file: %v", err)
			}
		}

		// both file pointers reached to EOF
		if err1 == io.EOF && err2 == io.EOF {
			log.Println("File received and merged successfully")
			break
		}
		// error other than io.EOF occured when reading part1 file
		if err1 != nil && err1 != io.EOF {
			log.Fatalf("Error reading part1 file: %v", err1)
		}
		// error other than io.EOF occured when reading part2 file
		if err2 != nil && err2 != io.EOF {
			log.Fatalf("Error reading part2 file: %v", err2)
		}
	}
}
