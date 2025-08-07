package internal

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-toast/toast"
	"github.com/google/uuid"
)

const (
	pingMessage  = "_PING_\n"
	pongMessage  = "_PONG_\n"
	pingInterval = 4 * time.Minute
	pingTimeout  = 5 * time.Second
	addr         = "localhost:8081"
	isConnected  = false

	// client certificate and key file paths
	certificateFile = "certificates/client.crt"
	keyFile         = "certificates/client.key"

	// event names
	NEW_MESSAGE            = "NEW_MESSAGE"
	EDIT_MESSAGE           = "EDIT_MESSAGE"
	DELETE_MESSAGE         = "DELETE_MESSAGE"
	MESSAGE_RECEIVED       = "MARK_MESSAGE_RECEIVED"
	GROUP_MESSAGE_READ     = "GROUP_MESSAGE_READ"
	ADDED_USER_TO_GROUP    = "ADD_USER_TO_GROUP"
	REMOVE_USER_FROM_GROUP = "REMOVE_USER_FROM_GROUP"
	MADE_ADMIN             = "MADE_ADMIN"
	REMOVE_ADMIN           = "REMOVE_ADMIN"
)

// quit channel to signal close the socket connection to server when disconnect command is called
var quit = make(chan struct{})

// group information for group event
type group struct {
	id          uuid.UUID
	username    string
	phonenumber string
}

// all group events will be unmarshalled into this struct
type groupEvent struct {
	Name      string `json:"name"`
	Group     group  `json:"group"`
	EmittedAt string `json:"emittedAt"`
}

// Message data for NEW_MESSAGE | EDIT_MESSAGE event
type newOrEditMessage struct {
	ID             uuid.UUID `json:"id"`
	GroupID        uuid.UUID `json:"group_id,omitempty"`
	SenderID       uuid.UUID `json:"sender_id"`
	SenderUsername string    `json:"sender_username,omitempty"`
	Description    string    `json:"description"`
	CreatedAt      string    `json:"created_at,omitempty"`
	UpdatedAt      string    `json:"updated_at,omitempty"`
}

// Message data for DELETE_MESSAGE event
type deleteMessage struct {
	ID       uuid.UUID `json:"id"`
	SenderID uuid.UUID `json:"sender_id"`
	GroupID  uuid.UUID `json:"group_id,omitempty"`
}

// Message data for MESSAGE_RECEIVED event
type markMessageReceived struct {
	ID         uuid.UUID `json:"id"`
	ReceiverID uuid.UUID `json:"receiver_id"`
}

// Message data for GROUP_MESSAGE_READ event
type markGroupMessageRead struct {
	ID                  uuid.UUID `json:"id"`
	GroupID             uuid.UUID `json:"group_id"`
	GroupMemberID       uuid.UUID `json:"group_member_id"`
	GroupMemberUsername string    `json:"group_member_username"`
}

func eventParser(event []byte) error {
	// finding the index of separator pipe '|'
	pipeIndex := bytes.Index(event, []byte("|"))
	if pipeIndex == -1 {
		return errors.New("message malfunctioned")
	}

	eventName := string(event[:pipeIndex])
	switch eventName {
	case NEW_MESSAGE:
		// show notification for new message
		message := &newOrEditMessage{}
		err := json.Unmarshal(event[pipeIndex+1:], message)
		if err != nil {
			return err
		}

		notification := toast.Notification{
			AppID:   "TerTerChat",
			Title:   message.SenderUsername,
			Message: message.Description,
		}
		err = notification.Push()
		if err != nil {
			return err
		}
	case EDIT_MESSAGE:
		message := &newOrEditMessage{}
		err := json.Unmarshal(event[pipeIndex+1:], message)
		if err != nil {
			return err
		}

		notification := toast.Notification{
			AppID:   "TerTerChat",
			Title:   message.SenderUsername,
			Message: message.Description,
		}
		err = notification.Push()
		if err != nil {
			return err
		}
	case DELETE_MESSAGE:
		message := &deleteMessage{}
		err := json.Unmarshal(event[pipeIndex+1:], message)
		if err != nil {
			return err
		}

		notification := toast.Notification{
			AppID:   "TerTerChat",
			Title:   "message deleted",
			Message: message.ID.String(),
		}
		if err = notification.Push(); err != nil {
			return err
		}
	case MESSAGE_RECEIVED:
		message := &markMessageReceived{}
		err := json.Unmarshal(event[pipeIndex+1:], message)
		if err != nil {
			return err
		}

		notification := toast.Notification{
			AppID:   "TerTerChat",
			Title:   "message received",
			Message: message.ID.String(),
		}
		if err = notification.Push(); err != nil {
			return err
		}
	case GROUP_MESSAGE_READ:
		message := &markGroupMessageRead{}
		err := json.Unmarshal(event[pipeIndex+1:], message)
		if err != nil {
			return err
		}

		notification := toast.Notification{
			AppID:   "TerTerChat",
			Title:   "group message read",
			Message: message.ID.String(),
		}
		if err = notification.Push(); err != nil {
			return err
		}
	case ADDED_USER_TO_GROUP:
		message := &groupEvent{}
		err := json.Unmarshal(event[pipeIndex+1:], message)
		if err != nil {
			return err
		}

		notification := toast.Notification{
			AppID:   "TerTerChat",
			Title:   "added user to group",
			Message: message.Group.username + message.Group.phonenumber,
		}
		if err = notification.Push(); err != nil {
			return err
		}
	case REMOVE_USER_FROM_GROUP:
		message := &groupEvent{}
		err := json.Unmarshal(event[pipeIndex+1:], message)
		if err != nil {
			return err
		}

		notification := toast.Notification{
			AppID:   "TerTerChat",
			Title:   "removed user from group" + message.Group.id.String(),
			Message: message.Group.username + message.Group.phonenumber,
		}
		if err = notification.Push(); err != nil {
			return err
		}
	case MADE_ADMIN:
		message := &groupEvent{}
		err := json.Unmarshal(event[pipeIndex+1:], message)
		if err != nil {
			return err
		}

		notification := toast.Notification{
			AppID:   "TerTerChat",
			Title:   "made user admin",
			Message: message.Group.username + message.Group.phonenumber,
		}
		if err = notification.Push(); err != nil {
			return err
		}
	case REMOVE_ADMIN:
		message := &groupEvent{}
		err := json.Unmarshal(event[pipeIndex+1:], message)
		if err != nil {
			return err
		}

		notification := toast.Notification{
			AppID:   "TerTerChat",
			Title:   "removed user from admin",
			Message: message.Group.username + message.Group.phonenumber,
		}
		if err = notification.Push(); err != nil {
			return err
		}
	default:
		return errors.New("invalid event")
	}

	return nil
}

func readFromConnection(connection net.Conn, writer chan<- []byte, wg *sync.WaitGroup) {
	fmt.Println("Starting to read from server")
	reader := bufio.NewReader(connection)
	defer func() {
		fmt.Println("Stopping to read from server")
		wg.Done()
		if _, ok := <-quit; ok {
			close(quit)
		}
	}()

	// reading from connection
	for {
		select {
		case <-quit:
			return
		default:
			connection.SetReadDeadline(time.Now().Add(pingTimeout * time.Second))
			message, err := reader.ReadBytes('\n')
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					fmt.Println("connnection timedout.")
					continue
				} else if err == io.EOF {
					fmt.Println("server closed the connection")
				} else {
					fmt.Println("error reading from server")
				}

				return
			}

			// start parsing the message to pass it to appropriate event handler
			// if the message is pong then pass it to writeToConnection
			msgString := strings.TrimSpace(string(message))
			switch msgString {
			case pingMessage[:len(pingMessage)-1]:
				writer <- []byte(pongMessage)
			case pongMessage[:len(pingMessage)-1]:
				writer <- []byte(pingMessage)
			default:
				// parse events
				if err = eventParser(message); err != nil {
					fmt.Printf("Error parsing the message: %v\n", err)
					continue
				}
			}
		}
	}
}

func writeToConnection(connection net.Conn, writer <-chan []byte, wg *sync.WaitGroup) {
	fmt.Println("Starting to write to server")
	ticker := time.NewTicker(pingInterval)
	defer func() {
		fmt.Println("Stopping write to server")
		ticker.Stop()
		wg.Done()
		if _, ok := <-quit; ok {
			close(quit)
		}
	}()

	for {
		select {
		case <-ticker.C:
			fmt.Println("writing ping message to server")
			connection.SetWriteDeadline(time.Now().Add(pingTimeout))
			if _, err := connection.Write([]byte(pingMessage)); err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					fmt.Println("connection timedout")
					continue
				}

				return
			}
		case message, ok := <-writer:
			fmt.Println("writing pong message to server")
			if !ok {
				fmt.Println("writer channel closed")
				return
			}

			connection.SetWriteDeadline(time.Now().Add(pingTimeout))
			if _, err := connection.Write(message); err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					fmt.Println("connection timedout")
					continue
				}

				return
			}
		case <-quit:
			return
		}
	}
}

// this function will initiate the tls tcp socket connection
func connect(phonenumber string, deamonWG *sync.WaitGroup) {
	defer deamonWG.Done()

	// loading rootCA and adding it to the trust store so that it can accept server's certificate
	rootCAs := x509.NewCertPool()
	caCert, err := os.ReadFile("certificates/ca.crt")
	if err != nil {
		fmt.Printf("Error connecting to server: %v", err)
		return
	}
	rootCAs.AppendCertsFromPEM(caCert)

	// loading client certificates and private key
	certificate, err := tls.LoadX509KeyPair(certificateFile, keyFile)
	if err != nil {
		fmt.Printf("Error connecting to server: %v", err)
		return
	}

	// configuring TLS for client
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256},
		RootCAs:      rootCAs,
		KeyLogWriter: os.Stdout,
	}

	// creating a dialer to connect to server and send the user phonenumber
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		fmt.Printf("Error connecting to server: %v", err)
		return
	}

	// sending phonenumber
	if _, err = conn.Write([]byte(phonenumber)); err != nil {
		fmt.Printf("Error connecting to server: %v", err)
		return
	}

	// creating a channel for communication between readFromConnection and writeToConnection
	writer := make(chan []byte, 10)

	// creating a channel to recieve OS signals to shutdown TCP connection
	// we'll catch Ctrl+C (SIGINT) and kill signals (SIGTERM)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-quit:
			fmt.Println("Closing connection to server")
			conn.Close()
			close(writer)
		case <-sigc:
			fmt.Println("Closing connection to server")
			conn.Close()
			close(writer)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)
	go readFromConnection(conn, writer, &wg)
	go writeToConnection(conn, writer, &wg)
	wg.Wait()
	fmt.Println("Connection to server was closed!")
}

func isConnectionAlive() string {
	if isConnected {
		return "connected\n"
	}

	return "disconnected\n"
}
