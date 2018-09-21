package build_test

import (
	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/libjavabuildpack/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"nodejs/v3/build"
	"path/filepath"
)

var _ = Describe("NewNode", func() {
	var stubNodeFixture = filepath.Join("v3", "stub-node.tar.gz")

	It("returns true if a build plan exists", func() {
		f := test.NewBuildFactory(T)
		f.AddBuildPlan(T, build.NodeDependency, libbuildpack.BuildPlanDependency{})
		f.AddDependency(T, build.NodeDependency, stubNodeFixture)

		_, ok, err := build.NewNode(f.Build)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
	})

	It("returns false if a build plan does not exist", func() {
		f := test.NewBuildFactory(T)

		_, ok, err := build.NewNode(f.Build)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
	})

	It("contributes node to the launch layer", func() {
		f := test.NewBuildFactory(T)
		f.AddBuildPlan(T, build.NodeDependency, libbuildpack.BuildPlanDependency{})
		f.AddDependency(T, build.NodeDependency, stubNodeFixture)

		nodeDep, _, err := build.NewNode(f.Build)
		Expect(err).NotTo(HaveOccurred())

		err = nodeDep.Contribute()
		Expect(err).NotTo(HaveOccurred())

		layerRoot := filepath.Join(f.Build.Launch.Root, "node")
		Expect(filepath.Join(layerRoot, "stub.txt")).To(BeARegularFile())
	})
})
