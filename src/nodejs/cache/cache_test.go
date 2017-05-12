package cache_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"nodejs/cache"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=cache.go --destination=mocks_test.go --package=cache_test

var _ = Describe("Cache", func() {
	var (
		err         error
		buildDir    string
		cacheDir    string
		c           *cache.Cache
		stager      *libbuildpack.Stager
		logger      *libbuildpack.Logger
		buffer      *bytes.Buffer
		mockCtrl    *gomock.Controller
		mockCommand *MockCommand
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "nodejs-buildpack.build.")
		Expect(err).To(BeNil())

		cacheDir, err = ioutil.TempDir("", "nodejs-buildpack.cache.")
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)

		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))

		mockCtrl = gomock.NewController(GinkgoT())
		mockCommand = NewMockCommand(mockCtrl)

		args := []string{buildDir, cacheDir}
		stager = libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})

		c = &cache.Cache{
			Stager:               stager,
			Log:                  logger,
			Command:              mockCommand,
			NodeVersion:          "1.1.1",
			NPMVersion:           "2.2.2",
			YarnVersion:          "3.3.3",
			PackageJSONCacheDirs: []string{},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(cacheDir)
		Expect(err).To(BeNil())
	})

	Describe("Initialize", func() {
		BeforeEach(func() {
			mockCommand.EXPECT().Execute("", gomock.Any(), gomock.Any(), "node", "--version").Do(func(_ string, buffer io.Writer, _ io.Writer, _ string, _ string) {
				buffer.Write([]byte("6.9.3\n"))
			}).Return(nil)

			mockCommand.EXPECT().Execute("", gomock.Any(), gomock.Any(), "npm", "--version").Do(func(_ string, buffer io.Writer, _ io.Writer, _ string, _ string) {
				buffer.Write([]byte("4.5.6\n"))
			}).Return(nil)

			mockCommand.EXPECT().Execute("", gomock.Any(), gomock.Any(), "yarn", "--version").Do(func(_ string, buffer io.Writer, _ io.Writer, _ string, _ string) {
				buffer.Write([]byte("9.8.7\n"))
			}).Return(nil)
		})

		It("sets node version", func() {
			Expect(c.Initialize()).To(Succeed())
			Expect(c.NodeVersion).To(Equal("6.9.3"))
		})

		It("sets npm version", func() {
			Expect(c.Initialize()).To(Succeed())
			Expect(c.NPMVersion).To(Equal("4.5.6"))
		})

		It("sets yarn version", func() {
			Expect(c.Initialize()).To(Succeed())
			Expect(c.YarnVersion).To(Equal("9.8.7"))
		})

		It("sets yarn version", func() {
			Expect(c.Initialize()).To(Succeed())
			Expect(c.YarnVersion).To(Equal("9.8.7"))
		})

		It("initializes config based values", func() {
			Expect(c.Initialize()).To(Succeed())
			Expect(c.PackageJSONCacheDirs).To(Equal([]string{}))
		})

		Context("package.json has cacheDirectories", func() {
			BeforeEach(func() {
				packageJSON := `
{
  "cacheDirectories" : [
		"first",
		"second"
	]
}
`
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "package.json"), []byte(packageJSON), 0644)).To(Succeed())
			})

			It("sets PackageJSONCacheDirs", func() {
				Expect(c.Initialize()).To(Succeed())
				Expect(c.PackageJSONCacheDirs).To(Equal([]string{"first", "second"}))
			})
		})

		Context("package.json has cache_directories", func() {
			BeforeEach(func() {
				packageJSON := `
{
  "cache_directories" : [
		"third",
		"fourth"
	]
}
`
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "package.json"), []byte(packageJSON), 0644)).To(Succeed())
			})

			It("sets PackageJSONCacheDirs", func() {
				Expect(c.Initialize()).To(Succeed())
				Expect(c.PackageJSONCacheDirs).To(Equal([]string{"third", "fourth"}))
			})
		})

	})

	Describe("Restore", func() {
		Context("there is a cache", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(cacheDir, "node", ".npm"), 0755)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(cacheDir, "node", ".npm", "cached"), []byte("xxx"), 0644)).To(Succeed())

				Expect(os.MkdirAll(filepath.Join(cacheDir, "node", ".cache", "yarn"), 0755)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(cacheDir, "node", ".cache", "yarn", "cached"), []byte("yyy"), 0644)).To(Succeed())

				Expect(os.MkdirAll(filepath.Join(cacheDir, "node", "bower_components"), 0755)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(cacheDir, "node", "bower_components", "cached"), []byte("zzz"), 0644)).To(Succeed())

				Expect(os.MkdirAll(filepath.Join(cacheDir, "node", "other_dir"), 0755)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(cacheDir, "node", "other_dir", "cached"), []byte("aaa"), 0644)).To(Succeed())
			})

			Context("signature changed", func() {
				BeforeEach(func() {
					Expect(ioutil.WriteFile(filepath.Join(cacheDir, "node", "signature"), []byte("1; 2; 3\n"), 0644)).To(BeNil())
				})

				It("alerts user", func() {
					Expect(c.Restore()).To(Succeed())

					Expect(buffer.String()).To(ContainSubstring("Skipping cache restore (new runtime signature)"))
				})

				It("does not restore the cache", func() {
					Expect(c.Restore()).To(Succeed())
					files, err := ioutil.ReadDir(filepath.Join(buildDir))
					Expect(err).To(BeNil())
					Expect(len(files)).To(Equal(0))
				})
			})

			Context("signatures match", func() {
				BeforeEach(func() {
					Expect(ioutil.WriteFile(filepath.Join(cacheDir, "node", "signature"), []byte("1.1.1; 2.2.2; 3.3.3\n"), 0644)).To(BeNil())
				})

				Context("cached directories are not in build dir", func() {
					It("alerts user", func() {
						Expect(c.Restore()).To(Succeed())

						Expect(buffer.String()).To(ContainSubstring("Loading 3 from cacheDirectories (default):"))
						Expect(buffer.String()).To(ContainSubstring("- .npm\n"))
						Expect(buffer.String()).To(ContainSubstring("- .cache/yarn\n"))
						Expect(buffer.String()).To(ContainSubstring("- bower_components\n"))
					})

					It("moves the requested cached directories", func() {
						Expect(c.Restore()).To(Succeed())
						files, err := ioutil.ReadDir(filepath.Join(buildDir))
						Expect(err).To(BeNil())

						Expect(len(files)).To(Equal(3))
						Expect(ioutil.ReadFile(filepath.Join(buildDir, ".npm", "cached"))).To(Equal([]byte("xxx")))
						Expect(ioutil.ReadFile(filepath.Join(buildDir, ".cache", "yarn", "cached"))).To(Equal([]byte("yyy")))
						Expect(ioutil.ReadFile(filepath.Join(buildDir, "bower_components", "cached"))).To(Equal([]byte("zzz")))
					})
				})

				Context("PackageJSONCacheDirs is set", func() {
					BeforeEach(func() {
						Expect(os.MkdirAll(filepath.Join(cacheDir, "node", "first"), 0755)).To(Succeed())
						Expect(ioutil.WriteFile(filepath.Join(cacheDir, "node", "first", "cached"), []byte("thing 1"), 0644)).To(Succeed())

						Expect(os.MkdirAll(filepath.Join(cacheDir, "node", "second"), 0755)).To(Succeed())
						Expect(ioutil.WriteFile(filepath.Join(cacheDir, "node", "second", "cached"), []byte("thing 2"), 0644)).To(Succeed())

						c.PackageJSONCacheDirs = []string{"first", "second"}
					})

					It("alerts user", func() {
						Expect(c.Restore()).To(Succeed())

						Expect(buffer.String()).To(ContainSubstring("Loading 2 from cacheDirectories (package.json):"))
						Expect(buffer.String()).To(ContainSubstring("- first\n"))
						Expect(buffer.String()).To(ContainSubstring("- second\n"))
					})

					It("moves the requested cached directories", func() {
						Expect(c.Restore()).To(Succeed())
						files, err := ioutil.ReadDir(filepath.Join(buildDir))
						Expect(err).To(BeNil())

						Expect(len(files)).To(Equal(2))
						Expect(ioutil.ReadFile(filepath.Join(buildDir, "first", "cached"))).To(Equal([]byte("thing 1")))
						Expect(ioutil.ReadFile(filepath.Join(buildDir, "second", "cached"))).To(Equal([]byte("thing 2")))
					})
				})

				Context("some cached directories are already in build dir", func() {
					BeforeEach(func() {
						Expect(os.MkdirAll(filepath.Join(buildDir, ".npm"), 0755)).To(Succeed())
						Expect(ioutil.WriteFile(filepath.Join(buildDir, ".npm", "cached"), []byte("from app"), 0644)).To(Succeed())
					})

					It("alerts user", func() {
						Expect(c.Restore()).To(Succeed())

						Expect(buffer.String()).To(ContainSubstring("Loading 3 from cacheDirectories (default):"))
						Expect(buffer.String()).To(ContainSubstring("- .npm (exists - skipping)\n"))
						Expect(buffer.String()).To(ContainSubstring("- .cache/yarn\n"))
						Expect(buffer.String()).To(ContainSubstring("- bower_components\n"))
					})

					It("moves the requested cached directories", func() {
						Expect(c.Restore()).To(Succeed())
						files, err := ioutil.ReadDir(filepath.Join(buildDir))
						Expect(err).To(BeNil())

						Expect(len(files)).To(Equal(3))
						Expect(ioutil.ReadFile(filepath.Join(buildDir, ".npm", "cached"))).To(Equal([]byte("from app")))
						Expect(ioutil.ReadFile(filepath.Join(buildDir, ".cache", "yarn", "cached"))).To(Equal([]byte("yyy")))
						Expect(ioutil.ReadFile(filepath.Join(buildDir, "bower_components", "cached"))).To(Equal([]byte("zzz")))
					})
				})

				Context("some requested directories are not in cache dir", func() {
					BeforeEach(func() {
						Expect(os.RemoveAll(filepath.Join(cacheDir, "node", ".npm"))).To(Succeed())
					})

					It("alerts user", func() {
						Expect(c.Restore()).To(Succeed())

						Expect(buffer.String()).To(ContainSubstring("Loading 3 from cacheDirectories (default):"))
						Expect(buffer.String()).To(ContainSubstring("- .npm (not cached - skipping)\n"))
						Expect(buffer.String()).To(ContainSubstring("- .cache/yarn\n"))
						Expect(buffer.String()).To(ContainSubstring("- bower_components\n"))
					})

					It("moves the requested cached directories", func() {
						Expect(c.Restore()).To(Succeed())
						files, err := ioutil.ReadDir(filepath.Join(buildDir))
						Expect(err).To(BeNil())

						Expect(len(files)).To(Equal(2))
						Expect(ioutil.ReadFile(filepath.Join(buildDir, ".cache", "yarn", "cached"))).To(Equal([]byte("yyy")))
						Expect(ioutil.ReadFile(filepath.Join(buildDir, "bower_components", "cached"))).To(Equal([]byte("zzz")))
					})
				})

				Context("NODE_MODULES_CACHE is false", func() {
					var oldNodeModulesCache string

					BeforeEach(func() {
						oldNodeModulesCache = os.Getenv("NODE_MODULES_CACHE")
						Expect(os.Setenv("NODE_MODULES_CACHE", "false")).To(Succeed())
					})

					AfterEach(func() {
						Expect(os.Setenv("NODE_MODULES_CACHE", oldNodeModulesCache)).To(Succeed())
					})

					It("alerts user", func() {
						Expect(c.Restore()).To(Succeed())

						Expect(buffer.String()).To(ContainSubstring("Skipping cache restore (disabled by config)"))
					})

					It("does not restore the cache", func() {
						Expect(c.Restore()).To(Succeed())
						files, err := ioutil.ReadDir(filepath.Join(buildDir))
						Expect(err).To(BeNil())
						Expect(len(files)).To(Equal(0))
					})
				})
			})
		})

		Context("there is not a cache", func() {
			It("alerts user", func() {
				Expect(c.Restore()).To(Succeed())
				Expect(buffer.String()).To(ContainSubstring("Skipping cache restore (no previous cache)"))
			})
		})
	})

	Describe("Save", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(filepath.Join(cacheDir, "node"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(cacheDir, "node", "old_cache"), []byte("old"), 0644)).To(Succeed())

			Expect(os.MkdirAll(filepath.Join(buildDir, ".npm"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(buildDir, ".npm", "build1"), []byte("build1"), 0644)).To(Succeed())

			Expect(os.MkdirAll(filepath.Join(buildDir, ".cache", "yarn"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(buildDir, ".cache", "yarn", "build2"), []byte("build2"), 0644)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(buildDir, ".cache", "build3"), []byte("build3"), 0644)).To(Succeed())

			Expect(os.MkdirAll(filepath.Join(buildDir, "bower_components"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(buildDir, "bower_components", "build4"), []byte("build4"), 0644)).To(Succeed())
		})

		It("clears the previous cache", func() {
			Expect(c.Save()).To(Succeed())
			Expect(filepath.Join(cacheDir, "node", "old_cache")).NotTo(BeAnExistingFile())
			Expect(buffer.String()).To(ContainSubstring("Clearing previous node cache"))
		})

		It("saves the signature to the cache", func() {
			Expect(c.Save()).To(Succeed())
			Expect(ioutil.ReadFile(filepath.Join(cacheDir, "node", "signature"))).To(Equal([]byte("1.1.1; 2.2.2; 3.3.3\n")))
		})

		It("removes .npm and .cache/yarn from the build dir", func() {
			Expect(c.Save()).To(Succeed())
			Expect(filepath.Join(buildDir, ".npm")).NotTo(BeAnExistingFile())
			Expect(filepath.Join(buildDir, ".cache", "yarn")).NotTo(BeAnExistingFile())
			Expect(ioutil.ReadFile(filepath.Join(buildDir, ".cache", "build3"))).To(Equal([]byte("build3")))
		})

		Context("NODE_MODULES_CACHE is false", func() {
			var oldNodeModulesCache string

			BeforeEach(func() {
				oldNodeModulesCache = os.Getenv("NODE_MODULES_CACHE")
				Expect(os.Setenv("NODE_MODULES_CACHE", "false")).To(Succeed())
			})

			AfterEach(func() {
				Expect(os.Setenv("NODE_MODULES_CACHE", oldNodeModulesCache)).To(Succeed())
			})

			It("alerts user", func() {
				Expect(c.Save()).To(Succeed())

				Expect(buffer.String()).To(ContainSubstring("Skipping cache save (disabled by config)"))
			})

			It("does not save the cache", func() {
				Expect(c.Save()).To(Succeed())
				files, err := filepath.Glob(filepath.Join(cacheDir, "node", "*"))
				Expect(err).To(BeNil())
				Expect(files).To(Equal([]string{filepath.Join(cacheDir, "node", "signature")}))
			})
		})

		Context("PackageJSONCacheDirs is set", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(buildDir, "first"), 0755)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "first", "cached"), []byte("thing 1"), 0644)).To(Succeed())

				Expect(os.MkdirAll(filepath.Join(buildDir, "second"), 0755)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "second", "cached"), []byte("thing 2"), 0644)).To(Succeed())

				c.PackageJSONCacheDirs = []string{"first", "second"}
			})

			It("alerts user", func() {
				Expect(c.Save()).To(Succeed())

				Expect(buffer.String()).To(ContainSubstring("Saving 2 cacheDirectories (package.json):"))
				Expect(buffer.String()).To(ContainSubstring("- first\n"))
				Expect(buffer.String()).To(ContainSubstring("- second\n"))
			})

			It("moves the requested cached directories", func() {
				Expect(c.Save()).To(Succeed())
				files, err := ioutil.ReadDir(filepath.Join(cacheDir, "node"))
				Expect(err).To(BeNil())

				Expect(len(files)).To(Equal(3))
				Expect(ioutil.ReadFile(filepath.Join(cacheDir, "node", "signature"))).To(Equal([]byte("1.1.1; 2.2.2; 3.3.3\n")))
				Expect(ioutil.ReadFile(filepath.Join(cacheDir, "node", "first", "cached"))).To(Equal([]byte("thing 1")))
				Expect(ioutil.ReadFile(filepath.Join(cacheDir, "node", "second", "cached"))).To(Equal([]byte("thing 2")))
			})
		})

		Context("default cache dirs exist in build dir", func() {
			It("alerts user", func() {
				Expect(c.Save()).To(Succeed())

				Expect(buffer.String()).To(ContainSubstring("Saving 3 cacheDirectories (default):"))
				Expect(buffer.String()).To(ContainSubstring("- .npm\n"))
				Expect(buffer.String()).To(ContainSubstring("- .cache/yarn\n"))
				Expect(buffer.String()).To(ContainSubstring("- bower_components\n"))
			})

			It("moves the requested cached directories", func() {
				Expect(c.Save()).To(Succeed())
				files, err := ioutil.ReadDir(filepath.Join(cacheDir, "node"))
				Expect(err).To(BeNil())

				Expect(len(files)).To(Equal(4))
				Expect(ioutil.ReadFile(filepath.Join(cacheDir, "node", "signature"))).To(Equal([]byte("1.1.1; 2.2.2; 3.3.3\n")))
				Expect(ioutil.ReadFile(filepath.Join(cacheDir, "node", ".npm", "build1"))).To(Equal([]byte("build1")))
				Expect(ioutil.ReadFile(filepath.Join(cacheDir, "node", ".cache", "yarn", "build2"))).To(Equal([]byte("build2")))
				Expect(ioutil.ReadFile(filepath.Join(cacheDir, "node", "bower_components", "build4"))).To(Equal([]byte("build4")))
			})
		})

		Context("some requested directories are not in cache dir", func() {
			BeforeEach(func() {
				Expect(os.RemoveAll(filepath.Join(buildDir, ".npm"))).To(Succeed())
			})

			It("alerts user", func() {
				Expect(c.Save()).To(Succeed())

				Expect(buffer.String()).To(ContainSubstring("Saving 3 cacheDirectories (default):"))
				Expect(buffer.String()).To(ContainSubstring("- .npm (nothing to cache)\n"))
				Expect(buffer.String()).To(ContainSubstring("- .cache/yarn\n"))
				Expect(buffer.String()).To(ContainSubstring("- bower_components\n"))
			})

			It("moves the requested cached directories", func() {
				Expect(c.Save()).To(Succeed())
				files, err := ioutil.ReadDir(filepath.Join(cacheDir, "node"))
				Expect(err).To(BeNil())

				Expect(len(files)).To(Equal(3))
				Expect(ioutil.ReadFile(filepath.Join(cacheDir, "node", "signature"))).To(Equal([]byte("1.1.1; 2.2.2; 3.3.3\n")))
				Expect(ioutil.ReadFile(filepath.Join(cacheDir, "node", ".cache", "yarn", "build2"))).To(Equal([]byte("build2")))
				Expect(ioutil.ReadFile(filepath.Join(cacheDir, "node", "bower_components", "build4"))).To(Equal([]byte("build4")))
			})
		})
	})
})
