package build_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

var T *testing.T

func TestBuild(t *testing.T) {
	T = t
	RegisterFailHandler(Fail)
	RunSpecs(t, "V3 Build Suite")
}
