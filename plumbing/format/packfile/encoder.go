package packfile

import (
	"crypto/sha1"
	"fmt"
	"hash"
	"io"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/utils/binary"

	"github.com/klauspost/compress/zlib"
)

// Encoder gets the data from the storage and write it into the writer in PACK
// format
type Encoder struct {
	storage storer.ObjectStorer
	w       io.Writer
	hash    hash.Hash
}

// NewEncoder creates a new packfile encoder using a specific Writer and
// ObjectStorer
func NewEncoder(w io.Writer, s storer.ObjectStorer) *Encoder {
	h := sha1.New()
	mw := io.MultiWriter(w, h)
	return &Encoder{
		storage: s,
		w:       mw,
		hash:    h,
	}
}

// Encode encodes objects specified using a list of hashes. This objects must
// exists into the storer
func (e *Encoder) Encode(hashes []plumbing.Hash) (plumbing.Hash, error) {
	if err := e.head(len(hashes)); err != nil {
		return plumbing.ZeroHash, err
	}

	for _, h := range hashes {
		o, err := e.storage.Object(plumbing.AnyObject, h)
		if err != nil {
			return plumbing.ZeroHash, err
		}

		if err := e.entry(o); err != nil {
			return plumbing.ZeroHash, err
		}
	}

	return e.footer()
}

func (e *Encoder) head(numEntries int) error {
	return binary.Write(
		e.w,
		signature,
		int32(VersionSupported),
		int32(numEntries),
	)
}

func (e *Encoder) entry(o plumbing.Object) error {
	t := o.Type()
	if t == plumbing.OFSDeltaObject || t == plumbing.REFDeltaObject {
		// TODO implements delta objects
		return fmt.Errorf("delta object not supported: %v", t)
	}

	if err := e.entryHead(t, o.Size()); err != nil {
		return err
	}

	zw := zlib.NewWriter(e.w)
	or, err := o.Reader()
	if err != nil {
		return err
	}
	_, err = io.Copy(zw, or)
	if err != nil {
		return err
	}

	return zw.Close()
}

func (e *Encoder) entryHead(typeNum plumbing.ObjectType, size int64) error {
	t := int64(typeNum)
	header := []byte{}
	c := (t << firstLengthBits) | (size & maskFirstLength)
	size >>= firstLengthBits
	for {
		if size == 0 {
			break
		}
		header = append(header, byte(c|maskContinue))
		c = size & int64(maskLength)
		size >>= lengthBits
	}

	header = append(header, byte(c))
	_, err := e.w.Write(header)

	return err
}

func (e *Encoder) footer() (plumbing.Hash, error) {
	b := e.hash.Sum(nil)
	var h plumbing.Hash
	copy(h[:], b)
	return h, binary.Write(e.w, h)
}
