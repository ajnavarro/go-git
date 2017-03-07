package object

import (
	"bufio"
	"fmt"
	"github.com/gogo/protobuf/io"
	"github.com/sergi/go-diff/diffmatchpatch"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"path/filepath"
	"strings"
)

const (
	eol       = "\n"
	noNewLine = "\\ No newline at end of file"

	context = 3
)

type hunk struct {
	addStart int
	addCount int
	remStart int
	remCount int
	lines    []string
}

type changeDiff struct {
	c Change
}

// WritePatchTo writes the generated Patch of this change to the specified Writer
func (c *changeDiff) WritePatchTo(w io.Writer) error {
	from, to, err := c.c.Files()
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

	diffsCount := len(diffs)
	//totalLinesCount := 0
	hunk := *hunk{}
	var hunks = []*hunk{}
	for diffIndex, diff := range diffs {
		lines := strings.Split(diff.Text, eol)
		linesCount := len(lines)

		switch diff.Type {
		case diffmatchpatch.DiffEqual:
			if linesCount <= context {
				hunk.lines = append(hunk.lines, diffLines(" ", lines))
			} else {
				// new hunk
				if diffIndex == 0 || diffIndex != diffsCount-1 {
					hunk = *hunk{}
					hunk.lines = append(hunk.lines, diffLines(" ", lines[:context]))
				}
				// end hunk
				if diffIndex == diffsCount-1 || diffIndex != 0 {
					hunk.lines = append(hunk.lines, diffLines(" ", lines[context:]))
					hunks = append(hunks, hunk)
				}
			}

		case diffmatchpatch.DiffInsert:
		case diffmatchpatch.DiffDelete:

		}

		// TODO if type is equals
		// if is first one get last n context lines
		// if is in the middle get first n lines and last n lines if n <= total diff lines get all
		// if is in the middle and is bigger than n lines create a hunk with previous lines

		// if is the last one get only last n context lines

		// TODO if type is insert
		// add and print only one

	}

	//
	//lenDiffs := len(diffs)
	//for iDiff, diff := range diffs {
	//	//buf.WriteString(fmt.Sprintf("@@ -%s +%s @@\n", range1, range2))
	//	lines := strings.Split(diff.Text, eol)
	//	lenLines := len(lines)
	//	for iLine, l := range lines {
	//		var diffl string
	//		if lenDiffs == iDiff+1 && lenLines == iLine+1 && l == "" {
	//			if l == "" {
	//				break
	//			} else {
	//				diffl = noNewLine
	//			}
	//		} else {
	//			switch diff.Type {
	//			case diffmatchpatch.DiffInsert:
	//				diffl = fmt.Sprintln("+", l)
	//			case diffmatchpatch.DiffDelete:
	//				diffl = fmt.Sprintln("-", l)
	//			case diffmatchpatch.DiffEqual:
	//				// TODO save last context lines
	//				//diffl = fmt.Sprintln(" ", l)
	//			}
	//		}
	//
	//		if _, err := diffBuf.WriteString(diffl); err != nil {
	//			return err
	//		}
	//	}
	//}
	//
	//// TODO print @@
	//
	//_, err = buf.ReadFrom(diffBuf)

	return err
}

func diffLines(prefix string, lines []string) []string {
	var result []string
	for _, l := range lines {
		result = append(result, fmt.Sprintln(prefix, l))
	}

	return result
}

// TODO
func formatRangeUnified(start, stop int) string {
	beginning := start + 1 // lines start numbering with one
	length := stop - start
	if length == 1 {
		return fmt.Sprintf("%d", beginning)
	}
	if length == 0 {
		beginning -= 1
	}
	return fmt.Sprintf("%d,%d", beginning, length)
}

func (c *changeDiff) filePath(prefix string, f *File) string {
	p := "/dev/null"
	if f != nil {
		p = filepath.Join(prefix, c.c.name())
	}

	return p
}

func (c *changeDiff) diffHeader(buf *bufio.Writer) error {
	_, err := buf.WriteString(fmt.Sprintf("diff --git %s %s\n",
		filepath.Join("a", c.c.name()), filepath.Join("b", c.c.name())))

	return err
}

func (c *changeDiff) binaryPatch(buf *bufio.Writer, from, to *File) error {
	if err := extendedHeaders(buf, from, to); err != nil {
		return err
	}

	_, err := buf.WriteString(fmt.Sprintf("Binary files %s and %s differ",
		filepath.Join("a", c.c.name()), filepath.Join("b", c.c.name())))

	return err
}

func (c *changeDiff) printIfBinaryPatch(buf *bufio.Writer, from, to *File) (bool, error) {
	fromIsBinary, err := checkIfBinary(from)
	if err != nil {
		return false, err
	}

	if fromIsBinary {
		err := c.binaryPatch(buf, from, to)

		return true, err
	}

	toIsBinary, err := checkIfBinary(to)
	if err != nil {
		return false, err
	}

	if toIsBinary {
		err := c.binaryPatch(buf, from, to)

		return true, err
	}

	return false, nil
}

func extendedHeaders(buf *bufio.Writer, from, to *File) error {
	//TODO these extended headers are not supported
	//copy from <path>
	//copy to <path>
	//rename from <path>
	//rename to <path>
	//similarity index <number>
	//dissimilarity index <number>
	var headers []string
	switch {
	case from == nil && to != nil:
		headers = []string{
			fmt.Sprintf("new file mode %s", to.Mode),
			fmt.Sprintf("index %s..%s", plumbing.ZeroHash, to.Hash),
		}
	case from != nil && to == nil:
		headers = []string{
			fmt.Sprintf("deleted file mode %s", from.Mode),
			fmt.Sprintf("index %s..%s", from.Hash, plumbing.ZeroHash),
		}
	case from != nil && to != nil:
		if from.Mode != to.Mode {
			headers = []string{
				fmt.Sprintf("old mode %s", from.Mode),
				fmt.Sprintf("new mode %s", to.Mode),
				fmt.Sprintf("index %s..%s", from.Hash, to.Hash),
			}
		} else {
			headers = []string{
				fmt.Sprintf("index %s..%s %s", from.Hash, to.Hash, to.Mode),
			}
		}
	}

	for _, h := range headers {
		if _, err := buf.WriteString(fmt.Sprintf("%s\n", h)); err != nil {
			return err
		}
	}

	return nil
}

func content(f *File) (string, error) {
	if f == nil {
		return "", nil
	}

	return f.Contents()
}

func checkIfBinary(f *File) (bool, error) {
	if f == nil {
		return false, nil
	}

	return f.IsBinary()
}
