package acceptance_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestAcceptanceV3(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "V3 Acceptance Suite")
}
