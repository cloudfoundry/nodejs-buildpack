package pnpm_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPNPM(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PNPM Suite")
}
