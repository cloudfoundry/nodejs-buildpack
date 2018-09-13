package brats_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestBratsV3(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "V3 Brats Suite")
}
