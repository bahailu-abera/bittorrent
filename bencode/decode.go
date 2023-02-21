package bencode

import (
	"bufio"
	"io"
)


func Decode(reader io.Reader) (data interface{}, err error) {
	// Check to see if the reader already fulfills the bufio.Reader interface.
	// Wrap it in a bufio.Reader if it doesn't.
	bufioReader, ok := reader.(*bufio.Reader)
	if !ok {
		bufioReader = newBufioReader(reader)
		defer bufioReaderPool.Put(bufioReader)
	}

	return decodeFromReader(bufioReader)
}
