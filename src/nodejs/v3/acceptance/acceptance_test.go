package acceptance_test

import (
	"github.com/cloudfoundry/libbuildpack/cutlass"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"nodejs/v3/dagger"
	"path/filepath"
)

var _ = Describe("Nodejs V3 buildpack", func() {
	var (
		rootDir string
		dagg    *dagger.Dagger
	)

	BeforeEach(func() {
		var err error

		rootDir, err = cutlass.FindRoot()
		Expect(err).ToNot(HaveOccurred())

		dagg, err = dagger.NewDagger(rootDir)
		Expect(err).ToNot(HaveOccurred())

		err = dagg.BundleBuildpack()
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		dagg.Destroy()
	})

	It("should create a working app in an OCI image", func() {
		app, err := dagg.Pack(filepath.Join(rootDir, "fixtures", "simple_app"))
		Expect(err).ToNot(HaveOccurred())

		err = app.Start()
		Expect(err).ToNot(HaveOccurred())
		defer app.Destroy()

		err = app.HTTPGet("/")
		Expect(err).ToNot(HaveOccurred())
	})
})
