package model_test

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/revel/cmd/model"
)

var versionTests = [][]string{
	{"v0.20.0-dev", "v0.20.0-dev"},
	{"v0.20-dev", "v0.20.0-dev"},
	{"v0.20.", "v0.20.0"},
	{"2.0", "2.0.0"},
}
// Test that the event handler can be attached and it dispatches the event received
func TestVersion(t *testing.T) {
	for _, v:= range versionTests {
		p,e:=model.ParseVersion(v[0])
		assert.Nil(t,e,"Should have parsed %s",v)
		assert.Equal(t,p.String(),v[1], "Should be equal %s==%s",p.String(),v)
	}
}

// test the ranges
func TestVersionRange(t *testing.T) {
	a,_ := model.ParseVersion("0.1.2")
	b,_ := model.ParseVersion("0.2.1")
	c,_ := model.ParseVersion("1.0.1")
	assert.True(t, b.MinorNewer(a), "B is newer then A")
	assert.False(t, b.MajorNewer(a), "B is not major newer then A")
	assert.False(t, b.MajorNewer(c), "B is not major newer then A")
	assert.True(t, c.MajorNewer(b), "C is major newer then b")
}