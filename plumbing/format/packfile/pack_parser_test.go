package packfile

import (
	"os"

	. "gopkg.in/check.v1"
)

type PackParserSuite struct {
	pp *PackParser
}

var _ = Suite(&PackParserSuite{})

func (s *PackParserSuite) SetUpTest(c *C) {
	pf, err := os.Open("/home/antonio/workspace/git/.git/objects/pack/pack-5df0417bd8b9f280b8b9a4a28a7a12d52d6f62db.pack")
	c.Assert(err, IsNil)
	s.pp = NewPackParser(pf)
}

func (s *PackParserSuite) BenchmarkPackParsing(c *C) {
	err := s.pp.Parse()
	c.Assert(err, IsNil)

	c.Log(len(s.pp.objectInfos))
	c.Fail()
}
