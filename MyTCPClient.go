/**
 * 20192253 Hongchan Yoon
 **/

package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	// Server config
	serverName := "nsl2.cau.ac.kr"
	serverPort := "22253"
	fmt.Println("hi")
	// Initialize conn (connect only once!)
	conn, err := net.Dial("tcp", serverName+":"+serverPort)
	// If error occurs during initialization, exit the program
	if err != nil {
		fmt.Println("Error listening:", err)
		os.Exit(1)
	}
	defer func() {
		conn.Close() // Close pconn when main() exits
		fmt.Println("hello")
	}()
	localAddr := conn.LocalAddr().(*net.TCPAddr)
	fmt.Printf("The client is running on port %d\n", localAddr.Port)

	// Handles abrupt exit
	abruptCloseHandler()

	for {
		// Displays menu
		showMenu()

		// Option Reader for client (i.e., 1,2,3,4,5)
		optionReader := bufio.NewReader(os.Stdin)
		option, _ := optionReader.ReadString('\n')
		// Trim white spaces if there are any
		option = strings.TrimSpace(option)

		switch option {
		case "1":
			fmt.Print("Input sentence: ")
			input, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			input = strings.TrimRight(input, "\n") // Remove newline character
			sendCommand(conn, "1", input)
		case "2":
			sendCommand(conn, "2", "")
		case "3":
			sendCommand(conn, "3", "")
		case "4":
			sendCommand(conn, "4", "")
		case "5":
			fmt.Println("Bye bye~")
			conn.Close()
			os.Exit(0)
		default:
			fmt.Print("Invalid option. Please try again.\n\n")
		}
	}
}

func showMenu() {
	fmt.Println("<Menu>")
	fmt.Println("1) convert text to UPPER-case")
	fmt.Println("2) get server running time")
	fmt.Println("3) get my IP address and port number")
	fmt.Println("4) get server request count")
	fmt.Println("5) exit")
	fmt.Print("Input option: ")
}

func sendCommand(conn net.Conn, command, text string) {

	// Message format: "command:text" e.g. 1:hello
	message := []byte(command + ":" + text)
	// Time starts right before sending message to the server
	start := time.Now()
	conn.Write(message)

	// Set a timeout for reading from the connection
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	buffer := make([]byte, 1024)

	count, err := conn.Read(buffer)
	if err != nil {
		fmt.Print("Server not responding. Please try again later.\n\n")
		return
	}

	// Time ends right after receiving message from the server
	elapsed := time.Since(start)

	// Prints out reply from server and RTT
	fmt.Printf("\nReply from server: %s\n", string(buffer[:count]))
	fmt.Printf("RTT = %.3f ms\n\n", elapsed.Seconds()*1000)
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
