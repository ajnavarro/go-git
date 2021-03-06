package http

import (
	"net/http"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteCommon struct{}

var _ = Suite(&SuiteCommon{})

func (s *SuiteCommon) TestNewBasicAuth(c *C) {
	a := NewBasicAuth("foo", "qux")

	c.Assert(a.Name(), Equals, "http-basic-auth")
	c.Assert(a.String(), Equals, "http-basic-auth - foo:*******")
}

func (s *SuiteCommon) TestNewHTTPError200(c *C) {
	res := &http.Response{StatusCode: 200}
	res.StatusCode = 200
	err := NewHTTPError(res)
	c.Assert(err, IsNil)
}

func (s *SuiteCommon) TestNewHTTPError401(c *C) {
	s.testNewHTTPError(c, 401, "authorization required")
}

func (s *SuiteCommon) TestNewHTTPError404(c *C) {
	s.testNewHTTPError(c, 404, "repository not found")
}

func (s *SuiteCommon) TestNewHTTPError40x(c *C) {
	s.testNewHTTPError(c, 402, "unexpected client error.*")
}

func (s *SuiteCommon) testNewHTTPError(c *C, code int, msg string) {
	req, _ := http.NewRequest("GET", "foo", nil)
	res := &http.Response{
		StatusCode: code,
		Request:    req,
	}

	err := NewHTTPError(res)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, msg)
}
