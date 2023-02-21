package handshake

import (
	"fmt"
	"io"
)

// A Handshake is a special message that a peer uses to identify itself
type HandShake struct {
	Pstr     string
	InfoHash [20]byte
	PeerID   [20]byte
}

// New Creates a new handshake with the standard pstr
func New(infoHash, peerID [20]byte) *HandShake {
	return &HandShake{
		Pstr:      "BitTorrent protocol",
		InfoHash:  infoHash,
		PeerID:    peerID,
	}
}

func (h *HandShake) Serialize() []byte {
	buf := make([]byte, len(h.Pstr)+49)
	buf[0] = byte(len(h.Pstr))
	curr := 1
	curr += copy(buf[curr:], h.Pstr)
	curr += copy(buf[curr:], make([]byte, 8)) // 8 reserved bytes
	curr += copy(buf[curr:], h.InfoHash[:])
	curr += copy(buf[curr:], h.PeerID[:])
	return buf
}

func Read(r io.Reader) (*HandShake, error) {
	lengthBuf := make([]byte, 1)
	_,err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, err
	}
	pstrlen := int(lengthBuf[0])
	if pstrlen == 0 {
		err := fmt.Errorf("pstrlen cannot be 0")
		return nil, err
	}
	handshakeBuf := make([]byte, 48+pstrlen)
	_, err = io.ReadFull(r, handshakeBuf)

	var infoHash, peerID [20]byte

	copy(infoHash[:], handshakeBuf[pstrlen+8:pstrlen+8+20])
	copy(peerID[:], handshakeBuf[pstrlen+8+20:])

	h := HandShake{
		Pstr:     string(handshakeBuf[0:pstrlen]),
		InfoHash: infoHash,
		PeerID:   peerID,
	}

	return &h, nil
}
