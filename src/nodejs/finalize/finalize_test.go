package finalize_test

import (
	"bytes"
	"io/ioutil"
	"nodejs/finalize"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=finalize.go --destination=mocks_test.go --package=finalize_test

var _ = Describe("Finalize", func() {
	var (
		err          error
		buildDir     string
		depsDir      string
		depsIdx      string
		finalizer    *finalize.Finalizer
		logger       *libbuildpack.Logger
		buffer       *bytes.Buffer
		mockCtrl     *gomock.Controller
		mockManifest *MockManifest
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "nodejs-buildpack.build.")
		Expect(err).To(BeNil())

		depsDir, err = ioutil.TempDir("", "nodejs-buildpack.deps.")
		Expect(err).To(BeNil())

		depsIdx = "9"
		Expect(os.MkdirAll(filepath.Join(depsDir, depsIdx), 0755)).To(Succeed())

		buffer = new(bytes.Buffer)

		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))

		mockCtrl = gomock.NewController(GinkgoT())
		mockManifest = NewMockManifest(mockCtrl)

		args := []string{buildDir, "", depsDir, depsIdx}
		stager := libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})

		finalizer = &finalize.Finalizer{
			Stager:   stager,
			Manifest: mockManifest,
			Log:      logger,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(depsDir)
		Expect(err).To(BeNil())
	})

	Describe("MoveNodeModulesToHome", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(filepath.Join(depsDir, depsIdx, "packages", "node_modules", "a", "b"), 0755)).To(Succeed())
		})

		It("moves node_modules back to app directory", func() {
			Expect(finalizer.MoveNodeModulesToHome()).To(Succeed())

			Expect(filepath.Join(buildDir, "node_modules", "a", "b")).To(BeADirectory())
			Expect(filepath.Join(depsDir, depsIdx, "packages")).ToNot(BeADirectory())
		})

		It("unsets NODE_PATH environment variable", func() {
			os.Setenv("NODE_PATH", "some value")

			Expect(finalizer.MoveNodeModulesToHome()).To(Succeed())

			_, set := os.LookupEnv("NODE_PATH")
			Expect(set).To(BeFalse())
		})

		Context("app/node_modules exists", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(buildDir, "node_modules", "c", "d"), 0755)).To(Succeed())
			})

			It("overwrites the existing node_modules directory", func() {
				Expect(finalizer.MoveNodeModulesToHome()).To(Succeed())

				Expect(filepath.Join(buildDir, "node_modules", "a", "b")).To(BeADirectory())
				Expect(filepath.Join(buildDir, "node_modules", "c", "d")).ToNot(BeADirectory())
				Expect(filepath.Join(depsDir, depsIdx, "packages")).ToNot(BeADirectory())
			})
		})

		Context("pkgDir/node_modules does NOT exist", func() {
			BeforeEach(func() {
				Expect(os.RemoveAll(filepath.Join(depsDir, depsIdx, "packages", "node_modules"))).To(Succeed())
			})

			It("does nothing", func() {
				Expect(finalizer.MoveNodeModulesToHome()).To(Succeed())
			})
		})
	})

	Describe("ReadPackageJSON", func() {
		Context("package.json has start script", func() {
			BeforeEach(func() {
				packageJSON := `
{
  "scripts" : {
		"script": "script",
		"start": "start-my-app",
		"thing": "thing"
	}
}
`
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "package.json"), []byte(packageJSON), 0644)).To(Succeed())
			})

			It("sets StartScript", func() {
				Expect(finalizer.ReadPackageJSON()).To(Succeed())
				Expect(finalizer.StartScript).To(Equal("start-my-app"))
			})
		})
	})

	Describe("CopyProfileScripts", func() {
		var buildpackDir string

		BeforeEach(func() {
			buildpackDir, err = ioutil.TempDir("", "nodejs-buildpack.buildpack.")
			Expect(err).To(BeNil())
			Expect(os.MkdirAll(filepath.Join(buildpackDir, "profile"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(buildpackDir, "profile", "test.sh"), []byte("Random Text"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(buildpackDir, "profile", "other.sh"), []byte("more Text"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(buildpackDir, "profile", "test.rb"), []byte("Ruby Text"), 0755)).To(Succeed())
			mockManifest.EXPECT().RootDir().Return(buildpackDir)
		})

		AfterEach(func() {
			Expect(os.RemoveAll(buildpackDir)).To(Succeed())
		})

		It("Copies scripts from <buildpack_dir>/profile to <dep_dir>/profile.d", func() {
			Expect(finalizer.CopyProfileScripts()).To(Succeed())
			Expect(ioutil.ReadFile(filepath.Join(depsDir, depsIdx, "profile.d", "test.sh"))).To(Equal([]byte("Random Text")))
			Expect(ioutil.ReadFile(filepath.Join(depsDir, depsIdx, "profile.d", "other.sh"))).To(Equal([]byte("more Text")))
		})

		It("Copies ruby scripts from <buildpack_dir>/profile to <dep_dir>/scripts", func() {
			Expect(finalizer.CopyProfileScripts()).To(Succeed())
			Expect(ioutil.ReadFile(filepath.Join(depsDir, depsIdx, "scripts", "test.rb"))).To(Equal([]byte("Ruby Text")))
			Expect(filepath.Join(depsDir, depsIdx, "profile.d", "test.rb")).ToNot(BeAnExistingFile())
		})

		It("Creates a profile.d file to source the ruby script", func() {
			Expect(finalizer.CopyProfileScripts()).To(Succeed())
			expected := "eval $(ruby $DEPS_DIR/9/scripts/test.rb)\n"
			actual, err := ioutil.ReadFile(filepath.Join(depsDir, depsIdx, "profile.d", "test.rb.sh"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(actual)).To(Equal(expected))
		})
	})

	Describe("WarnNoStart", func() {
		Context("Procfile exists", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "Procfile"), []byte("xxx"), 0644)).To(Succeed())
			})

			It("Doesn't log a warning", func() {
				Expect(finalizer.WarnNoStart()).To(Succeed())
				Expect(buffer.String()).To(Equal(""))
			})
		})

		Context("StartScript exists", func() {
			BeforeEach(func() {
				finalizer.StartScript = "npm run"
			})

			It("Doesn't log a warning", func() {
				Expect(finalizer.WarnNoStart()).To(Succeed())
				Expect(buffer.String()).To(Equal(""))
			})
		})

		Context("server.js exists", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "server.js"), []byte("xxx"), 0644)).To(Succeed())
			})

			It("Doesn't log a warning", func() {
				Expect(finalizer.WarnNoStart()).To(Succeed())
				Expect(buffer.String()).To(Equal(""))
			})
		})

		Context("none of the above exists", func() {
			It("logs a warning", func() {
				Expect(finalizer.WarnNoStart()).To(Succeed())
				Expect(buffer.String()).To(ContainSubstring("**WARNING** This app may not specify any way to start a node process\n"))
				Expect(buffer.String()).To(ContainSubstring("See: https://docs.cloudfoundry.org/buildpacks/node/node-tips.html#start"))
			})
		})
	})
})
