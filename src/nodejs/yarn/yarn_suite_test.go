package yarn_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestYarn(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Yarn Suite")
}
