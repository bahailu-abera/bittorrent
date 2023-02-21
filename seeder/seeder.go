package seeder

import (
	"log"
	"os"
	"fmt"
	"sync"

	"bittorrent/peer2peer"
	"bittorrent/client"
	"bittorrent/bitfield"
	"bittorrent/message"
)

func handleRequestError(torrent peer2peer.Torrent, index, begin, length int) error {
	numPieces := len(torrent.PieceHashes)
	pieceLength := torrent.PieceLength
	fileSize := torrent.Length

	if index < 0 || index >= numPieces {
		return fmt.Errorf("Invalid piece index %d", index)
	}

	// Check if this is the last piece
        lastPiece := index == numPieces-1

        // Calculate the length of this piece
        var len int
        if lastPiece {
		len = int(pieceLength * numPieces - fileSize)
        } else {
		len = pieceLength
        }
	// Check if the requested block offset and length are valid
	if begin < 0 || begin+length > len || length <= 0 {
		return fmt.Errorf("Invalid block offset %d or length %d", begin, length)
	}

	return nil
}

func SeedFile(clients []*client.Client, torrent peer2peer.Torrent,
	path string) {
	fmt.Println("I have called I am the seeder")
	// Open the file for reading
	file, err := os.Open(path)
	if err != nil {
		log.Printf("Failed to open file: %v", err)
		return
	}
	defer file.Close()

	// Create a bitfield indicating that all pieces are available
	numPieces := len(torrent.PieceHashes)
	bitField := make(bitfield.Bitfield, (numPieces+7)/8)
	for i := 0; i < numPieces; i++ {
		bitField.SetPiece(i)
	}

	// Use a WaitGroup to wait for all clients to finish serving
	var wg sync.WaitGroup
	wg.Add(len(clients))

	// Serve each client
	for _, c := range clients {
		c.SendUnchoke()
		c.SendNotInterested()

		for i := 0; i < numPieces; i++ {
			if bitField.HasPiece(i) {
				c.SendHave(i)
			}
		}

		// Start a goroutine to serve the client
		go serveClient(&wg, c, torrent, file)
	}

	// Wait for all clients to finish serving
	wg.Wait()
}

func serveClient(wg *sync.WaitGroup, c *client.Client, torrent peer2peer.Torrent, file *os.File) {
	defer func() {
		c.Conn.Close()
		wg.Done() // Signal that this client has finished serving
	}()

	for {
		fmt.Println("Wait for  a message from the client")
		// Wait for a message from the client
		msg, err := c.Read()
		if err != nil {
			log.Printf("Error reading from client: %v", err)
			//continue
			return
		}
		if msg == nil {
			// Keep-alive message
			continue
		}

		switch msg.ID {
		case message.MsgRequest:
			// Parse the request message
			index, begin, length, err := message.ParseRequest(msg)
			if err != nil {
				log.Printf("Error parsing request message: %v", err)
				continue
			}

			// Check if the requested block is valid
			err = handleRequestError(torrent, index, begin, length)
			if err != nil {
				log.Printf("Error handling request: %v", err)
				continue
			}

			// Get the requested data from the file reader
			// offset := int64(index)*int64(torrent.PieceLength) + int64(begin)
			data, err := getData(file, torrent, index, begin, length)
			if err != nil {
				log.Printf("Error getting data from file: %v", err)
				continue
			}
			// Send the data to the client
			err = c.SendPiece(index, begin, data)
			if err != nil {
				log.Printf("Error sending data to client: %v", err)
			}
			fmt.Println("Sending piece %d for peer %s",
				index, c.Peer.String())
		}
	}
}

func getData(file *os.File, torrent peer2peer.Torrent, index, begin, length int) ([]byte, error) {
	numPieces := len(torrent.PieceHashes)
	pieceLength := torrent.PieceLength
	fileSize := torrent.Length

	offset := int64(index) * int64(pieceLength) + int64(begin)

        // Check if this is the last piece
        lastPiece := index == numPieces-1

        // Calculate the length of this piece
        var len int
        if lastPiece {
		len = int(fileSize - int(offset))
        } else {
		len = torrent.PieceLength
        }

	buf := make([]byte, len)

        // Read the data from the file
        _, err := file.ReadAt(buf[:len], offset)
        if err != nil {
		log.Printf("Error reading from file: %v", err)
		return nil, err
        }
	return buf, nil
}
