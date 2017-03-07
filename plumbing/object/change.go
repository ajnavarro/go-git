package object

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"srcd.works/go-git.v4/plumbing"
	"srcd.works/go-git.v4/utils/merkletrie"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// Change values represent a detected change between two git trees.  For
// modifications, From is the original status of the node and To is its
// final status.  For insertions, From is the zero value and for
// deletions To is the zero value.
type Change struct {
	From ChangeEntry
	To   ChangeEntry
}

var empty = ChangeEntry{}

// Action returns the kind of action represented by the change, an
// insertion, a deletion or a modification.
func (c *Change) Action() (merkletrie.Action, error) {
	if c.From == empty && c.To == empty {
		return merkletrie.Action(0),
			fmt.Errorf("malformed change: empty from and to")
	}
	if c.From == empty {
		return merkletrie.Insert, nil
	}
	if c.To == empty {
		return merkletrie.Delete, nil
	}

	return merkletrie.Modify, nil
}

// Files return the files before and after a change.
// For insertions from will be nil. For deletions to will be nil.
func (c *Change) Files() (from, to *File, err error) {
	action, err := c.Action()
	if err != nil {
		return
	}

	if action == merkletrie.Insert || action == merkletrie.Modify {
		to, err = c.To.Tree.TreeEntryFile(&c.To.TreeEntry)
		if err != nil {
			return
		}
	}

	if action == merkletrie.Delete || action == merkletrie.Modify {
		from, err = c.From.Tree.TreeEntryFile(&c.From.TreeEntry)
		if err != nil {
			return
		}
	}

	return
}

// WritePatchTo writes the generated Patch of this change to the specified Writer
func (c *Change) WritePatchTo(w io.Writer) error {
	from, to, err := c.Files()
	if err != nil {
		return err
	}

	buf := bufio.NewWriter(w)

	if err := c.diffHeader(buf); err != nil {
		return err
	}

	isBinary, err := c.printIfBinaryPatch(buf, from, to)
	if isBinary || err != nil {
		return err
	}

	if err := extendedHeaders(buf, from, to); err != nil {
		return err
	}

	buf.WriteString(fmt.Sprintf("--- %s\n", c.filePath("a", from)))
	buf.WriteString(fmt.Sprintf("+++ %s\n", c.filePath("b", to)))

	fc, err := content(from)
	if err != nil {
		return err
	}

	tc, err := content(to)
	if err != nil {
		return err
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(fc, tc, false)

	diffBuf := bytes.NewBuffer(nil)
	lenDiffs := len(diffs)
	for iDiff, diff := range diffs {
		//buf.WriteString(fmt.Sprintf("@@ -%s +%s @@\n", range1, range2))
		lines := strings.Split(diff.Text, "\n")
		lenLines := len(lines)
		for iLine, line := range lines {
			if lenDiffs == iDiff+1 && lenLines == iLine+1 && line == "" {
				if line == "" {
					break
				} else {
					// TODO no new line at the end of the file
				}
			}
			switch diff.Type {
			case diffmatchpatch.DiffInsert:
				diffBuf.WriteString(fmt.Sprintf("+%s\n", line))
			case diffmatchpatch.DiffDelete:
				diffBuf.WriteString(fmt.Sprintf("-%s\n", line))
			case diffmatchpatch.DiffEqual:
				diffBuf.WriteString(fmt.Sprintf(" %s\n", line))
			}
		}
	}

	// TODO print @@

	_, err = buf.ReadFrom(diffBuf)

	return err
}


func (c Change) Patch() (string, error) {
	buf := bytes.NewBuffer(nil)

	if err := c.WritePatchTo(buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (c *Change) String() string {
	action, err := c.Action()
	if err != nil {
		return fmt.Sprintf("malformed change")
	}

	return fmt.Sprintf("<Action: %s, Path: %s>", action, c.name())
}

func (c *Change) name() string {
	if c.From != empty {
		return c.From.Name
	}

	return c.To.Name
}

// ChangeEntry values represent a node that has suffered a change.
type ChangeEntry struct {
	// Full path of the node using "/" as separator.
	Name string
	// Parent tree of the node that has changed.
	Tree *Tree
	// The entry of the node.
	TreeEntry TreeEntry
}

// Changes represents a collection of changes between two git trees.
// Implements sort.Interface lexicographically over the path of the
// changed files.
type Changes []*Change

func (c Changes) Len() int {
	return len(c)
}

func (c Changes) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c Changes) Less(i, j int) bool {
	return strings.Compare(c[i].name(), c[j].name()) < 0
}

func (c Changes) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("[")
	comma := ""
	for _, v := range c {
		buffer.WriteString(comma)
		buffer.WriteString(v.String())
		comma = ", "
	}
	buffer.WriteString("]")

	return buffer.String()
}

// WritePatchTo writes the generated Patch of this changes to the specified Writer
func (c Changes) WritePatchTo(w io.Writer) error {
	buf := bufio.NewWriter(w)
	for _, change := range c {
		if err := change.WritePatchTo(buf); err != nil {
			return err
		}
	}

	return nil
}

// Patch returns a string that represent all the patches of this groups of changes
func (c Changes) Patch() (string, error) {
	buf := bytes.NewBuffer(nil)
	if err := c.WritePatchTo(buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}
