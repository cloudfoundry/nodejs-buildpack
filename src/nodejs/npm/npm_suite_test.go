package npm_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestNpm(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Npm Suite")
}
