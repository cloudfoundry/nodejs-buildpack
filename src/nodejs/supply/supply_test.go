package supply_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	"github.com/cloudfoundry/nodejs-buildpack/src/nodejs/supply"
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
		cacheDir        string
		depsDir         string
		depsIdx         string
		depDir          string
		supplier        *supply.Supplier
		logger          *libbuildpack.Logger
		buffer          *bytes.Buffer
		mockCtrl        *gomock.Controller
		mockYarn        *MockYarn
		mockNPM         *MockNPM
		mockManifest    *MockManifest
		mockInstaller   *MockInstaller
		mockCommand     *MockCommand
		installNode     func(libbuildpack.Dependency, string)
		installOnlyYarn func(string, string)
	)

	BeforeEach(func() {
		depsDir, err = os.MkdirTemp("", "nodejs-buildpack.deps.")
		Expect(err).To(BeNil())
		cacheDir, err = os.MkdirTemp("", "nodejs-buildpack.cache.")
		Expect(err).To(BeNil())

		buildDir, err = os.MkdirTemp("", "nodejs-buildpack.build.")
		Expect(err).To(BeNil())

		depsIdx = "14"
		depDir = filepath.Join(depsDir, depsIdx)

		err = os.MkdirAll(depDir, 0755)
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)

		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))

		mockCtrl = gomock.NewController(GinkgoT())
		mockManifest = NewMockManifest(mockCtrl)
		mockInstaller = NewMockInstaller(mockCtrl)
		mockCommand = NewMockCommand(mockCtrl)
		mockYarn = NewMockYarn(mockCtrl)
		mockNPM = NewMockNPM(mockCtrl)

		installNode = func(dep libbuildpack.Dependency, nodeDir string) {
			err := os.MkdirAll(filepath.Join(nodeDir, "bin"), 0755)
			Expect(err).To(BeNil())

			err = os.WriteFile(filepath.Join(nodeDir, "bin", "node"), []byte("node exe"), 0644)
			Expect(err).To(BeNil())

			err = os.WriteFile(filepath.Join(nodeDir, "bin", "npm"), []byte("npm exe"), 0644)
			Expect(err).To(BeNil())
		}

		installOnlyYarn = func(_ string, yarnDir string) {
			err := os.MkdirAll(filepath.Join(yarnDir, "bin"), 0755)
			Expect(err).To(BeNil())

			err = os.WriteFile(filepath.Join(yarnDir, "bin", "yarn"), []byte("yarn exe"), 0644)
			Expect(err).To(BeNil())

			err = os.WriteFile(filepath.Join(yarnDir, "bin", "yarnpkg"), []byte("yarnpkg exe"), 0644)
			Expect(err).To(BeNil())
		}

		args := []string{buildDir, cacheDir, depsDir, depsIdx}
		stager := libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})

		supplier = &supply.Supplier{
			Stager:    stager,
			Yarn:      mockYarn,
			NPM:       mockNPM,
			Log:       logger,
			Manifest:  mockManifest,
			Installer: mockInstaller,
			Command:   mockCommand,
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
				os.WriteFile(filepath.Join(buildDir, "package.json"), []byte(packageJSON), 0644)
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

					Expect(supplier.PackageJSONNodeVersion).To(Equal("node-y"))
					Expect(supplier.YarnVersion).To(Equal("*"))
					Expect(supplier.NPMVersion).To(Equal("npm-x"))
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

					Expect(supplier.NodeVersion).To(Equal(""))
					Expect(supplier.YarnVersion).To(Equal(""))
					Expect(supplier.NPMVersion).To(Equal(""))
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

					Expect(supplier.NodeVersion).To(Equal(""))
					Expect(supplier.YarnVersion).To(Equal(""))
					Expect(supplier.NPMVersion).To(Equal(""))
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

	Describe("Load .nvmrc contents", func() {
		Context("digits", func() {
			It("will trim and transform nvmrc to appropriate semver for Masterminds semver library", func() {
				nvmrcFile := filepath.Join(buildDir, ".nvmrc")
				defer os.Remove(nvmrcFile)

				testCases := [][]string{
					{"10", "10.*.*"},
					{"10.2", "10.2.*"},
					{"v10", "10.*.*"},
					{"10.2.3", "10.2.3"},
					{"v10.2.3", "10.2.3"},
				}

				for _, testCase := range testCases {
					Expect(os.WriteFile(nvmrcFile, []byte(testCase[0]), 0777)).To(Succeed())
					Expect(supplier.LoadNvmrc()).To(Succeed())
					Expect(supplier.NvmrcNodeVersion).To(Equal(testCase[1]), fmt.Sprintf("failed for test case %s : %s", testCase[0], testCase[1]))
				}
			})
		})

		Context("lts/something", func() {
			It("will read and trim lts versions to appropriate semver for Masterminds semver library", func() {
				nvmrcFile := filepath.Join(buildDir, ".nvmrc")
				defer os.Remove(nvmrcFile)

				testCases := [][]string{
					{"lts/hydrogen", "18.*.*"},
					{"lts/*", "18.*.*"},
				}

				for _, testCase := range testCases {
					Expect(os.WriteFile(nvmrcFile, []byte(testCase[0]), 0777)).To(Succeed())
					Expect(supplier.LoadNvmrc()).To(Succeed())
					Expect(supplier.NvmrcNodeVersion).To(Equal(testCase[1]), fmt.Sprintf("failed for test case %s : %s", testCase[0], testCase[1]))
				}
			})
		})

		Context("node", func() {
			It("should read and trim lts versions", func() {
				nvmrcFile := filepath.Join(buildDir, ".nvmrc")
				defer os.Remove(nvmrcFile)
				Expect(os.WriteFile(nvmrcFile, []byte("node"), 0777)).To(Succeed())
				Expect(supplier.LoadNvmrc()).To(Succeed())
				Expect(supplier.NvmrcNodeVersion).To(Equal("*"), fmt.Sprintf("failed for test case %s : %s", "node", "*"))
			})
		})
	})

	Describe("WarnNodeEngine", func() {
		Context("node version not specified", func() {
			It("warns that nvmrc version will be ignored in favor of package.json", func() {
				supplier.NvmrcNodeVersion = "13.*.*"
				supplier.PackageJSONNodeVersion = "13.*.*"
				supplier.WarnNodeEngine()
				Expect(buffer.String()).To(BeEmpty())
			})

			It("warns that different nvmrc version will be ignored in favor of package.json", func() {
				supplier.NvmrcNodeVersion = "13.*.*"
				supplier.PackageJSONNodeVersion = "14.*.*"
				supplier.WarnNodeEngine()
				Expect(buffer.String()).To(ContainSubstring("**WARNING** Node version in .nvmrc ignored in favor of 'engines' field in package.json"))
			})

			It("warns that node version hasn't been set", func() {
				supplier.NvmrcNodeVersion = ""
				supplier.PackageJSONNodeVersion = ""
				supplier.WarnNodeEngine()
				Expect(buffer.String()).To(ContainSubstring("**WARNING** Node version not specified in package.json or .nvmrc. See: http://docs.cloudfoundry.org/buildpacks/node/node-tips.html"))
			})
		})

		Context("node version is set to node in nvmrc", func() {
			It("warns that latest node version is being used", func() {
				supplier.NvmrcNodeVersion = "node"
				supplier.WarnNodeEngine()
				Expect(buffer.String()).To(ContainSubstring("**WARNING** .nvmrc specified latest node version, this will be selected from versions available in manifest.yml"))
				Expect(buffer.String()).To(ContainSubstring("**WARNING** Using the node version specified in your .nvmrc See: http://docs.cloudfoundry.org/buildpacks/node/node-tips.html"))

			})
		})

		Context("node version is set to lts in nvmrc", func() {
			It("warns that latest lts version is being used", func() {
				supplier.NvmrcNodeVersion = "lts/*"
				supplier.WarnNodeEngine()
				Expect(buffer.String()).To(ContainSubstring("**WARNING** .nvmrc specified an lts version, this will be selected from versions available in manifest.yml"))
			})
		})

		Context("node version is *", func() {
			It("warns that the node semver is dangerous", func() {
				supplier.PackageJSONNodeVersion = "*"
				supplier.WarnNodeEngine()
				Expect(buffer.String()).To(ContainSubstring("**WARNING** Dangerous semver range (*) in engines.node. See: http://docs.cloudfoundry.org/buildpacks/node/node-tips.html"))
			})
		})

		Context("node version is >x", func() {
			It("warns that the node semver is dangerous", func() {
				supplier.PackageJSONNodeVersion = ">5"
				supplier.WarnNodeEngine()
				Expect(buffer.String()).To(ContainSubstring("**WARNING** Dangerous semver range (>) in engines.node. See: http://docs.cloudfoundry.org/buildpacks/node/node-tips.html"))
			})
		})

		Context("node version is 'safe' semver", func() {
			It("does not log anything", func() {
				supplier.PackageJSONNodeVersion = "~>10"
				supplier.WarnNodeEngine()
				Expect(buffer.String()).To(Equal(""))
			})
		})

	})

	Describe("When nvmrc is present", func() {
		var (
			dep      libbuildpack.Dependency
			versions []string
		)

		BeforeEach(func() {
			dep = libbuildpack.Dependency{Name: "node", Version: "6.10.2"}
			mockManifest.EXPECT().DefaultVersion("node").Return(dep, nil).AnyTimes()
			versions = []string{
				"4.0.0", "4.0.1", "4.2.3",
				"6.0.0", "6.0.2", "6.2.3",
				"8.0.1", "8.0.3", "8.2.3",
				"10.0.0", "10.0.4", "10.2.3",
				"11.0.0", "11.0.5", "11.2.3",
			}
			mockManifest.EXPECT().AllDependencyVersions("node").Return(versions).AnyTimes()
		})

		Context("nvmrc is present and engines field in package.json is present", func() {
			It("selects the version from the engines field in packages.json", func() {
				supplier.PackageJSONNodeVersion = "10.0.0"
				supplier.NvmrcNodeVersion = "10.2.3"
				Expect(supplier.ChooseNodeVersion()).To(Succeed())
				Expect(supplier.NodeVersion).To(Equal("10.0.0"))
			})
		})

		Context("nvmrc is present and engines field in package.json is missing", func() {
			It("selects the version in nvmrc", func() {
				supplier.PackageJSONNodeVersion = ""
				supplier.NvmrcNodeVersion = "10.2.3"
				Expect(supplier.ChooseNodeVersion()).To(Succeed())
				Expect(supplier.NodeVersion).To(Equal("10.2.3"))
			})
		})

		Context("nvmrc is missing and engines field in package.json is present", func() {
			It("selects version from engines in package.json", func() {
				supplier.PackageJSONNodeVersion = "11.2.3"
				supplier.NvmrcNodeVersion = ""
				Expect(supplier.ChooseNodeVersion()).To(Succeed())
				Expect(supplier.NodeVersion).To(Equal("11.2.3"))
			})
		})

		Context("package.json engines field and nvmrc are both specified", func() {
			It("selects version from package.json engines field", func() {
				supplier.NvmrcNodeVersion = "8.*.*"
				supplier.PackageJSONNodeVersion = ">8.0.3"
				err := supplier.ChooseNodeVersion()
				Expect(err).ToNot(HaveOccurred())
				Expect(supplier.NodeVersion).To(Equal("11.2.3"))
			})
		})
	})

	Describe(".nvmrc validation", func() {

		AfterEach(func() {
			Expect(os.Remove(filepath.Join(buildDir, ".nvmrc"))).To(Succeed())
		})

		Context("given valid .nvmrc", func() {
			It("validate should succeed", func() {
				validVersions := []string{"11.4", "node", "lts/*", "lts/hydrogen", "10", "10.1.1"}
				for _, version := range validVersions {
					Expect(os.WriteFile(filepath.Join(buildDir, ".nvmrc"), []byte(version), 0777)).To(Succeed())
					Expect(supplier.LoadNvmrc()).To(Succeed())
				}
			})
		})

		Context("given an invalid .nvmrc", func() {
			It("validate should be fail", func() {
				invalidVersions := []string{"11.4.x", "invalid", "~1.1.2", ">11.0", "< 11.4.2", "^1.2.3", "11.*.*", "10.1.x", "10.1.X", "lts/invalidname"}
				for _, version := range invalidVersions {
					Expect(os.WriteFile(filepath.Join(buildDir, ".nvmrc"), []byte(version), 0777)).To(Succeed())
					Expect(supplier.LoadNvmrc()).ToNot(Succeed())
				}
			})
		})
	})

	Describe("InstallNode", func() {
		var nodeDir string

		BeforeEach(func() {
			nodeDir = filepath.Join(depDir, "node")
		})

		AfterEach(func() {
			Expect(os.RemoveAll(nodeDir)).To(Succeed())
		})

		Context("node version use semver", func() {
			BeforeEach(func() {
				versions := []string{"6.10.2", "6.11.1", "4.8.2", "4.8.3", "7.0.0"}
				mockManifest.EXPECT().AllDependencyVersions("node").Return(versions)
				mockManifest.EXPECT().DefaultVersion("node").Return(libbuildpack.Dependency{"node", "0.0.0"}, nil).AnyTimes()
			})

			It("installs the correct version from the manifest", func() {
				dep := libbuildpack.Dependency{Name: "node", Version: "4.8.3"}
				mockInstaller.EXPECT().InstallDependency(dep, nodeDir).Do(installNode).Return(nil)

				supplier.PackageJSONNodeVersion = "~>4"
				err = supplier.ChooseNodeVersion()
				Expect(err).To(BeNil())
				err = supplier.InstallNode()
				Expect(err).To(BeNil())
			})

			It("handles '>=6.11.1 <7.0'", func() {
				dep := libbuildpack.Dependency{Name: "node", Version: "6.11.1"}
				mockInstaller.EXPECT().InstallDependency(dep, nodeDir).Do(installNode).Return(nil)

				supplier.PackageJSONNodeVersion = ">=6.11.1 <7.0.0"
				err = supplier.ChooseNodeVersion()
				Expect(err).To(BeNil())
				err = supplier.InstallNode()
				Expect(err).To(BeNil())
			})

			It("handles '>=6.11.1, <7.0'", func() {
				dep := libbuildpack.Dependency{Name: "node", Version: "6.11.1"}
				mockInstaller.EXPECT().InstallDependency(dep, nodeDir).Do(installNode).Return(nil)

				supplier.PackageJSONNodeVersion = ">=6.11.1, <7.0"
				err = supplier.ChooseNodeVersion()
				Expect(err).To(BeNil())
				err = supplier.InstallNode()
				Expect(err).To(BeNil())
			})

			It("creates a symlink in <depDir>/bin", func() {
				dep := libbuildpack.Dependency{Name: "node", Version: "6.10.2"}
				mockInstaller.EXPECT().InstallDependency(dep, nodeDir).Do(installNode).Return(nil)
				supplier.PackageJSONNodeVersion = "6.10.*"
				err = supplier.ChooseNodeVersion()
				Expect(err).To(BeNil())
				err = supplier.InstallNode()
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
				mockManifest.EXPECT().AllDependencyVersions(gomock.Any())
				mockInstaller.EXPECT().InstallDependency(dep, nodeDir).Do(installNode).Return(nil)

				supplier.NodeVersion = ""

				err = supplier.ChooseNodeVersion()
				Expect(err).To(BeNil())

				err = supplier.InstallNode()
				Expect(err).To(BeNil())
			})
		})

		Context("Installing Node >=18", func() {
			It("SSL_CERT_DIR env variable is set", func() {
				dep := libbuildpack.Dependency{Name: "node", Version: "18.0.0"}
				mockManifest.EXPECT().DefaultVersion("node").Return(dep, nil)
				mockManifest.EXPECT().AllDependencyVersions(gomock.Any())
				mockInstaller.EXPECT().InstallDependency(dep, nodeDir).Do(installNode).Return(nil)

				supplier.NodeVersion = ""

				err = supplier.ChooseNodeVersion()
				Expect(err).To(BeNil())

				err = supplier.InstallNode()
				Expect(err).To(BeNil())

				_, SSLEnvironmentVariable := os.LookupEnv("SSL_CERT_DIR")
				Expect(SSLEnvironmentVariable).To(BeTrue())
				os.Unsetenv("SSL_CERT_DIR")
			})
		})

		Context("Installing Node <18", func() {
			It("SSL_CERT_DIR env variable is not set", func() {
				dep := libbuildpack.Dependency{Name: "node", Version: "16.0.0"}
				mockManifest.EXPECT().DefaultVersion("node").Return(dep, nil)
				mockManifest.EXPECT().AllDependencyVersions(gomock.Any())
				mockInstaller.EXPECT().InstallDependency(dep, nodeDir).Do(installNode).Return(nil)

				supplier.NodeVersion = ""

				err = supplier.ChooseNodeVersion()
				Expect(err).To(BeNil())

				err = supplier.InstallNode()
				Expect(err).To(BeNil())

				_, SSLEnvironmentVariable := os.LookupEnv("SSL_CERT_DIR")
				Expect(SSLEnvironmentVariable).To(BeFalse())
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
				mockInstaller.EXPECT().InstallOnlyVersion("yarn", yarnInstallDir).Do(installOnlyYarn).Return(nil)

				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "yarn", "--version").Do(func(_ string, buffer io.Writer, _ io.Writer, _ string, _ string) {
					buffer.Write([]byte("0.32.5\n"))
				}).Return(nil)
			})

			It("installs the only version in the manifest", func() {
				supplier.YarnVersion = ""

				err = supplier.InstallYarn()
				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("Installed yarn 0.32.5"))
			})

			It("creates a symlink in <depDir>/bin", func() {
				supplier.YarnVersion = ""
				err = supplier.InstallYarn()
				Expect(err).To(BeNil())

				link, err := os.Readlink(filepath.Join(depsDir, depsIdx, "bin", "yarn"))
				Expect(err).To(BeNil())
				Expect(link).To(Equal("../yarn/bin/yarn"))

				link, err = os.Readlink(filepath.Join(depsDir, depsIdx, "bin", "yarnpkg"))
				Expect(err).To(BeNil())
				Expect(link).To(Equal("../yarn/bin/yarnpkg"))
			})
		})

		Context("requested yarn version is in manifest", func() {
			BeforeEach(func() {
				versions := []string{"0.32.5"}
				mockManifest.EXPECT().AllDependencyVersions("yarn").Return(versions)
				mockInstaller.EXPECT().InstallOnlyVersion("yarn", yarnInstallDir).Do(installOnlyYarn).Return(nil)

				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "yarn", "--version").Do(func(_ string, buffer io.Writer, _ io.Writer, _ string, _ string) {
					buffer.Write([]byte("0.32.5\n"))
				}).Return(nil)
			})

			It("installs the correct version from the manifest", func() {
				supplier.YarnVersion = "0.32.x"
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
				supplier.YarnVersion = "1.0.x"
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
					supplier.NPMVersion = "1.2.3"

					err = supplier.InstallNPM()
					Expect(err).To(BeNil())

					Expect(buffer.String()).To(ContainSubstring("npm 1.2.3 already installed with node"))
				})
			})
			Context("requested version has minor .x and is already installed", func() {
				It("Uses the version of npm packaged with node", func() {
					supplier.NPMVersion = "1.2.x"

					err = supplier.InstallNPM()
					Expect(err).To(BeNil())

					Expect(buffer.String()).To(ContainSubstring("npm 1.2.3 already installed with node"))
				})
			})

			It("installs the requested npm version using packaged npm", func() {
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(),
					"npm", "install", "--unsafe-perm", "--quiet", "-g", "npm@4.5.6",
					"--userconfig", filepath.Join(buildDir, ".npmrc")).Return(nil)

				supplier.NPMVersion = "4.5.6"
				err = supplier.InstallNPM()
				Expect(err).To(BeNil())

				Expect(buffer.String()).To(ContainSubstring("Downloading and installing npm 4.5.6 (replacing version 1.2.3)..."))
			})
		})
	})

	Describe("ReadPackageJSON", func() {

		Context("package.json has prebuild script", func() {
			BeforeEach(func() {
				packageJSON := `
{
  "scripts" : {
		"script": "script",
		"heroku-prebuild": "makestuff",
		"thing": "thing"
	}
}
`
				Expect(os.WriteFile(filepath.Join(buildDir, "package.json"), []byte(packageJSON), 0644)).To(Succeed())
			})

			It("sets PreBuild", func() {
				Expect(supplier.ReadPackageJSON()).To(Succeed())
				Expect(supplier.PreBuild).To(Equal("makestuff"))
			})
		})

		Context("package.json has postbuild script", func() {
			BeforeEach(func() {
				packageJSON := `
{
  "scripts" : {
		"script": "script",
		"heroku-postbuild": "logstuff",
		"thing": "thing"
	}
}
`
				Expect(os.WriteFile(filepath.Join(buildDir, "package.json"), []byte(packageJSON), 0644)).To(Succeed())
			})

			It("sets PostBuild", func() {
				Expect(supplier.ReadPackageJSON()).To(Succeed())
				Expect(supplier.PostBuild).To(Equal("logstuff"))
			})
		})

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
				Expect(os.WriteFile(filepath.Join(buildDir, "package.json"), []byte(packageJSON), 0644)).To(Succeed())
			})

			It("sets StartScript", func() {
				Expect(supplier.ReadPackageJSON()).To(Succeed())
				Expect(supplier.StartScript).To(Equal("start-my-app"))
			})
		})

		Context("package.json does not exist", func() {
			It("warns user", func() {
				Expect(supplier.ReadPackageJSON()).To(Succeed())
				Expect(buffer.String()).To(ContainSubstring("**WARNING** No package.json found"))
			})
		})

		Context("yarn.lock exists", func() {
			BeforeEach(func() {
				Expect(os.WriteFile(filepath.Join(buildDir, "yarn.lock"), []byte("{}"), 0644)).To(Succeed())
			})
			It("sets UseYarn to true", func() {
				Expect(supplier.ReadPackageJSON()).To(Succeed())
				Expect(supplier.UseYarn).To(BeTrue())
			})
		})

		Context("yarn.lock does not exist", func() {
			It("sets UseYarn to false", func() {
				Expect(supplier.ReadPackageJSON()).To(Succeed())
				Expect(supplier.UseYarn).To(BeFalse())
			})
		})

		Context("node_modules exists", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(buildDir, "node_modules"), 0755)).To(Succeed())
			})
			It("sets NPMRebuild to true", func() {
				Expect(supplier.ReadPackageJSON()).To(Succeed())
				Expect(supplier.IsVendored).To(BeTrue())
			})
		})

		Context("node_modules does not exist", func() {
			It("sets NPMRebuild to false", func() {
				Expect(supplier.ReadPackageJSON()).To(Succeed())
				Expect(supplier.IsVendored).To(BeFalse())
			})
		})

		Context("dev dependencies exist", func() {
			BeforeEach(func() {
				packageJSON := `
{
	"devDependencies": {
    "logger": "^0.0.1"
  }
}
`
				Expect(os.WriteFile(filepath.Join(buildDir, "package.json"), []byte(packageJSON), 0644)).To(Succeed())
			})
			It("sets HasDevDependencies to true", func() {
				Expect(supplier.ReadPackageJSON()).To(Succeed())
				Expect(supplier.HasDevDependencies).To(BeTrue())
			})
		})

		Context("dev dependencies do not exist", func() {
			It("sets HasDevDependencies to false", func() {
				Expect(supplier.ReadPackageJSON()).To(Succeed())
				Expect(supplier.HasDevDependencies).To(BeFalse())
			})
		})
	})

	Describe("TipVendorDependencies", func() {
		Context("node_modules exists and has subdirectories", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(buildDir, "node_modules", "exciting_module"), 0755)).To(BeNil())
			})

			It("does not log anything", func() {
				Expect(supplier.TipVendorDependencies()).To(BeNil())
				Expect(buffer.String()).To(Equal(""))
			})
		})

		Context("node_modules exists and has NO subdirectories", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(buildDir, "node_modules"), 0755)).To(BeNil())
				Expect(os.WriteFile(filepath.Join(buildDir, "node_modules", "a_file"), []byte("content"), 0644)).To(BeNil())
			})

			It("logs a pro tip", func() {
				Expect(supplier.TipVendorDependencies()).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("PRO TIP: It is recommended to vendor the application's Node.js dependencies"))
				Expect(buffer.String()).To(ContainSubstring("http://docs.cloudfoundry.org/buildpacks/node/index.html#vendoring"))
			})
		})

		Context("node_modules does not exist", func() {
			It("logs a pro tip", func() {
				Expect(supplier.TipVendorDependencies()).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("PRO TIP: It is recommended to vendor the application's Node.js dependencies"))
				Expect(buffer.String()).To(ContainSubstring("http://docs.cloudfoundry.org/buildpacks/node/index.html#vendoring"))
			})
		})
	})

	Describe("ListNodeConfig", func() {
		DescribeTable("outputs relevant env vars",
			func(key string, value string, expected string) {
				supplier.ListNodeConfig([]string{fmt.Sprintf("%s=%s", key, value)})
				Expect(buffer.String()).To(Equal(expected))
			},

			Entry("NPM_CONFIG_", "NPM_CONFIG_THING", "someval", "       NPM_CONFIG_THING=someval\n"),
			Entry("YARN_", "YARN_KEY", "aval", "       YARN_KEY=aval\n"),
			Entry("NODE_", "NODE_EXCITING", "newval", "       NODE_EXCITING=newval\n"),
			Entry("NOT_RELEVANT", "NOT_RELEVANT", "anything", ""),
		)

		It("warns about NODE_ENV override", func() {
			supplier.ListNodeConfig([]string{"NPM_CONFIG_PRODUCTION=true", "NODE_ENV=development"})
			Expect(buffer.String()).To(ContainSubstring("npm scripts will see NODE_ENV=production (not 'development')"))
			Expect(buffer.String()).To(ContainSubstring("https://docs.npmjs.com/misc/config#production"))
		})
	})

	Describe("WarnUntrackedDependencies", func() {
		var (
			logfile  *os.File
			contents string
		)

		JustBeforeEach(func() {
			logfile, err = os.CreateTemp("", "nodejs-buildpack.log")
			Expect(err).To(BeNil())

			_, err = logfile.Write([]byte(contents))
			Expect(err).To(BeNil())
			Expect(logfile.Sync()).To(Succeed())

			supplier.Logfile = logfile
			Expect(supplier.WarnUntrackedDependencies()).To(Succeed())
		})

		AfterEach(func() {
			Expect(logfile.Close()).To(Succeed())
			Expect(os.Remove(logfile.Name())).To(Succeed())
		})

		Context("gulp not found", func() {
			BeforeEach(func() {
				contents = "stuff\ngulp: not found\nstuff\n"
			})
			It("warns the user", func() {
				Expect(buffer.String()).To(ContainSubstring("Gulp may not be tracked in package.json"))
			})
		})

		Context("gulp command not found", func() {
			BeforeEach(func() {
				contents = "stuff\ngulp: command not found\nstuff\n"
			})
			It("warns the user", func() {
				Expect(buffer.String()).To(ContainSubstring("Gulp may not be tracked in package.json"))
			})
		})
		Context("bower not found", func() {
			BeforeEach(func() {
				contents = "stuff\nbower: not found\nstuff\n"
			})
			It("warns the user", func() {
				Expect(buffer.String()).To(ContainSubstring("Bower may not be tracked in package.json"))
			})
		})

		Context("bower command not found", func() {
			BeforeEach(func() {
				contents = "stuff\nbower: command not found\nstuff\n"
			})
			It("warns the user", func() {
				Expect(buffer.String()).To(ContainSubstring("Bower may not be tracked in package.json"))
			})
		})

		Context("grunt not found", func() {
			BeforeEach(func() {
				contents = "stuff\ngrunt: not found\nstuff\n"
			})
			It("warns the user", func() {
				Expect(buffer.String()).To(ContainSubstring("Grunt may not be tracked in package.json"))
			})
		})

		Context("grunt command not found", func() {
			BeforeEach(func() {
				contents = "stuff\ngrunt: command not found\nstuff\n"
			})
			It("warns the user", func() {
				Expect(buffer.String()).To(ContainSubstring("Grunt may not be tracked in package.json"))
			})
		})

		Context("no 'not found' errors", func() {
			BeforeEach(func() {
				contents = "stuff\ngood command\nstuff\n"
			})
			It("does not warn the user", func() {
				Expect(buffer.String()).To(BeEmpty())
			})
		})
	})

	Describe("WarnMissingDevDeps", func() {
		var (
			logfile  *os.File
			contents string
		)

		JustBeforeEach(func() {
			logfile, err = os.CreateTemp("", "nodejs-buildpack.log")
			Expect(err).To(BeNil())

			_, err = logfile.Write([]byte(contents))
			Expect(err).To(BeNil())
			Expect(logfile.Sync()).To(Succeed())

			supplier.Logfile = logfile
			Expect(supplier.WarnMissingDevDeps()).To(Succeed())
		})

		AfterEach(func() {
			Expect(logfile.Close()).To(Succeed())
			Expect(os.Remove(logfile.Name())).To(Succeed())
		})

		Context("cannot find module", func() {
			BeforeEach(func() {
				contents = "stuff\ncannot find module\nstuff\n"
			})

			It("warns the user", func() {
				Expect(buffer.String()).To(ContainSubstring("A module may be missing from 'dependencies' in package.json"))
			})

			Context("$NPM_CONFIG_PRODUCTION == true", func() {
				BeforeEach(func() {
					Expect(os.Setenv("NPM_CONFIG_PRODUCTION", "true")).To(Succeed())
				})
				AfterEach(func() {
					Expect(os.Unsetenv("NPM_CONFIG_PRODUCTION")).To(Succeed())
				})

				Context("package.json has dev dependencies", func() {
					BeforeEach(func() {
						supplier.HasDevDependencies = true
					})

					It("warns the user", func() {
						Expect(buffer.String()).To(ContainSubstring("This module may be specified in 'devDependencies' instead of 'dependencies'"))
						Expect(buffer.String()).To(ContainSubstring("See: https://devcenter.heroku.com/articles/nodejs-support#devdependencies"))
					})
				})

				Context("package.json does not gave dev dependencies", func() {
					BeforeEach(func() {
						supplier.HasDevDependencies = false
					})

					It("does not warn the user", func() {
						Expect(buffer.String()).ToNot(ContainSubstring("devDependencies"))
						Expect(buffer.String()).ToNot(ContainSubstring("devdependencies"))
					})
				})
			})
		})

		Context("no missing module errors", func() {
			BeforeEach(func() {
				contents = "stuff\nstuff\n"
			})
			It("does not warn the user", func() {
				Expect(buffer.String()).To(BeEmpty())
			})
		})
	})

	Describe("OverrideCacheFromApp", func() {
		Context("cache dir has deprecated bower_components directory", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(cacheDir, "bower_components", "subdir"), 0755)).To(Succeed())
			})
			It("deletes the deprecated directory", func() {
				Expect(supplier.OverrideCacheFromApp()).To(Succeed())
				Expect(filepath.Join(cacheDir, "bower_components", "subdir")).ToNot(BeADirectory())
			})
		})
		Context("app has '.npm' directory", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(buildDir, ".npm", "subdir"), 0755)).To(Succeed())
			})
			It("copies directory to cache", func() {
				Expect(supplier.OverrideCacheFromApp()).To(Succeed())
				Expect(filepath.Join(buildDir, ".npm", "subdir")).To(BeADirectory())
				Expect(filepath.Join(cacheDir, ".npm", "subdir")).To(BeADirectory())
			})
		})
		Context("app has '.cache/yarn' directory", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(buildDir, ".cache", "yarn", "subdir"), 0755)).To(Succeed())
			})
			It("copies directory to cache", func() {
				Expect(supplier.OverrideCacheFromApp()).To(Succeed())
				Expect(filepath.Join(buildDir, ".cache", "yarn", "subdir")).To(BeADirectory())
				Expect(filepath.Join(cacheDir, ".cache", "yarn", "subdir")).To(BeADirectory())
			})
		})
	})

	Describe("BuildDependencies", func() {
		Context("using yarn", func() {
			BeforeEach(func() {
				supplier.UseYarn = true
				mockYarn.EXPECT().Build(buildDir, cacheDir).DoAndReturn(func(string, string) error {
					Expect(os.MkdirAll(filepath.Join(buildDir, "node_modules"), 0755)).To(Succeed())
					return nil
				})
			})

			It("runs yarn build", func() {
				Expect(supplier.BuildDependencies()).To(Succeed())
			})

			It("runs the prebuild script, when prebuild is specified", func() {
				supplier.PreBuild = "prescriptive"
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "yarn", "run", "heroku-prebuild")
				Expect(supplier.BuildDependencies()).To(Succeed())
				Expect(buffer.String()).To(ContainSubstring("Running heroku-prebuild (yarn)"))
			})

			It("runs the postbuild script, when postbuild is specified", func() {
				supplier.PostBuild = "descriptive"
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "yarn", "run", "heroku-postbuild")
				Expect(supplier.BuildDependencies()).To(Succeed())
				Expect(buffer.String()).To(ContainSubstring("Running heroku-postbuild (yarn)"))
			})
		})

		Describe("using npm", func() {
			BeforeEach(func() {
				supplier.UseYarn = false
			})

			It("runs npm build when node_modules does not exist", func() {
				supplier.IsVendored = false
				mockNPM.EXPECT().Build(gomock.Any(), gomock.Any()).DoAndReturn(func(string, string) error {
					Expect(os.MkdirAll(filepath.Join(buildDir, "node_modules"), 0755)).To(Succeed())
					return nil
				})
				Expect(supplier.BuildDependencies()).To(Succeed())
			})

			It("runs npm rebuild, when node_modules exists", func() {
				supplier.IsVendored = true
				mockNPM.EXPECT().Rebuild(buildDir).Return(nil)
				Expect(supplier.BuildDependencies()).To(Succeed())
			})

			It("runs the prebuild script, when prebuild is specified", func() {
				supplier.PreBuild = "prescriptive"
				mockNPM.EXPECT().Build(gomock.Any(), gomock.Any()).DoAndReturn(func(string, string) error {
					Expect(os.MkdirAll(filepath.Join(buildDir, "node_modules"), 0755)).To(Succeed())
					return nil
				})
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "npm", "run", "heroku-prebuild", "--if-present")
				Expect(supplier.BuildDependencies()).To(Succeed())
				Expect(buffer.String()).To(ContainSubstring("Running heroku-prebuild (npm)"))
			})

			It("runs the postbuild script, when postbuild is specified", func() {
				supplier.PostBuild = "descriptive"
				mockNPM.EXPECT().Build(buildDir, cacheDir).DoAndReturn(func(string, string) error {
					Expect(os.MkdirAll(filepath.Join(buildDir, "node_modules"), 0755)).To(Succeed())
					return nil
				})
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "npm", "run", "heroku-postbuild", "--if-present")
				Expect(supplier.BuildDependencies()).To(Succeed())
				Expect(buffer.String()).To(ContainSubstring("Running heroku-postbuild (npm)"))
			})
		})
	})

	Describe("MoveDependencyArtifacts", func() {
		Context("when app is already vendored", func() {
			BeforeEach(func() {
				supplier.IsVendored = true
				Expect(os.MkdirAll(filepath.Join(buildDir, "node_modules", "a", "b"), 0755)).To(Succeed())
				Expect(supplier.MoveDependencyArtifacts()).To(Succeed())
			})

			It("does NOT moves node_modules into deps directory after installing them", func() {
				Expect(filepath.Join(buildDir, "node_modules", "a", "b")).To(BeADirectory())
				Expect(filepath.Join(depDir, "node_modules")).ToNot(BeADirectory())
			})

			It("does NOT set NODE_PATH environment file", func() {
				Expect(filepath.Join(depDir, "env", "NODE_PATH")).ToNot(BeAnExistingFile())
			})

			It("does NOT sets NODE_PATH environment variable", func() {
				_, nodePathSet := os.LookupEnv("NODE_PATH")
				Expect(nodePathSet).To(BeFalse())
			})
		})

		Context("when app is NOT vendored", func() {
			BeforeEach(func() {
				supplier.IsVendored = false
				Expect(os.MkdirAll(filepath.Join(buildDir, "node_modules", "a", "b"), 0755)).To(Succeed())
				Expect(supplier.MoveDependencyArtifacts()).To(Succeed())
			})

			AfterEach(func() {
				Expect(os.Unsetenv("NODE_PATH")).To(Succeed())
			})

			It("moves node_modules and .yarnrc into deps directory after installing them", func() {
				Expect(filepath.Join(buildDir, "node_modules")).ToNot(BeADirectory())
				Expect(filepath.Join(depDir, "node_modules", "a", "b")).To(BeADirectory())
			})

			It("sets NODE_PATH environment file", func() {
				Expect(os.ReadFile(filepath.Join(depDir, "env", "NODE_PATH"))).To(Equal([]byte(filepath.Join(depDir, "node_modules"))))
			})

			It("sets NODE_PATH environment variable", func() {
				Expect(os.Getenv("NODE_PATH")).To(Equal(filepath.Join(depDir, "node_modules")))
			})

			It("does not error if no node_modules are installed", func() {
				Expect(os.RemoveAll(filepath.Join(buildDir, "node_modules"))).To(Succeed())

				Expect(supplier.MoveDependencyArtifacts()).To(Succeed())
				Expect(filepath.Join(buildDir, "node_modules")).ToNot(BeADirectory())
			})
		})

	})

	Describe("ListDependencies", func() {
		var oldNodeVerbose string

		BeforeEach(func() {
			oldNodeVerbose = os.Getenv("NODE_VERBOSE")
		})

		AfterEach(func() {
			Expect(os.Setenv("NODE_VERBOSE", oldNodeVerbose)).To(Succeed())
		})

		Context("package manager is yarn", func() {
			BeforeEach(func() {
				supplier.UseYarn = true
			})

			Context("NODE_VERBOSE is true", func() {
				BeforeEach(func() {
					Expect(os.Setenv("NODE_VERBOSE", "true")).To(Succeed())
				})

				It("lists the installed packages", func() {
					mockCommand.EXPECT().Execute(buildDir, gomock.Any(), io.Discard, "npm", "ls", "--depth=0").Return(nil).Do(func(_ string, outBuf io.Writer, _ io.Writer, _ string, _ ...string) {
						_, err := outBuf.Write([]byte("some-dep" + supply.UnmetDependency))
						Expect(err).NotTo(HaveOccurred())
					})

					deps, err := supplier.ListDependencies()
					Expect(err).NotTo(HaveOccurred())
					Expect(deps).To(ContainSubstring(supply.UnmetDependency))
					Expect(buffer.String()).To(ContainSubstring(supply.UnmetDependency))
				})
			})

			Context("NODE_VERBOSE is not true", func() {
				It("does not list the installed packages", func() {
					mockCommand.EXPECT().Execute(buildDir, gomock.Any(), io.Discard, "npm", "ls", "--depth=0").Return(nil).Do(func(_ string, outBuf io.Writer, _ io.Writer, _ string, _ ...string) {
						_, err := outBuf.Write([]byte("some-dep" + supply.UnmetDependency))
						buffer.WriteString("some-dep")
						Expect(err).NotTo(HaveOccurred())
					})
					deps, err := supplier.ListDependencies()
					Expect(err).NotTo(HaveOccurred())
					Expect(deps).To(ContainSubstring(supply.UnmetDependency))
					Expect(buffer.String()).NotTo(ContainSubstring(supply.UnmetDependency))
				})
			})
		})
	})

	Describe("WarnUnmetDependencies", func() {
		var (
			contents string
		)

		JustBeforeEach(func() {
			supplier.WarnUnmetDependencies(contents)
		})

		Context("package manager is yarn", func() {
			BeforeEach(func() {
				supplier.UseYarn = true
			})
			Context("there are unmet dependencies", func() {
				BeforeEach(func() {
					contents = "stuff\nsome unmet dependency stuff\nstuff\n"
				})
				It("warns the user", func() {
					Expect(buffer.String()).To(ContainSubstring("Unmet dependencies don't fail yarn install but may cause runtime issues"))
					Expect(buffer.String()).To(ContainSubstring("See: https://github.com/npm/npm/issues/7494"))
				})
			})
			Context("there are unmet peer dependencies", func() {
				BeforeEach(func() {
					contents = "stuff\nsome unmet peer dependency stuff\nstuff\n"
				})
				It("warns the user", func() {
					Expect(buffer.String()).To(ContainSubstring("Unmet dependencies don't fail yarn install but may cause runtime issues"))
					Expect(buffer.String()).To(ContainSubstring("See: https://github.com/npm/npm/issues/7494"))
				})
			})
			Context("there are NO unmet peer dependencies", func() {
				BeforeEach(func() {
					contents = "stuff\nsome stuff\nstuff\n"
				})
				It("warns the user", func() {
					Expect(buffer.String()).To(BeEmpty())
				})
			})
		})
		Context("package manager is npm", func() {
			Context("there are unmet dependencies", func() {
				BeforeEach(func() {
					contents = "stuff\nsome unmet dependency stuff\nstuff\n"
				})
				It("warns the user", func() {
					Expect(buffer.String()).To(ContainSubstring("Unmet dependencies don't fail npm install but may cause runtime issues"))
					Expect(buffer.String()).To(ContainSubstring("See: https://github.com/npm/npm/issues/7494"))
				})
			})
			Context("there are unmet peer dependencies", func() {
				BeforeEach(func() {
					contents = "stuff\nsome unmet peer dependency stuff\nstuff\n"
				})
				It("warns the user", func() {
					Expect(buffer.String()).To(ContainSubstring("Unmet dependencies don't fail npm install but may cause runtime issues"))
					Expect(buffer.String()).To(ContainSubstring("See: https://github.com/npm/npm/issues/7494"))
				})
			})
			Context("there are UNMET PEER DEPENDENCIES", func() {
				BeforeEach(func() {
					contents = "stuff\nsome UNMET PEER DEPENDENCY stuff\nstuff\n"
				})
				It("warns the user", func() {
					Expect(buffer.String()).To(ContainSubstring("Unmet dependencies don't fail npm install but may cause runtime issues"))
					Expect(buffer.String()).To(ContainSubstring("See: https://github.com/npm/npm/issues/7494"))
				})
			})
			Context("there are NO unmet peer dependencies", func() {
				BeforeEach(func() {
					contents = "stuff\nsome stuff\nstuff\n"
				})
				It("warns the user", func() {
					Expect(buffer.String()).To(BeEmpty())
				})
			})
		})
	})

	Describe("CreateDefaultEnv for Node <18", func() {
		BeforeEach(func() {
			supplier.NodeVersion = "16.0.0"
		})

		It("writes an env file for NODE_HOME", func() {
			err = supplier.CreateDefaultEnv()
			Expect(err).To(BeNil())

			contents, err := os.ReadFile(filepath.Join(depsDir, depsIdx, "env", "NODE_HOME"))
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
			Entry("WEB_MEMORY", "WEB_MEMORY", "a value"),
			Entry("WEB_CONCURRENCY", "WEB_CONCURRENCY", "another value"),
		)

		DescribeTable("environment with default was not set",
			func(key string, expected string) {
				oldValue := os.Getenv(key)
				defer os.Setenv(key, oldValue)
				Expect(os.Unsetenv(key)).To(BeNil())

				Expect(supplier.CreateDefaultEnv()).To(BeNil())
				contents, err := os.ReadFile(filepath.Join(depsDir, depsIdx, "env", key))
				Expect(err).To(BeNil())

				Expect(string(contents)).To(Equal(expected))
			},
			Entry("NODE_ENV", "NODE_ENV", "production"),
			Entry("NPM_CONFIG_PRODUCTION", "NPM_CONFIG_PRODUCTION", "true"),
			Entry("NPM_CONFIG_LOGLEVEL", "NPM_CONFIG_LOGLEVEL", "error"),
			Entry("NODE_MODULES_CACHE", "NODE_MODULES_CACHE", "true"),
			Entry("NODE_VERBOSE", "NODE_VERBOSE", "false"),
			Entry("WEB_MEMORY", "WEB_MEMORY", "512"),
			Entry("WEB_CONCURRENCY", "WEB_CONCURRENCY", "1"),
		)

		It("writes profile.d script for runtime", func() {
			err = supplier.CreateDefaultEnv()
			Expect(err).To(BeNil())

			contents, err := os.ReadFile(filepath.Join(depsDir, depsIdx, "profile.d", "node.sh"))
			Expect(err).To(BeNil())

			Expect(string(contents)).To(ContainSubstring("export NODE_HOME=" + filepath.Join("$DEPS_DIR", depsIdx, "node")))
			Expect(string(contents)).To(ContainSubstring("export NODE_ENV=${NODE_ENV:-production}"))
			nodePathString := `
if [ ! -d "$HOME/node_modules" ]; then
	export NODE_PATH=${NODE_PATH:-"$DEPS_DIR/14/node_modules"}
	ln -s "$DEPS_DIR/14/node_modules" "$HOME/node_modules"
else
	export NODE_PATH=${NODE_PATH:-"$HOME/node_modules"}
fi
export PATH=$PATH:"$HOME/bin":$NODE_PATH/.bin
`
			Expect(string(contents)).To(ContainSubstring(nodePathString))
			Expect(string(contents)).To(Not(ContainSubstring("export SSL_CERT_DIR=${SSL_CERT_DIR:-/etc/ssl/certs}")))
		})
	})

	Describe("CreateDefaultEnv for Node >=18", func() {
		BeforeEach(func() {
			supplier.NodeVersion = "18.0.0"
		})

		It("writes an env file for NODE_HOME", func() {
			err = supplier.CreateDefaultEnv()
			Expect(err).To(BeNil())

			contents, err := os.ReadFile(filepath.Join(depsDir, depsIdx, "env", "NODE_HOME"))
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
			Entry("WEB_MEMORY", "WEB_MEMORY", "a value"),
			Entry("WEB_CONCURRENCY", "WEB_CONCURRENCY", "another value"),
		)

		DescribeTable("environment with default was not set",
			func(key string, expected string) {
				oldValue := os.Getenv(key)
				defer os.Setenv(key, oldValue)
				Expect(os.Unsetenv(key)).To(BeNil())

				Expect(supplier.CreateDefaultEnv()).To(BeNil())
				contents, err := os.ReadFile(filepath.Join(depsDir, depsIdx, "env", key))
				Expect(err).To(BeNil())

				Expect(string(contents)).To(Equal(expected))
			},
			Entry("NODE_ENV", "NODE_ENV", "production"),
			Entry("NPM_CONFIG_PRODUCTION", "NPM_CONFIG_PRODUCTION", "true"),
			Entry("NPM_CONFIG_LOGLEVEL", "NPM_CONFIG_LOGLEVEL", "error"),
			Entry("NODE_MODULES_CACHE", "NODE_MODULES_CACHE", "true"),
			Entry("NODE_VERBOSE", "NODE_VERBOSE", "false"),
			Entry("WEB_MEMORY", "WEB_MEMORY", "512"),
			Entry("WEB_CONCURRENCY", "WEB_CONCURRENCY", "1"),
		)

		It("writes profile.d script for runtime", func() {
			err = supplier.CreateDefaultEnv()
			Expect(err).To(BeNil())

			contents, err := os.ReadFile(filepath.Join(depsDir, depsIdx, "profile.d", "node.sh"))
			Expect(err).To(BeNil())

			Expect(string(contents)).To(ContainSubstring("export NODE_HOME=" + filepath.Join("$DEPS_DIR", depsIdx, "node")))
			Expect(string(contents)).To(ContainSubstring("export NODE_ENV=${NODE_ENV:-production}"))
			nodePathString := `
if [ ! -d "$HOME/node_modules" ]; then
	export NODE_PATH=${NODE_PATH:-"$DEPS_DIR/14/node_modules"}
	ln -s "$DEPS_DIR/14/node_modules" "$HOME/node_modules"
else
	export NODE_PATH=${NODE_PATH:-"$HOME/node_modules"}
fi
export PATH=$PATH:"$HOME/bin":$NODE_PATH/.bin
`
			Expect(string(contents)).To(ContainSubstring(nodePathString))
			Expect(string(contents)).To(ContainSubstring("export SSL_CERT_DIR=${SSL_CERT_DIR:-/etc/ssl/certs}"))
		})
	})
})
