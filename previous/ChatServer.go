package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

type Client struct {
	conn     net.Conn
	nickname string
}

var (
	clients    sync.Map
	serverPort = "22253"
)

func main() {
	// Server Socket
	listener, err := net.Listen("tcp", ":"+serverPort)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer listener.Close() // Ensure listener socket is closed

	fmt.Printf("The server is ready to receive on port %s\n", serverPort)

	// Handles abrupt exit
	abruptCloseHandler()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err.Error())
			continue
		}

		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()

	// Check if num of Clients < 8
	if getClientCount() >= 8 {
		msg := "[chatting room full. cannot connect]\n"
		conn.Write([]byte("\\warning:" + msg))
	}

	reader := bufio.NewReader(conn)
	nickname, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading nickname:", err.Error())
		return
	}
	nickname = strings.TrimSpace(nickname)
	// Check if the nickname is already in use
	if _, exists := clients.Load(nickname); exists {
		conn.Write([]byte("\\warning:[nickname already used by another user. cannot connect]\n"))
		return
	}

	client := &Client{
		conn:     conn,
		nickname: nickname,
	}
	// Nickname is ok to use so store client in syncMap
	clients.Store(nickname, client)

	fmt.Fprintf(conn, "\\server:[Welcome %s to CAU net-class chat room at %s. There are %d users in the room]\n",
		nickname, conn.LocalAddr().String(), getClientCount())

	broadcastExcept(nickname, fmt.Sprintf("\\server:[%s joined from %s. There are %d users in the room]\n",
		nickname, conn.RemoteAddr().String(), getClientCount()))

	buffer := make([]byte, 1024)
	for {
		count, err := client.conn.Read(buffer)
		if err != nil {
			if _, ok := clients.Load(client.nickname); ok { //Check if Client is still in syncMap list
				// Client still in the map, which means, unexpected connection loss
				handleDisconnect(client)
			}
			break
		}
		handleMessage(client, string(buffer[:count]))
	}
}

func readMessages(client *Client) {
	buffer := make([]byte, 1024)
	for {
		count, err := client.conn.Read(buffer)
		if err != nil {
			if _, ok := clients.Load(client.nickname); ok { //Check if Client is still in syncMap list
				// Client still in the map, which means, unexpected connection loss
				handleDisconnect(client)
			}
			return
		}
		handleMessage(client, string(buffer[:count]))
	}
}

func handleMessage(sender *Client, message string) {
	parts := strings.Split(message, ":")
	command := parts[0] // Command: User input (e.g. 0,1,2,3)
	text := parts[1]    // Text: user message part
	ignoreCase := false

	// Check if the message contains "I hate professor"
	if strings.Contains(strings.ToLower(message), "i hate professor") {
		fmt.Fprintf(sender.conn, "\\warning:[You have been disconnected for inappropriate behavior]\n")
		// Disconnect the client
		clients.Delete(sender.nickname)
		ignoreCase = true
	}

	// Handle command
	switch command {
	case "0": // just normal chat message
		broadcastMessage(sender.nickname + ":" + text)
		if ignoreCase {
			handleIgnoreCase(sender)
		}
	case "1": // \ls
		listConnectedUsers(sender)
	case "2": // \secret
		parts := strings.SplitN(text, "\\", 2)
		secretReceiver := parts[0]
		textMessage := parts[1]

		// Find the client with the matching nickname
		secretClient := findClientByNickname(secretReceiver)

		// If the client is found, send the secret message
		if secretClient != nil {
			secretClient.conn.Write([]byte("\\" + sender.nickname + ":" + textMessage + "\n"))
			if ignoreCase {
				handleIgnoreCase(sender)
			}
		} else {
			sender.conn.Write([]byte(fmt.Sprintf("\\server:[No client found with the nickname %s]\n", secretReceiver)))
		}
	case "3": // \except
		parts := strings.SplitN(text, "\\", 2)
		excepted := parts[0]
		textMessage := parts[1]

		// Find the client with the matching nickname
		exceptedClient := findClientByNickname(excepted)
		if exceptedClient != nil {
			broadcastExcept(excepted, sender.nickname+":"+textMessage)
			if ignoreCase {
				handleIgnoreCase(sender)
			}
		} else {
			sender.conn.Write([]byte(fmt.Sprintf("\\server:[No client found with the nickname %s\n", excepted)))
		}
	case "4": // \ping
		sender.conn.Write([]byte("\\ping:\n"))
	case "-2": // \exit
		handleDisconnect(sender)
	default: // invalid command
		sender.conn.Write([]byte("\\server:[invalid command]\n"))
	}
}

func listConnectedUsers(sender *Client) {
	userList := "Connected Users:\n"
	clients.Range(func(key, value interface{}) bool {
		client := value.(*Client)
		if client != nil {
			userList += fmt.Sprintf("%s: %s\n", client.nickname, client.conn.RemoteAddr().(*net.TCPAddr))
		}
		return true
	})
	sender.conn.Write([]byte("\\server:[" + userList + "]\n"))
}

func broadcastMessage(message string) {
	clients.Range(func(key, value interface{}) bool {
		client := value.(*Client)
		client.conn.Write([]byte(message))
		return true
	})
}

func broadcastExcept(excludedNickname string, message string) {
	clients.Range(func(key, value interface{}) bool {
		client := value.(*Client)
		if client.nickname != excludedNickname && client != nil {
			client.conn.Write([]byte(message))
		}
		return true
	})
}

func handleDisconnect(client *Client) {
	clients.Delete(client.nickname)
	client.conn.Close()
	broadcastMessage(fmt.Sprintf("\\server:[%s left the room. There are %d users now]\n",
		client.nickname, getClientCount()))
}
func getClientCount() int {
	count := 0
	clients.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

func findClientByNickname(nickname string) *Client {
	var targetClient *Client
	clients.Range(func(key, value interface{}) bool {
		client := value.(*Client)
		if client.nickname == nickname {
			targetClient = client
			return false // Stop the iteration once the client is found
		}
		return true
	})
	return targetClient
}

// Handle Ignore case
func handleIgnoreCase(sender *Client) {
	sender.conn.Close()
	broadcastMessage(fmt.Sprintf("\\server:[%s has been disconnected. There are %d users in the chat room.]\n",
		sender.nickname, getClientCount()))
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
