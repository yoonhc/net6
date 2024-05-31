/**
 * 20192253 Hongchan Yoon
 **/

package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	requestCount    int                      // Variable to store the number of client requests
	requestMu       sync.Mutex               // Mutex for concurrent access to requestCount variable
	startTime       = time.Now()             // Server start time
	clientIDCounter int                      // Counter for assigning client IDs
	clientIDs       = make(map[net.Addr]int) // Map to store client IDs
	clientMu        sync.Mutex               // Mutex for concurrent access to clientIDs map
)

func main() {
	serverPort := "22253"

	// Server Socket
	listener, err := net.Listen("tcp", ":"+serverPort)
	if err != nil {
		fmt.Println("Error listening:", err)
		os.Exit(1)
	}
	defer listener.Close() // Close listener when main() exits

	fmt.Printf("The server is ready to receive on port %s\n", serverPort)

	// Handles abrupt exit
	abruptCloseHandler()

	// Print number of clients every 10 seconds
	go printClientCountPeriodically()

	// Loop forever
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		// Assign unique ID to client and print out connection info
		clientID := assignClientID(conn.RemoteAddr())

		// Can handle multiple client requests by using go routine
		go handleClientRequest(conn, clientID)
	}
}

func handleClientRequest(conn net.Conn, clientID int) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	connectionMsg := fmt.Sprintf("Connection Request from %s", conn.RemoteAddr().(*net.TCPAddr))

	// loop until there's an error (e.g. client closed connection)
	for {
		count, err := conn.Read(buffer)
		if err != nil {
			// Client disconnected
			clientMu.Lock()
			delete(clientIDs, conn.RemoteAddr())

			// Print out the (1) current time, (2) client ID, and the (3) number of clients
			fmt.Printf("[Time: %s] Client %d disconnected. Number of clients connected = %d\n",
				getTimeNow(), clientID, len(clientIDs))
			clientMu.Unlock()

			break
		}

		// Increments request count
		requestMu.Lock()
		requestCount++
		requestMu.Unlock()

		// Split message into 2 parts by ":"
		message := string(buffer[:count])
		parts := strings.SplitN(message, ":", 2)

		if len(parts) != 2 {
			fmt.Println("Invalid message format")
			continue
		}

		command := parts[0] // Command: User input (e.g. 1,2,3,4)
		text := parts[1]    // Text: text to convert to Uppercase

		// Handle command
		switch command {
		case "1":
			fmt.Printf("%s Command 1\n", connectionMsg)
			response := strings.ToUpper(text)
			conn.Write([]byte(response))
		case "2":
			fmt.Printf("%s Command 2\n", connectionMsg)
			response := fmt.Sprintf("run time = %s", getServerRunningTime())
			conn.Write([]byte(response))
		case "3":
			fmt.Printf("%s Command 3\n", connectionMsg)
			response := fmt.Sprintf("client IP = %s, port = %d",
				conn.RemoteAddr().(*net.TCPAddr).IP, conn.RemoteAddr().(*net.TCPAddr).Port)
			conn.Write([]byte(response))
		case "4":
			fmt.Printf("%s Command 4\n", connectionMsg)
			response := fmt.Sprintf("requests served = %d", getRequestCount())
			conn.Write([]byte(response))
		default:
			fmt.Println("Invalid command:", command)
		}
	}
}

// Returns server running time HH:MM:SS
func getServerRunningTime() string {
	duration := time.Since(startTime).Round(time.Second)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// returns total request count
func getRequestCount() int {
	requestMu.Lock()
	defer requestMu.Unlock()
	return requestCount
}

// Handles abrut close such as control c exit
func abruptCloseHandler() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-interrupt
		fmt.Println("\nBye bye~")
		os.Exit(0)
	}()
}

// Assigns a unique client ID print out connection info
func assignClientID(addr net.Addr) int {
	clientMu.Lock()
	defer clientMu.Unlock()

	// Increment the clientIDCounter to get the next available client ID
	clientIDCounter++

	// Assign the new client ID to the provided network address
	clientIDs[addr] = clientIDCounter

	// Print out the (1) current time, (2) client ID, and the (3) number of clients
	fmt.Printf("[Time: %s] Client %d connected. Number of clients connected = %d\n",
		getTimeNow(), clientIDCounter, len(clientIDs))

	return clientIDCounter
}

func getTimeNow() string {
	return time.Now().Format("15:04:05")
}

// Prints the number of connected clients every 10 seconds
func printClientCountPeriodically() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			clientMu.Lock()
			numClients := len(clientIDs)
			fmt.Printf("[Time: %s] Number of clients connected = %d\n", getTimeNow(), numClients)
			clientMu.Unlock()
		}
	}
}
