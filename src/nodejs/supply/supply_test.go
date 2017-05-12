package supply_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"nodejs/supply"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=supply.go --destination=mocks_test.go --package=supply_test

var _ = Describe("Supply", func() {
	var (
		err             error
		buildDir        string
		depsDir         string
		depsIdx         string
		depDir          string
		supplier        *supply.Supplier
		logger          *libbuildpack.Logger
		buffer          *bytes.Buffer
		mockCtrl        *gomock.Controller
		mockManifest    *MockManifest
		mockCommand     *MockCommand
		installNode     func(libbuildpack.Dependency, string)
		installOnlyYarn func(string, string)
	)

	BeforeEach(func() {
		depsDir, err = ioutil.TempDir("", "nodejs-buildpack.deps.")
		Expect(err).To(BeNil())

		buildDir, err = ioutil.TempDir("", "nodejs-buildpack.build.")
		Expect(err).To(BeNil())

		depsIdx = "14"
		depDir = filepath.Join(depsDir, depsIdx)

		err = os.MkdirAll(depDir, 0755)
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)

		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))

		mockCtrl = gomock.NewController(GinkgoT())
		mockManifest = NewMockManifest(mockCtrl)
		mockCommand = NewMockCommand(mockCtrl)

		installNode = func(dep libbuildpack.Dependency, nodeDir string) {
			subDir := fmt.Sprintf("node-v%s-linux-x64", dep.Version)
			err := os.MkdirAll(filepath.Join(nodeDir, subDir, "bin"), 0755)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(nodeDir, subDir, "bin", "node"), []byte("node exe"), 0644)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(nodeDir, subDir, "bin", "npm"), []byte("npm exe"), 0644)
			Expect(err).To(BeNil())
		}

		installOnlyYarn = func(_ string, yarnDir string) {
			err := os.MkdirAll(filepath.Join(yarnDir, "dist", "bin"), 0755)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(yarnDir, "dist", "bin", "yarn"), []byte("yarn exe"), 0644)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(yarnDir, "dist", "bin", "yarnpkg"), []byte("yarnpkg exe"), 0644)
			Expect(err).To(BeNil())
		}

		args := []string{buildDir, "", depsDir, depsIdx}
		stager := libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})

		supplier = &supply.Supplier{
			Stager:   stager,
			Log:      logger,
			Manifest: mockManifest,
			Command:  mockCommand,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(depsDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())
	})

	Describe("LoadPackageJSON", func() {
		var packageJSON string

		JustBeforeEach(func() {
			if packageJSON != "" {
				ioutil.WriteFile(filepath.Join(buildDir, "package.json"), []byte(packageJSON), 0644)
			}
		})

		Context("File is invalid JSON", func() {
			BeforeEach(func() {
				packageJSON = `not actually JSON`
			})

			It("returns an error", func() {
				err = supplier.LoadPackageJSON()
				Expect(err).NotTo(BeNil())
			})
		})

		Context("File is valid JSON", func() {
			Context("has an engines section", func() {
				BeforeEach(func() {
					packageJSON = `
{
  "name": "node",
  "version": "1.0.0",
  "main": "server.js",
  "author": "CF Buildpacks Team",
  "dependencies": {
    "logfmt": "~1.1.2",
    "express": "~4.0.0"
  },
  "engines" : {
		"yarn" : "*",
		"npm"  : "npm-x",
		"node" : "node-y",
		"something" : "3.2.1"
	}
}
`
				})

				It("loads the engines into the supplier", func() {
					err = supplier.LoadPackageJSON()
					Expect(err).To(BeNil())

					Expect(supplier.Node).To(Equal("node-y"))
					Expect(supplier.Yarn).To(Equal("*"))
					Expect(supplier.NPM).To(Equal("npm-x"))
				})

				It("logs the node and npm versions", func() {
					err = supplier.LoadPackageJSON()
					Expect(err).To(BeNil())

					Expect(buffer.String()).To(ContainSubstring("engines.node (package.json): node-y"))
					Expect(buffer.String()).To(ContainSubstring("engines.npm (package.json): npm-x"))
				})

				Context("the engines section contains iojs", func() {
					BeforeEach(func() {
						packageJSON = `
{
  "engines" : {
		"iojs" : "*"
	}
}
`
					})

					It("returns an error", func() {
						err = supplier.LoadPackageJSON()
						Expect(err).NotTo(BeNil())

						Expect(err.Error()).To(ContainSubstring("io.js not supported by this buildpack"))
					})
				})
			})

			Context("does not have an engines section", func() {
				BeforeEach(func() {
					packageJSON = `
{
  "name": "node",
  "version": "1.0.0",
  "main": "server.js",
  "author": "CF Buildpacks Team",
  "dependencies": {
    "logfmt": "~1.1.2",
    "express": "~4.0.0"
  }
}
`
				})

				It("loads the engine struct with empty strings", func() {
					err = supplier.LoadPackageJSON()
					Expect(err).To(BeNil())

					Expect(supplier.Node).To(Equal(""))
					Expect(supplier.Yarn).To(Equal(""))
					Expect(supplier.NPM).To(Equal(""))
				})

				It("logs that node and npm are not set", func() {
					err = supplier.LoadPackageJSON()
					Expect(err).To(BeNil())

					Expect(buffer.String()).To(ContainSubstring("engines.node (package.json): unspecified"))
					Expect(buffer.String()).To(ContainSubstring("engines.npm (package.json): unspecified (use default)"))
				})
			})

			Context("package.json does not exist", func() {
				BeforeEach(func() {
					packageJSON = ""
				})

				It("loads the engine struct with empty strings", func() {
					err = supplier.LoadPackageJSON()
					Expect(err).To(BeNil())

					Expect(supplier.Node).To(Equal(""))
					Expect(supplier.Yarn).To(Equal(""))
					Expect(supplier.NPM).To(Equal(""))
				})

				It("logs that node and npm are not set", func() {
					err = supplier.LoadPackageJSON()
					Expect(err).To(BeNil())

					Expect(buffer.String()).To(ContainSubstring("engines.node (package.json): unspecified"))
					Expect(buffer.String()).To(ContainSubstring("engines.npm (package.json): unspecified (use default)"))
				})
			})
		})
	})

	Describe("WarnNodeEngine", func() {
		Context("node version not specified", func() {
			It("warns that node version hasn't been set", func() {
				supplier.WarnNodeEngine()
				Expect(buffer.String()).To(ContainSubstring("**WARNING** Node version not specified in package.json. See: http://docs.cloudfoundry.org/buildpacks/node/node-tips.html"))
			})
		})

		Context("node version is *", func() {
			It("warns that the node semver is dangerous", func() {
				supplier.Node = "*"
				supplier.WarnNodeEngine()
				Expect(buffer.String()).To(ContainSubstring("**WARNING** Dangerous semver range (*) in engines.node. See: http://docs.cloudfoundry.org/buildpacks/node/node-tips.html"))
			})
		})

		Context("node version is >x", func() {
			It("warns that the node semver is dangerous", func() {
				supplier.Node = ">5"
				supplier.WarnNodeEngine()
				Expect(buffer.String()).To(ContainSubstring("**WARNING** Dangerous semver range (>) in engines.node. See: http://docs.cloudfoundry.org/buildpacks/node/node-tips.html"))
			})
		})

		Context("node version is 'safe' semver", func() {
			It("does not log anything", func() {
				supplier.Node = "~>6"
				supplier.WarnNodeEngine()
				Expect(buffer.String()).To(Equal(""))
			})
		})
	})

	Describe("InstallNode", func() {
		var nodeInstallDir string
		var nodeTmpDir string

		BeforeEach(func() {
			nodeInstallDir = filepath.Join(depsDir, depsIdx, "node")
			nodeTmpDir, err = ioutil.TempDir("", "nodejs-buildpack.temp")
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(nodeTmpDir)).To(Succeed())
		})

		Context("node version use semver", func() {
			BeforeEach(func() {
				versions := []string{"6.10.2", "6.11.1", "4.8.2", "4.8.3"}
				mockManifest.EXPECT().AllDependencyVersions("node").Return(versions)
			})

			It("installs the correct version from the manifest", func() {
				dep := libbuildpack.Dependency{Name: "node", Version: "4.8.3"}
				mockManifest.EXPECT().InstallDependency(dep, nodeTmpDir).Do(installNode).Return(nil)

				supplier.Node = "~>4"
				err = supplier.InstallNode(nodeTmpDir)
				Expect(err).To(BeNil())
			})

			It("creates a symlink in <depDir>/bin", func() {
				dep := libbuildpack.Dependency{Name: "node", Version: "6.10.2"}
				mockManifest.EXPECT().InstallDependency(dep, nodeTmpDir).Do(installNode).Return(nil)

				supplier.Node = "6.10.*"
				err = supplier.InstallNode(nodeTmpDir)
				Expect(err).To(BeNil())

				link, err := os.Readlink(filepath.Join(depsDir, depsIdx, "bin", "node"))
				Expect(err).To(BeNil())

				Expect(link).To(Equal("../node/bin/node"))

				link, err = os.Readlink(filepath.Join(depsDir, depsIdx, "bin", "npm"))
				Expect(err).To(BeNil())

				Expect(link).To(Equal("../node/bin/npm"))
			})
		})

		Context("node version is unset", func() {
			It("installs the default version from the manifest", func() {
				dep := libbuildpack.Dependency{Name: "node", Version: "6.10.2"}
				mockManifest.EXPECT().DefaultVersion("node").Return(dep, nil)
				mockManifest.EXPECT().InstallDependency(dep, nodeTmpDir).Do(installNode).Return(nil)

				supplier.Node = ""

				err = supplier.InstallNode(nodeTmpDir)
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("InstallYarn", func() {
		var yarnInstallDir string

		BeforeEach(func() {
			yarnInstallDir = filepath.Join(depsDir, depsIdx, "yarn")
		})

		Context("yarn version is unset", func() {
			BeforeEach(func() {
				mockManifest.EXPECT().InstallOnlyVersion("yarn", yarnInstallDir).Do(installOnlyYarn).Return(nil)

				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "yarn", "--version").Do(func(_ string, buffer io.Writer, _ io.Writer, _ string, _ string) {
					buffer.Write([]byte("0.32.5\n"))
				}).Return(nil)
			})

			It("installs the only version in the manifest", func() {
				supplier.Yarn = ""

				err = supplier.InstallYarn()
				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("Installed yarn 0.32.5"))
			})

			It("creates a symlink in <depDir>/bin", func() {
				supplier.Yarn = ""
				err = supplier.InstallYarn()
				Expect(err).To(BeNil())

				link, err := os.Readlink(filepath.Join(depsDir, depsIdx, "bin", "yarn"))
				Expect(err).To(BeNil())
				Expect(link).To(Equal("../yarn/dist/bin/yarn"))

				link, err = os.Readlink(filepath.Join(depsDir, depsIdx, "bin", "yarnpkg"))
				Expect(err).To(BeNil())
				Expect(link).To(Equal("../yarn/dist/bin/yarnpkg"))
			})
		})

		Context("requested yarn version is in manifest", func() {
			BeforeEach(func() {
				versions := []string{"0.32.5"}
				mockManifest.EXPECT().AllDependencyVersions("yarn").Return(versions)
				mockManifest.EXPECT().InstallOnlyVersion("yarn", yarnInstallDir).Do(installOnlyYarn).Return(nil)

				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "yarn", "--version").Do(func(_ string, buffer io.Writer, _ io.Writer, _ string, _ string) {
					buffer.Write([]byte("0.32.5\n"))
				}).Return(nil)
			})

			It("installs the correct version from the manifest", func() {
				supplier.Yarn = "0.32.x"
				err = supplier.InstallYarn()
				Expect(err).To(BeNil())

				Expect(buffer.String()).To(ContainSubstring("Installed yarn 0.32.5"))
			})
		})

		Context("requested yarn version is not in manifest", func() {
			BeforeEach(func() {
				versions := []string{"0.32.5"}
				mockManifest.EXPECT().AllDependencyVersions("yarn").Return(versions)
			})

			It("returns an error", func() {
				supplier.Yarn = "1.0.x"
				err = supplier.InstallYarn()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal("package.json requested 1.0.x, buildpack only includes yarn version 0.32.5"))
			})
		})
	})

	Describe("InstallNPM", func() {
		BeforeEach(func() {
			mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "npm", "--version").Do(func(_ string, buffer io.Writer, _ io.Writer, _ string, _ string) {
				buffer.Write([]byte("1.2.3\n"))
			}).Return(nil)
		})

		Context("npm version is not set", func() {
			It("uses the version of npm packaged with node", func() {
				err = supplier.InstallNPM()
				Expect(err).To(BeNil())

				Expect(buffer.String()).To(ContainSubstring("Using default npm version: 1.2.3"))
			})
		})

		Context("npm version is set", func() {
			Context("requested version is already installed", func() {
				It("Uses the version of npm packaged with node", func() {
					supplier.NPM = "1.2.3"

					err = supplier.InstallNPM()
					Expect(err).To(BeNil())

					Expect(buffer.String()).To(ContainSubstring("npm 1.2.3 already installed with node"))
				})
			})

			It("installs the requested npm version using packaged npm", func() {
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "npm", "install", "--unsafe-perm", "--quiet", "-g", "npm@4.5.6").Return(nil)
				supplier.NPM = "4.5.6"

				err = supplier.InstallNPM()
				Expect(err).To(BeNil())

				Expect(buffer.String()).To(ContainSubstring("Downloading and installing npm 4.5.6 (replacing version 1.2.3)..."))
			})
		})
	})

	Describe("CreateDefaultEnv", func() {
		It("writes an env file for NODE_HOME", func() {
			err = supplier.CreateDefaultEnv()
			Expect(err).To(BeNil())

			contents, err := ioutil.ReadFile(filepath.Join(depsDir, depsIdx, "env", "NODE_HOME"))
			Expect(err).To(BeNil())

			Expect(string(contents)).To(Equal(filepath.Join(depsDir, depsIdx, "node")))
		})

		DescribeTable("environment with default has a value",
			func(key string, value string) {
				oldValue := os.Getenv(key)
				defer os.Setenv(key, oldValue)

				Expect(os.Setenv(key, value)).To(BeNil())
				Expect(supplier.CreateDefaultEnv()).To(BeNil())
				Expect(filepath.Join(depsDir, depsIdx, "env", key)).NotTo(BeAnExistingFile())
			},
			Entry("NODE_ENV", "NODE_ENV", "anything"),
			Entry("NPM_CONFIG_PRODUCTION", "NPM_CONFIG_PRODUCTION", "some value"),
			Entry("NPM_CONFIG_LOGLEVEL", "NPM_CONFIG_LOGLEVEL", "everything"),
			Entry("NODE_MODULES_CACHE", "NODE_MODULES_CACHE", "false"),
			Entry("NODE_VERBOSE", "NODE_VERBOSE", "many words"),
		)

		DescribeTable("environment with default was not set",
			func(key string, expected string) {
				oldValue := os.Getenv(key)
				defer os.Setenv(key, oldValue)
				Expect(os.Unsetenv(key)).To(BeNil())

				Expect(supplier.CreateDefaultEnv()).To(BeNil())
				contents, err := ioutil.ReadFile(filepath.Join(depsDir, depsIdx, "env", key))
				Expect(err).To(BeNil())

				Expect(string(contents)).To(Equal(expected))
			},
			Entry("NODE_ENV", "NODE_ENV", "production"),
			Entry("NPM_CONFIG_PRODUCTION", "NPM_CONFIG_PRODUCTION", "true"),
			Entry("NPM_CONFIG_LOGLEVEL", "NPM_CONFIG_LOGLEVEL", "error"),
			Entry("NODE_MODULES_CACHE", "NODE_MODULES_CACHE", "true"),
			Entry("NODE_VERBOSE", "NODE_VERBOSE", "false"),
		)

		It("writes profile.d script for runtime", func() {
			err = supplier.CreateDefaultEnv()
			Expect(err).To(BeNil())

			contents, err := ioutil.ReadFile(filepath.Join(depsDir, depsIdx, "profile.d", "node.sh"))
			Expect(err).To(BeNil())

			Expect(string(contents)).To(ContainSubstring("export NODE_HOME=" + filepath.Join("$DEPS_DIR", depsIdx, "node")))
			Expect(string(contents)).To(ContainSubstring("export NODE_ENV=${NODE_ENV:-production}"))
		})
	})
})
