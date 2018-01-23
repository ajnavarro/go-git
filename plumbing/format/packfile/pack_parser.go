package packfile

import (
	"errors"
	"fmt"
	"io"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/cache"
)

type PackParser struct {
	scanner     *Scanner
	oiByHash    map[plumbing.Hash]*ObjectInfo
	oiByOffset  map[int64]*ObjectInfo
	objectInfos []*ObjectInfo

	cache cache.Object
}

func NewPackParser(r io.Reader) *PackParser {
	return &PackParser{scanner: NewScanner(r), cache: cache.NewObjectLRUDefault()}
}

func (pp *PackParser) Close() (plumbing.Hash, error) {
	h, err := pp.scanner.Checksum()
	if err != nil {
		return plumbing.ZeroHash, err
	}

	pp.oiByHash = nil
	pp.oiByOffset = nil
	pp.objectInfos = nil

	return h, nil
}

func (pp *PackParser) Parse() error {
	count, err := pp.init()
	if err != nil {
		return err
	}

	for i := 0; i < count; i++ {
		oh, err := pp.scanner.NextObjectHeader()
		if err != nil {
			return err
		}

		var ota *ObjectInfo
		switch t := oh.Type; t {
		case plumbing.OFSDeltaObject, plumbing.REFDeltaObject:
			var next *ObjectInfo
			var ok bool
			if t == plumbing.OFSDeltaObject {
				next, ok = pp.oiByOffset[oh.OffsetReference]
			} else {
				next, ok = pp.oiByHash[oh.Reference]
			}

			if !ok {
				// TODO improve error
				return errors.New("Reference delta not found")
			}

			ota = newDeltaObject(oh.Offset, oh.Length, t, next)
		default:
			ota = newBaseObject(oh.Offset, oh.Length, t)
		}

		_, crc, err := pp.scanner.NextObject(ota)
		if err != nil {
			return err
		}

		ota.Crc32 = crc

		pp.oiByOffset[oh.Offset] = ota
		pp.oiByHash[oh.Reference] = ota

		pp.objectInfos = append(pp.objectInfos, ota)
	}

	//TODO
	return pp.resolveDeltas()
}

func (pp *PackParser) resolveDeltas() error {
	for _, oi := range pp.objectInfos {
		delete(pp.oiByHash, oi.Sum())
		delete(pp.oiByOffset, oi.Offset)

		if oi.Parent == nil {
			// not a delta
			continue
		}

		rd := pp.addChilds(oi)

		visit := newDeltaVisitBase(rd)

		switch rd.Type {
		case plumbing.CommitObject,
			plumbing.TreeObject,
			plumbing.BlobObject,
			plumbing.TagObject:
			// TODO add data of this object to the visitor
			//visit.data =
			visit.id = oi.Sum()
		default:
			return fmt.Errorf("bad object type: %s", oi.Type)
		}

		// TODO check CRC

		if err := pp.resolveDeltaChain(visit); err != nil {
			return err
		}
	}

	return nil
}

func (pp *PackParser) resolveDeltaChain(visit *deltaVisit) error {
	return nil
}

func (pp *PackParser) addChilds(c *ObjectInfo) *ObjectInfo {
	var nextChild *ObjectInfo
	for c.Parent != nil {
		c.Child = nextChild
		nextChild, c = c, c.Parent
	}

	return c
}

func (pp *PackParser) init() (int, error) {
	_, c, err := pp.scanner.Header()
	if err != nil {
		return 0, err
	}

	count := int(c)
	pp.oiByHash = make(map[plumbing.Hash]*ObjectInfo, count)
	pp.oiByOffset = make(map[int64]*ObjectInfo, count)

	return count, nil
}

type deltaVisit struct {
	delta     *ObjectInfo
	id        plumbing.Hash
	data      []byte
	parent    *deltaVisit
	nextChild *ObjectInfo
}

func newDeltaVisitBase(nc *ObjectInfo) *deltaVisit {
	return &deltaVisit{nextChild: nc}
}

func newDeltaVisit(parent *deltaVisit) *deltaVisit {
	dv := &deltaVisit{
		parent: parent,
		delta:  parent.nextChild,
	}

	parent.nextChild = dv.delta.Parent

	return dv
}

func (dv *deltaVisit) Next() *deltaVisit {
	// If our parent has no more children, discard it.
	if dv.parent != nil && dv.parent.nextChild == nil {
		dv.parent.data = nil
		dv.parent = dv.parent.parent
	}

	if dv.nextChild != nil {
		return newDeltaVisit(dv)
	}

	// If we have no child ourselves, our parent must (if it exists),
	// due to the discard rule above. With no parent, we are done.
	if dv.parent != nil {
		return newDeltaVisit(dv.parent)
	}

	return nil
}

type ObjectInfo struct {
	plumbing.Hasher

	Offset int64
	Length int64
	Type   plumbing.ObjectType

	Crc32 uint32

	Parent *ObjectInfo
	Child  *ObjectInfo
}

func newBaseObject(offset, length int64, t plumbing.ObjectType) *ObjectInfo {
	return &ObjectInfo{plumbing.NewHasher(t, length), offset, length, t, 0, nil, nil}
}

func newDeltaObject(offset, length int64, t plumbing.ObjectType, parent *ObjectInfo) *ObjectInfo {
	return &ObjectInfo{plumbing.NewHasher(t, length), offset, length, t, 0, parent, nil}
}
