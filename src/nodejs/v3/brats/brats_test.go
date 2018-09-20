package brats_test

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

	It("should run V3 detect", func() {
		detectResult, err := dagg.Detect(filepath.Join(rootDir, "fixtures", "simple_app"))
		Expect(err).ToNot(HaveOccurred())

		Expect(len(detectResult.Group.Buildpacks)).To(Equal(1))
		Expect(detectResult.Group.Buildpacks[0].Id).To(Equal("org.cloudfoundry.buildpacks.nodejs"))
		Expect(detectResult.Group.Buildpacks[0].Version).To(Equal("1.6.32"))

		Expect(len(detectResult.BuildPlan)).To(Equal(1))
		Expect(detectResult.BuildPlan).To(HaveKey("node"))
		Expect(detectResult.BuildPlan["node"].Version).To(Equal("~>10"))
	})

	It("should run V3 build", func() {
		launch, err := dagg.Build(filepath.Join(rootDir, "fixtures", "simple_app"))
		Expect(err).ToNot(HaveOccurred())

		Expect(len(launch.Processes)).To(Equal(1))
		Expect(launch.Processes[0].Type).To(Equal("web"))
		Expect(launch.Processes[0].Command).To(Equal("npm start"))
	})
})
