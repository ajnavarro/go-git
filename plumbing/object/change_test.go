package object

import (
	"sort"
	"fmt"

	"srcd.works/go-git.v4/plumbing"
	"srcd.works/go-git.v4/plumbing/filemode"
	"srcd.works/go-git.v4/plumbing/storer"
	"srcd.works/go-git.v4/storage/filesystem"
	"srcd.works/go-git.v4/utils/merkletrie"

	fixtures "github.com/src-d/go-git-fixtures"
	. "gopkg.in/check.v1"
)

type ChangeSuite struct {
	fixtures.Suite
	Storer  storer.EncodedObjectStorer
	Fixture *fixtures.Fixture
}

func (s *ChangeSuite) SetUpSuite(c *C) {
	s.Suite.SetUpSuite(c)
	s.Fixture = fixtures.ByURL("https://github.com/src-d/go-git.git").
		ByTag(".git").One()
	sto, err := filesystem.NewStorage(s.Fixture.DotGit())
	c.Assert(err, IsNil)
	s.Storer = sto
}

func (s *ChangeSuite) tree(c *C, h plumbing.Hash) *Tree {
	t, err := GetTree(s.Storer, h)
	c.Assert(err, IsNil)
	return t
}

var _ = Suite(&ChangeSuite{})

func (s *ChangeSuite) TestInsert(c *C) {
	// Commit a5078b19f08f63e7948abd0a5e2fb7d319d3a565 of the go-git
	// fixture inserted "examples/clone/main.go".
	//
	// On that commit, the "examples/clone" tree is
	//     6efca3ff41cab651332f9ebc0c96bb26be809615
	//
	// and the "examples/colone/main.go" is
	//     f95dc8f7923add1a8b9f72ecb1e8db1402de601a

	path := "examples/clone/main.go"
	name := "main.go"
	mode := filemode.Regular
	blob := plumbing.NewHash("f95dc8f7923add1a8b9f72ecb1e8db1402de601a")
	tree := plumbing.NewHash("6efca3ff41cab651332f9ebc0c96bb26be809615")

	change := &Change{
		From: empty,
		To: ChangeEntry{
			Name: path,
			Tree: s.tree(c, tree),
			TreeEntry: TreeEntry{
				Name: name,
				Mode: mode,
				Hash: blob,
			},
		},
	}

	action, err := change.Action()
	c.Assert(err, IsNil)
	c.Assert(action, Equals, merkletrie.Insert)

	from, to, err := change.Files()
	c.Assert(err, IsNil)
	c.Assert(from, IsNil)
	c.Assert(to.Name, Equals, name)
	c.Assert(to.Blob.Hash, Equals, blob)

	str := change.String()
	c.Assert(str, Equals, "<Action: Insert, Path: examples/clone/main.go>")

	p, err := change.Patch()
	c.Assert(err, IsNil)
	fmt.Println(p)
	//c.Assert(p, DeepEquals, diffInsert)
}

func (s *ChangeSuite) TestDelete(c *C) {
	// Commit f6011d65d57c2a866e231fc21a39cb618f86f9ea of the go-git
	// fixture deleted "utils/difftree/difftree.go".
	//
	// The parent of that commit is
	//     9b4a386db3d98a4362516a00ef3d04d4698c9bcd.
	//
	// On that parent commit, the "utils/difftree" tree is
	//     f3d11566401ce4b0808aab9dd6fad3d5abf1481a.
	//
	// and the "utils/difftree/difftree.go" is
	//     e2cb9a5719daf634d45a063112b4044ee81da13ea.

	path := "utils/difftree/difftree.go"
	name := "difftree.go"
	mode := filemode.Regular
	blob := plumbing.NewHash("e2cb9a5719daf634d45a063112b4044ee81da13e")
	tree := plumbing.NewHash("f3d11566401ce4b0808aab9dd6fad3d5abf1481a")

	change := &Change{
		From: ChangeEntry{
			Name: path,
			Tree: s.tree(c, tree),
			TreeEntry: TreeEntry{
				Name: name,
				Mode: mode,
				Hash: blob,
			},
		},
		To: empty,
	}

	action, err := change.Action()
	c.Assert(err, IsNil)
	c.Assert(action, Equals, merkletrie.Delete)

	from, to, err := change.Files()
	c.Assert(err, IsNil)
	c.Assert(to, IsNil)
	c.Assert(from.Name, Equals, name)
	c.Assert(from.Blob.Hash, Equals, blob)

	str := change.String()
	c.Assert(str, Equals, "<Action: Delete, Path: utils/difftree/difftree.go>")

	//// TODO check diff result
	//p, err := change.Patch()
	//c.Assert(err, IsNil)
	//fmt.Println(p)
}

func (s *ChangeSuite) TestModify(c *C) {
	// Commit 7beaad711378a4daafccc2c04bc46d36df2a0fd1 of the go-git
	// fixture modified "examples/latest/latest.go".
	// the "examples/latest" tree is
	//     b1f01b730b855c82431918cb338ad47ed558999b.
	// and "examples/latest/latest.go" is blob
	//     05f583ace3a9a078d8150905a53a4d82567f125f.
	//
	// The parent of that commit is
	//     337148ef6d751477796922ac127b416b8478fcc4.
	// the "examples/latest" tree is
	//     8b0af31d2544acb5c4f3816a602f11418cbd126e.
	// and "examples/latest/latest.go" is blob
	//     de927fad935d172929aacf20e71f3bf0b91dd6f9.

	path := "utils/difftree/difftree.go"
	name := "difftree.go"
	mode := filemode.Regular
	fromBlob := plumbing.NewHash("05f583ace3a9a078d8150905a53a4d82567f125f")
	fromTree := plumbing.NewHash("b1f01b730b855c82431918cb338ad47ed558999b")
	toBlob := plumbing.NewHash("de927fad935d172929aacf20e71f3bf0b91dd6f9")
	toTree := plumbing.NewHash("8b0af31d2544acb5c4f3816a602f11418cbd126e")

	change := &Change{
		From: ChangeEntry{
			Name: path,
			Tree: s.tree(c, fromTree),
			TreeEntry: TreeEntry{
				Name: name,
				Mode: mode,
				Hash: fromBlob,
			},
		},
		To: ChangeEntry{
			Name: path,
			Tree: s.tree(c, toTree),
			TreeEntry: TreeEntry{
				Name: name,
				Mode: mode,
				Hash: toBlob,
			},
		},
	}

	action, err := change.Action()
	c.Assert(err, IsNil)
	c.Assert(action, Equals, merkletrie.Modify)

	from, to, err := change.Files()
	c.Assert(err, IsNil)

	c.Assert(from.Name, Equals, name)
	c.Assert(from.Blob.Hash, Equals, fromBlob)
	c.Assert(to.Name, Equals, name)
	c.Assert(to.Blob.Hash, Equals, toBlob)

	str := change.String()
	c.Assert(str, Equals, "<Action: Modify, Path: utils/difftree/difftree.go>")

	//// TODO check diff result
	//p, err := change.Patch()
	//c.Assert(err, IsNil)
	//fmt.Println(p)
}

func (s *ChangeSuite) TestEmptyChangeFails(c *C) {
	change := &Change{}

	_, err := change.Action()
	c.Assert(err, ErrorMatches, "malformed.*")

	_, _, err = change.Files()
	c.Assert(err, ErrorMatches, "malformed.*")

	str := change.String()
	c.Assert(str, Equals, "malformed change")
}

func (s *ChangeSuite) TestErrorsFindingChildsAreDetected(c *C) {
	// Commit 7beaad711378a4daafccc2c04bc46d36df2a0fd1 of the go-git
	// fixture modified "examples/latest/latest.go".
	// the "examples/latest" tree is
	//     b1f01b730b855c82431918cb338ad47ed558999b.
	// and "examples/latest/latest.go" is blob
	//     05f583ace3a9a078d8150905a53a4d82567f125f.
	//
	// The parent of that commit is
	//     337148ef6d751477796922ac127b416b8478fcc4.
	// the "examples/latest" tree is
	//     8b0af31d2544acb5c4f3816a602f11418cbd126e.
	// and "examples/latest/latest.go" is blob
	//     de927fad935d172929aacf20e71f3bf0b91dd6f9.

	path := "utils/difftree/difftree.go"
	name := "difftree.go"
	mode := filemode.Regular
	fromBlob := plumbing.NewHash("aaaa") // does not exists
	fromTree := plumbing.NewHash("b1f01b730b855c82431918cb338ad47ed558999b")
	toBlob := plumbing.NewHash("bbbb") // does not exists
	toTree := plumbing.NewHash("8b0af31d2544acb5c4f3816a602f11418cbd126e")

	change := &Change{
		From: ChangeEntry{
			Name: path,
			Tree: s.tree(c, fromTree),
			TreeEntry: TreeEntry{
				Name: name,
				Mode: mode,
				Hash: fromBlob,
			},
		},
		To: ChangeEntry{},
	}

	_, _, err := change.Files()
	c.Assert(err, ErrorMatches, "object not found")

	change = &Change{
		From: empty,
		To: ChangeEntry{
			Name: path,
			Tree: s.tree(c, toTree),
			TreeEntry: TreeEntry{
				Name: name,
				Mode: mode,
				Hash: toBlob,
			},
		},
	}

	_, _, err = change.Files()
	c.Assert(err, ErrorMatches, "object not found")
}

func (s *ChangeSuite) TestChangesString(c *C) {
	expected := "[]"
	changes := Changes{}
	obtained := changes.String()
	c.Assert(obtained, Equals, expected)

	expected = "[<Action: Modify, Path: bla>]"
	changes = make([]*Change, 1)
	changes[0] = &Change{}
	changes[0].From.Name = "bla"
	changes[0].To.Name = "bla"

	obtained = changes.String()
	c.Assert(obtained, Equals, expected)

	expected = "[<Action: Modify, Path: bla>, <Action: Delete, Path: foo/bar>]"
	changes = make([]*Change, 2)
	changes[0] = &Change{}
	changes[0].From.Name = "bla"
	changes[0].To.Name = "bla"
	changes[1] = &Change{}
	changes[1].From.Name = "foo/bar"
	obtained = changes.String()
	c.Assert(obtained, Equals, expected)
}

func (s *ChangeSuite) TestChangesSort(c *C) {
	changes := make(Changes, 3)
	changes[0] = &Change{}
	changes[0].From.Name = "z"
	changes[0].To.Name = "z"
	changes[1] = &Change{}
	changes[1].From.Name = "b/b"
	changes[2] = &Change{}
	changes[2].To.Name = "b/a"

	expected := "[<Action: Insert, Path: b/a>, " +
		"<Action: Delete, Path: b/b>, " +
		"<Action: Modify, Path: z>]"

	sort.Sort(changes)
	c.Assert(changes.String(), Equals, expected)
}

// TODO file modes added a 0 before the number
// TODO short hashes not supported
var diffInsert = `diff --git a/examples/clone/main.go b/examples/clone/main.go
new file mode 0100644
index 0000000000000000000000000000000000000000..f95dc8f7923add1a8b9f72ecb1e8db1402de601a
--- /dev/null
+++ b/examples/clone/main.go
@@ -0,0 +1,84 @@
+package main
+
+import (
+	"io"
+	"os"
+	"path/filepath"
+
+	"github.com/fatih/color"
+
+	"gopkg.in/src-d/go-git.v4"
+)
+
+func main() {
+	checkArgs()
+	url := os.Args[1]
+	directory := os.Args[2]
+
+	r := git.NewMemoryRepository()
+
+	// Clone the given repository, using depth we create a shallow clone :
+	// > git clone <url> --depth 1
+	color.Blue("git clone %s --depth 1 %s", url, directory)
+
+	err := r.Clone(&git.CloneOptions{
+		URL:   url,
+		Depth: 1,
+	})
+	checkIfError(err)
+
+	// ... retrieving the branch being pointed by HEAD
+	ref, err := r.Head()
+	checkIfError(err)
+	// ... retrieving the commit object
+	commit, err := r.Commit(ref.Hash())
+	checkIfError(err)
+
+	// ... we get all the files from the commit
+	files, err := commit.Files()
+	checkIfError(err)
+
+	// ... now we iterate the files to save to disk
+	err = files.ForEach(func(f *git.File) error {
+		abs := filepath.Join(directory, f.Name)
+		dir := filepath.Dir(abs)
+
+		os.MkdirAll(dir, 0777)
+		file, err := os.Create(abs)
+		if err != nil {
+			return err
+		}
+
+		defer file.Close()
+		r, err := f.Reader()
+		if err != nil {
+			return err
+		}
+
+		defer r.Close()
+
+		if err := file.Chmod(f.Mode); err != nil {
+			return err
+		}
+
+		_, err = io.Copy(file, r)
+		return err
+	})
+	checkIfError(err)
+}
+
+func checkIfError(err error) {
+	if err == nil {
+		return
+	}
+
+	color.Red("error: %s", err)
+	os.Exit(1)
+}
+
+func checkArgs() {
+	if len(os.Args) < 3 {
+		color.Cyan("Usage: %s <url> <directory>", os.Args[0])
+		os.Exit(1)
+	}
+}`
