package tarutil

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
)

// blockSize is the size of each block in a tar archive.
const blockSize int64 = 512

// zeroBlock is a block of all zeros.
var zeroBlock block

// block is a block in a tar archive.
type block [blockSize]byte

// blockPadding computes the number of bytes needed to pad offset up to the
// nearest block edge where 0 <= n < blockSize.
func blockPadding(offset int64) (n int64) {
	return -offset & (blockSize - 1)
}

// EntryWriter writes tar archive entry to a io.WriteSeeker without knowing the
// length of payload.
type EntryWriter struct {
	base io.WriteSeeker
	size int64
	pos  int64
}

// NewEntryWriter creates a new EntryWriter.
func NewEntryWriter(ws io.WriteSeeker) (*EntryWriter, error) {
	// skip header
	pos, err := ws.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}
	if _, err := ws.Seek(blockSize, io.SeekCurrent); err != nil {
		return nil, err
	}
	return &EntryWriter{
		base: ws,
		pos:  pos,
	}, nil
}

// Write writes p to the underlying io.Writer.
func (ew *EntryWriter) Write(p []byte) (int, error) {
	n, err := ew.base.Write(p)
	ew.size += int64(n)
	return n, err
}

// Commit writes the header and padding to the underlying io.WriteSeeker.
func (ew *EntryWriter) Commit(name string) error {
	// update header
	header := &tar.Header{
		Name: name,
		Size: ew.size,
		Mode: 0644,
	}
	headerBuf := bytes.NewBuffer(nil)
	tw := tar.NewWriter(headerBuf)
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	headerBytes := headerBuf.Bytes()
	if len(headerBytes) != int(blockSize) {
		return errors.New("invalid header size")
	}
	if _, err := ew.base.Seek(ew.pos, io.SeekStart); err != nil {
		return err
	}
	if _, err := ew.base.Write(headerBytes); err != nil {
		return err
	}

	// write padding
	offset := ew.pos + blockSize + ew.size
	if _, err := ew.base.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	if paddingSize := blockPadding(ew.size); paddingSize > 0 {
		if _, err := ew.base.Write(zeroBlock[:paddingSize]); err != nil {
			return err
		}
	}

	return nil
}

// Size returns the size of the payload.
func (ew *EntryWriter) Size() int64 {
	return ew.size
}

// WriteFile writes a file to a tar archive as a single entry.
func WriteFile(w io.Writer, name string, data []byte) error {
	header := &tar.Header{
		Name: name,
		Size: int64(len(data)),
		Mode: 0644,
	}
	tw := tar.NewWriter(w)
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	if _, err := tw.Write(data); err != nil {
		return err
	}
	return tw.Flush()
}

// Copy copies content to a tar archive as a single entry.
func Copy(w io.Writer, name string, size int64, r io.Reader) error {
	header := &tar.Header{
		Name: name,
		Size: size,
		Mode: 0644,
	}
	tw := tar.NewWriter(w)
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	if _, err := io.Copy(tw, r); err != nil {
		return err
	}
	return tw.Flush()
}

// Close closes a tar file by writing an EOF mark.
func Close(w io.Writer) error {
	tw := tar.NewWriter(w)
	return tw.Close()
}
