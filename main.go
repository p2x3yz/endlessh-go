package main

//  echo -n "test out the server" | nc localhost 3333

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/golang/glog"
)

var (
	numCurrentClients int64
	numTotalClients   uint64
	numTotalBytes     uint64
)

func main() {
	intervalMs := flag.Int("interval_ms", 1000, "Message millisecond delay")
	bannerMaxLength := flag.Int64("line_length", 32, "Maximum banner line length")
	maxClients := flag.Int64("max_clients", 4096, "Maximum number of clients")
	connType := flag.String("conn_type", "tcp", "Connection type. Possible values are tcp, tcp4, tcp6")
	connHost := flag.String("host", "0.0.0.0", "Listening address")
	connPort := flag.String("port", "2222", "Listening port")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %v \n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	rand.Seed(time.Now().UnixNano())
	interval := time.Duration(*intervalMs) * time.Millisecond
	// Listen for incoming connections.
	if *connType == "tcp6" && *connHost == "0.0.0.0" {
		*connHost = "[::]"
	}
	l, err := net.Listen(*connType, *connHost+":"+*connPort)
	if err != nil {
		glog.Errorf("Error listening: %v", err)
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	glog.Infof("Listening on %v:%v", *connHost, *connPort)

	clients := make(chan *client, *maxClients)
	go func(clients chan *client, interval time.Duration, bannerMaxLength int64) {
		for {
			c, more := <-clients
			if !more {
				return
			}
			if time.Now().Before(c.next) {
				time.Sleep(c.next.Sub(time.Now()))
			}
			err := c.Send(bannerMaxLength)
			if err != nil {
				c.Close()
				continue
			}
			c.next = time.Now().Add(interval)
			go func() { clients <- c }()
		}
	}(clients, interval, *bannerMaxLength)
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			glog.Errorf("Error accepting: %v", err)
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		for numCurrentClients >= *maxClients {
			time.Sleep(interval)
		}
		clients <- NewClient(conn, interval, *maxClients)
	}
}
