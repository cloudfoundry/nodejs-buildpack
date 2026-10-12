package pnpm_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPnpm(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pnpm Suite")
}
