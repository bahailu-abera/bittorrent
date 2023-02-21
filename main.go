package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"bittorrent/torrent"
	"bittorrent/seeder"
)

func main() {
	inPath := os.Args[1]
	outPath := os.Args[2]

	// Open the dot torrent file
	tf, err := torrent.Open(inPath)
	if err != nil {
		log.Fatal(err)
	}

	tor, err := tf.GetTorrent()
	if err != nil {
		log.Fatal(err)
	}

	// Connect to peers and download file
	keepAliveChan := make(chan bool)
	clients, err := torrent.ConnectToPeers(tor, keepAliveChan)
	fmt.Printf("Number of clients is %d\n", len(clients))
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		for {
			select {
			case <-keepAliveChan:
				for _, c := range clients {
					c.SendKeepAlive()
				}
			}
		}
	}()

	// Download file and start seeding
	err = tf.DownloadToFile(outPath, tor, clients)
	if err != nil {
		log.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		seeder.SeedFile(clients, tor, outPath)
	}()

	wg.Wait()
	fmt.Println("Main function completed")
}
