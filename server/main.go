package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type client struct {
	ch chan<- string
	ID string
}

var (
	enter   = make(chan client)
	leave   = make(chan client)
	message = make(chan string)
	token   = make(chan struct{}, 16)
)

func main() {
	listener, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}
	go broadcast()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
			continue
		}
		go handler(conn)
	}
}

func broadcast() {
	clients := make(map[client]bool)
	for {
		select {
		case msg := <-message:
			for cli := range clients {
				cli.ch <- msg
			}
		case cli := <-enter:
			clients[cli] = true
			var online []string
			for c := range clients {
				online = append(online, c.ID)
			}
			cli.ch <- fmt.Sprintf("Online: %v", online)
		case cli := <-leave:
			delete(clients, cli)
			close(cli.ch)
		}
	}
}

func handler(conn net.Conn) {
	token <- struct{}{}
	ch := make(chan string)
	var once sync.Once
	var clean = func() {
		conn.Close()
	}
	go func() {
		for msg := range ch {
			fmt.Fprintln(conn, msg)
		}
	}()
	fmt.Fprintln(conn, "Enter your ID: ")
	input := bufio.NewScanner(conn)
	input.Scan()
	ID := input.Text()
	message <- ID + " has arrived"
	enter <- client{ch, ID}
	alive := make(chan struct{})
	go func() {
		for {
			select {
			case <-time.After(5 * time.Minute):
				fmt.Fprintln(conn, "Timeout, link closed")
				once.Do(clean)
				return
			case <-alive:
			}
		}
	}()
	for input.Scan() {
		message <- ID + ": " + input.Text()
		alive <- struct{}{}
	}
	leave <- client{ch, ID}
	message <- ID + " has left"
	once.Do(clean)
	<-token
}
