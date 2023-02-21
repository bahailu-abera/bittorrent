package torrent

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"os"
	"log"
	"time"
	"sync"

	"bittorrent/bencode"
	"bittorrent/peer2peer"
	"bittorrent/client"
	"bittorrent/peers"
)

// Port to listen on
const Port uint16 = 6881

// TorrentFile encodes the metadata from a .torrent file
type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

type bencodeInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

func (t *TorrentFile) GetTorrent() (peer2peer.Torrent, error) {
	var peerID [20]byte
	_, err := rand.Read(peerID[:])
	if err != nil {
		return peer2peer.Torrent{}, err
	}

	peers, err := t.requestPeers(peerID, Port)
	if err != nil {
		return peer2peer.Torrent{}, err
	}

	torrent := peer2peer.Torrent{
		Peers:       peers,
		PeerID:      peerID,
		InfoHash:    t.InfoHash,
		PieceHashes: t.PieceHashes,
		PieceLength: t.PieceLength,
		Length:      t.Length,
		Name:        t.Name,
	}

	return torrent, nil
}

// // Connect to peers
// func ConnectToPeers(torrent peer2peer.Torrent,
// 	keepAliveChan chan bool) ([]*client.Client, error) {

// 	var clients []*client.Client
// 	for _, peer := range torrent.Peers {
// 		c, err := client.New(peer, torrent.PeerID, torrent.InfoHash)
// 		if err != nil {
// 			log.Printf("Could not handshake with %s. Disconnecting\n", peer.IP)
// 			continue
// 		}
// 		log.Printf("Completed handshake with %s\n", peer.IP)
// 		clients = append(clients, c)
// 	}
// 	// Start a goroutine that sends KeepAlive messages to the peer
//         go func() {
//             for {
//                 select {
//                 case <-time.After(30 * time.Second):
//                     keepAliveChan <- true
//                 }
//             }
//         }()

// 	if len(clients) == 0 {
// 		return nil, fmt.Errorf("failed to connect to any peers")
// 	}

// 	return clients, nil
// }

// Connect to peers concurrently
func ConnectToPeers(torrent peer2peer.Torrent,
	keepAliveChan chan bool) ([]*client.Client, error) {

	var clients []*client.Client
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, peer := range torrent.Peers {
		wg.Add(1)
		go func(p peers.Peer) {
			defer wg.Done()
			c, err := client.New(p, torrent.PeerID, torrent.InfoHash)
			if err != nil {
				log.Printf("Could not handshake with %s. Disconnecting\n", p.IP)
				return
			}
			log.Printf("Completed handshake with %s\n", p.IP)
			mu.Lock()
			clients = append(clients, c)
			mu.Unlock()
		}(peer)
	}

	// Start a goroutine that sends KeepAlive messages to the peer
	go func() {
		for {
			select {
			case <-time.After(30 * time.Second):
				keepAliveChan <- true
			}
		}
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	if len(clients) == 0 {
		return nil, fmt.Errorf("failed to connect to any peers")
	}
	fmt.Println("handshake completed")
	return clients, nil
}

func (t *TorrentFile) DownloadToFile(path string,
	torrent peer2peer.Torrent, clients []*client.Client) error {
	buf, err := torrent.Download(clients)
	if err != nil {
		return err
	}

	outFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = outFile.Write(buf)
	if err != nil {
		return err
	}

	fmt.Println("------------------------Download completed-----------------------------------------")
	return nil
}

// Open parses a torrent file
func Open(path string) (TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return TorrentFile{}, err
	}
	defer file.Close()

	bto := bencodeTorrent{}
	err = bencode.Unmarshal(file, &bto)
	if err != nil {
		return TorrentFile{}, err
	}
	return bto.toTorrentFile()
}

func (i *bencodeInfo) hash() ([20]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, *i)
	if err != nil {
		return [20]byte{}, err
	}
	h := sha1.Sum(buf.Bytes())
	return h, nil
}

func (i *bencodeInfo) splitPieceHashes() ([][20]byte, error) {
	hashLen := 20 // Length of SHA-1 hash
	buf := []byte(i.Pieces)
	if len(buf)%hashLen != 0 {
		err := fmt.Errorf("Received malformed pieces of length %d", len(buf))
		return nil, err
	}
	numHashes := len(buf) / hashLen
	hashes := make([][20]byte, numHashes)

	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*hashLen:(i+1)*hashLen])
	}
	return hashes, nil
}

func (bto *bencodeTorrent) toTorrentFile() (TorrentFile, error) {
	infoHash, err := bto.Info.hash()
	if err != nil {
		return TorrentFile{}, err
	}
	pieceHashes, err := bto.Info.splitPieceHashes()
	if err != nil {
		return TorrentFile{}, err
	}
	t := TorrentFile{
		Announce:    bto.Announce,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: bto.Info.PieceLength,
		Length:      bto.Info.Length,
		Name:        bto.Info.Name,
	}
	return t, nil
}
