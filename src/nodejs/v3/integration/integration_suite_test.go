package integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestIntegrationV3(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "V3 Integration Suite")
}
