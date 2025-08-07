package internal

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

const (
	socketFileName = "cli.sock"
	socketType     = "unix"
)

var (
	isDeamonRunning = false               // variable to track status of deamon process whether it is running or not
	shutdownChannel = make(chan struct{}) // channel use to shutdown deamon process when disconnect command is executed
)

func getSocketPath() string {
	return filepath.Join(os.TempDir(), socketFileName)
}

// main entry point for deamon process
func StartDeamon(phonenumber string) error {
	// this is a very crucial step for unix sockets
	// it remove any old socket file that might exist
	// this prevents the "address already in use" error if the daemon previously
	// crashed without properly cleaning up
	socketPath := getSocketPath()
	if err := os.RemoveAll(socketPath); err != nil {
		return err
	}

	// creating a unix listener. This will also create the socket file
	// this socket file will be used for IPC between this process and other commands
	// who need to communicate with this process
	listener, err := net.Listen(socketType, socketPath)
	if err != nil {
		return err
	}
	isDeamonRunning = true
	defer func() {
		log.Println("Closing unix socket listener")
		listener.Close()

		// cleaning up the socket file on gracefull shutdown
		if err = os.Remove(socketPath); err != nil {
			log.Printf("Error removing socket file: %v", err)
		}
	}()

	log.Println("Deamon process started. Listening on: ", socketPath)

	// creating waitgroup for signal handler and connect goroutine
	var wg sync.WaitGroup
	wg.Add(2)

	// launching a goroutine to check for OS signals on sigc channel
	// and for disconnect command to signal on shutdown channel
	go func() {
		defer func() {
			isDeamonRunning = false
			wg.Done()
		}()
		// creating a channel to shutdown deamon process from OS signals
		// we'll catch Ctrl+C (SIGINT) and kill signals (SIGTERM)
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-sigc:
			log.Println("Received OS shutdown signal.")
		case <-shutdownChannel:
			log.Println("Received internal shutdown command")
		}

		// emitting signal to tcp connection to shutdown
		close(quit)
	}()

	// starting the TCP socket connection to server
	go connect(phonenumber, &wg)

	// main loop which will continue to accept connections from other processes or commands
	// until any OS signal like SIGINT/SIGTERM is emitted or disconnect command is executed
	for {
		conn, err := listener.Accept()
		if err != nil {
			// if the error is due to listener being closed, we will exit gracefully
			if errors.Is(err, net.ErrClosed) {
				fmt.Printf("Listener closed, exiting.")
				break
			}

			// for any other errors log, emit shutdown signal and break
			log.Printf("Unexpected error accepting connections: %v. Signaling shutdown", err)
			shutdownChannel <- struct{}{}
			break
		}

		// launching a connection handler goroutine for every new connection
		go handleConnection(conn)
	}

	// waiting for all the goroutines launched from this Deamon Process to finish
	wg.Wait()
	log.Println("Deamon process has fully shutdown")
	return nil
}

func handleConnection(connection net.Conn) {
	reader := bufio.NewReader(connection)
	command, err := reader.ReadBytes('\n')
	if err != nil {
		fmt.Printf("Error reading from process: %v", err)
		return
	}

	switch strings.TrimSpace(string(command)) {
	case "status":
		connection.Write([]byte(isConnectionAlive()))
	case "disconnect":
		shutdownChannel <- struct{}{}
	}
}
