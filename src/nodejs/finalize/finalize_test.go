package finalize_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"nodejs/finalize"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
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
		mockCommand  *MockCommand
		mockYarn     *MockYarn
		mockNPM      *MockNPM
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
		mockCommand = NewMockCommand(mockCtrl)
		mockYarn = NewMockYarn(mockCtrl)
		mockNPM = NewMockNPM(mockCtrl)
		mockManifest = NewMockManifest(mockCtrl)

		args := []string{buildDir, "", depsDir, depsIdx}
		stager := libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})

		finalizer = &finalize.Finalizer{
			Stager:   stager,
			Yarn:     mockYarn,
			NPM:      mockNPM,
			Manifest: mockManifest,
			Command:  mockCommand,
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

	Describe("TipVendorDependencies", func() {
		Context("node_modules exists and has subdirectories", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(buildDir, "node_modules", "exciting_module"), 0755)).To(BeNil())
			})

			It("does not log anything", func() {
				Expect(finalizer.TipVendorDependencies()).To(BeNil())
				Expect(buffer.String()).To(Equal(""))
			})
		})

		Context("node_modules exists and has NO subdirectories", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(buildDir, "node_modules"), 0755)).To(BeNil())
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "node_modules", "a_file"), []byte("content"), 0644)).To(BeNil())
			})

			It("logs a pro tip", func() {
				Expect(finalizer.TipVendorDependencies()).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("PRO TIP: It is recommended to vendor the application's Node.js dependencies"))
				Expect(buffer.String()).To(ContainSubstring("http://docs.cloudfoundry.org/buildpacks/node/index.html#vendoring"))
			})
		})

		Context("node_modules does not exist", func() {
			It("logs a pro tip", func() {
				Expect(finalizer.TipVendorDependencies()).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("PRO TIP: It is recommended to vendor the application's Node.js dependencies"))
				Expect(buffer.String()).To(ContainSubstring("http://docs.cloudfoundry.org/buildpacks/node/index.html#vendoring"))
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
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "package.json"), []byte(packageJSON), 0644)).To(Succeed())
			})

			It("sets PreBuild", func() {
				Expect(finalizer.ReadPackageJSON()).To(Succeed())
				Expect(finalizer.PreBuild).To(Equal("makestuff"))
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
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "package.json"), []byte(packageJSON), 0644)).To(Succeed())
			})

			It("sets PostBuild", func() {
				Expect(finalizer.ReadPackageJSON()).To(Succeed())
				Expect(finalizer.PostBuild).To(Equal("logstuff"))
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
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "package.json"), []byte(packageJSON), 0644)).To(Succeed())
			})

			It("sets StartScript", func() {
				Expect(finalizer.ReadPackageJSON()).To(Succeed())
				Expect(finalizer.StartScript).To(Equal("start-my-app"))
			})
		})

		Context("package.json does not exist", func() {
			It("warns user", func() {
				Expect(finalizer.ReadPackageJSON()).To(Succeed())
				Expect(buffer.String()).To(ContainSubstring("**WARNING** No package.json found"))
			})
		})

		Context("yarn.lock exists", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "yarn.lock"), []byte("{}"), 0644)).To(Succeed())
			})
			It("sets UseYarn to true", func() {
				Expect(finalizer.ReadPackageJSON()).To(Succeed())
				Expect(finalizer.UseYarn).To(BeTrue())
			})
		})

		Context("yarn.lock does not exist", func() {
			It("sets UseYarn to false", func() {
				Expect(finalizer.ReadPackageJSON()).To(Succeed())
				Expect(finalizer.UseYarn).To(BeFalse())
			})
		})

		Context("node_modules exists", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(buildDir, "node_modules"), 0755)).To(Succeed())
			})
			It("sets NPMRebuild to true", func() {
				Expect(finalizer.ReadPackageJSON()).To(Succeed())
				Expect(finalizer.NPMRebuild).To(BeTrue())
			})
		})

		Context("node_modules does not exist", func() {
			It("sets NPMRebuild to false", func() {
				Expect(finalizer.ReadPackageJSON()).To(Succeed())
				Expect(finalizer.NPMRebuild).To(BeFalse())
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
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "package.json"), []byte(packageJSON), 0644)).To(Succeed())
			})
			It("sets HasDevDependencies to true", func() {
				Expect(finalizer.ReadPackageJSON()).To(Succeed())
				Expect(finalizer.HasDevDependencies).To(BeTrue())
			})
		})

		Context("dev dependencies do not exist", func() {
			It("sets HasDevDependencies to false", func() {
				Expect(finalizer.ReadPackageJSON()).To(Succeed())
				Expect(finalizer.HasDevDependencies).To(BeFalse())
			})
		})
	})

	Describe("ListNodeConfig", func() {
		DescribeTable("outputs relevant env vars",
			func(key string, value string, expected string) {
				finalizer.ListNodeConfig([]string{fmt.Sprintf("%s=%s", key, value)})
				Expect(buffer.String()).To(Equal(expected))
			},

			Entry("NPM_CONFIG_", "NPM_CONFIG_THING", "someval", "       NPM_CONFIG_THING=someval\n"),
			Entry("YARN_", "YARN_KEY", "aval", "       YARN_KEY=aval\n"),
			Entry("NODE_", "NODE_EXCITING", "newval", "       NODE_EXCITING=newval\n"),
			Entry("NOT_RELEVANT", "NOT_RELEVANT", "anything", ""),
		)

		It("warns about NODE_ENV override", func() {
			finalizer.ListNodeConfig([]string{"NPM_CONFIG_PRODUCTION=true", "NODE_ENV=development"})
			Expect(buffer.String()).To(ContainSubstring("npm scripts will see NODE_ENV=production (not 'development')"))
			Expect(buffer.String()).To(ContainSubstring("https://docs.npmjs.com/misc/config#production"))
		})
	})

	Describe("BuildDependencies", func() {
		Context("yarn.lock exists", func() {
			BeforeEach(func() {
				finalizer.UseYarn = true
				mockYarn.EXPECT().Build().Return(nil)
			})

			It("runs yarn install", func() {
				Expect(finalizer.BuildDependencies()).To(Succeed())
			})

			Context("prebuild is specified", func() {
				BeforeEach(func() {
					finalizer.PreBuild = "prescriptive"
				})

				It("runs the prebuild script", func() {
					mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "yarn", "run", "prescriptive")
					Expect(finalizer.BuildDependencies()).To(Succeed())
					Expect(buffer.String()).To(ContainSubstring("Running prescriptive (yarn)"))
				})
			})

			Context("postbuild is specified", func() {
				BeforeEach(func() {
					finalizer.PostBuild = "descriptive"
				})

				It("runs the postbuild script", func() {
					mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "yarn", "run", "descriptive")
					Expect(finalizer.BuildDependencies()).To(Succeed())
					Expect(buffer.String()).To(ContainSubstring("Running descriptive (yarn)"))
				})
			})
		})

		Context("yarn.lock does not exist", func() {
			It("runs npm install", func() {
				mockNPM.EXPECT().Build().Return(nil)
				Expect(finalizer.BuildDependencies()).To(Succeed())
			})

			Context("prebuild is specified", func() {
				BeforeEach(func() {
					mockNPM.EXPECT().Build().Return(nil)
					finalizer.PreBuild = "prescriptive"
				})

				It("runs the prebuild script", func() {
					mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "npm", "run", "prescriptive", "--if-present")
					Expect(finalizer.BuildDependencies()).To(Succeed())
					Expect(buffer.String()).To(ContainSubstring("Running prescriptive (npm)"))
				})
			})

			Context("npm rebuild is specified", func() {
				BeforeEach(func() {
					mockNPM.EXPECT().Rebuild().Return(nil)
					finalizer.NPMRebuild = true
				})

				It("runs npm rebuild ", func() {
					Expect(finalizer.BuildDependencies()).To(Succeed())
					Expect(buffer.String()).To(ContainSubstring("Prebuild detected (node_modules already exists)"))
				})
			})

			Context("postbuild is specified", func() {
				BeforeEach(func() {
					mockNPM.EXPECT().Build().Return(nil)
					finalizer.PostBuild = "descriptive"
				})

				It("runs the postbuild script", func() {
					mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "npm", "run", "descriptive", "--if-present")
					Expect(finalizer.BuildDependencies()).To(Succeed())
					Expect(buffer.String()).To(ContainSubstring("Running descriptive (npm)"))
				})
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
	})

	Describe("CopyProfileScripts", func() {
		var oldNodeVerbose string

		BeforeEach(func() {
			oldNodeVerbose = os.Getenv("NODE_VERBOSE")
		})

		AfterEach(func() {
			Expect(os.Setenv("NODE_VERBOSE", oldNodeVerbose)).To(Succeed())
		})

		Context("package manager is yarn", func() {
			BeforeEach(func() {
				finalizer.UseYarn = true
			})

			Context("NODE_VERBOSE is true", func() {
				BeforeEach(func() {
					Expect(os.Setenv("NODE_VERBOSE", "true")).To(Succeed())
				})

				It("lists the installed packages", func() {
					mockCommand.EXPECT().Execute(buildDir, gomock.Any(), ioutil.Discard, "yarn", "list", "--depth=0").Return(nil)
					finalizer.ListDependencies()
				})
			})

			Context("NODE_VERBOSE is not true", func() {
				It("does not list the installed packages", func() {
					finalizer.ListDependencies()
				})
			})
		})

		Context("package manager is npm", func() {
			BeforeEach(func() {
				finalizer.UseYarn = false
			})

			Context("NODE_VERBOSE is true", func() {
				BeforeEach(func() {
					Expect(os.Setenv("NODE_VERBOSE", "true")).To(Succeed())
				})

				It("lists the installed packages", func() {
					mockCommand.EXPECT().Execute(buildDir, gomock.Any(), ioutil.Discard, "npm", "ls", "--depth=0").Return(nil)
					finalizer.ListDependencies()
				})
			})

			Context("NODE_VERBOSE is not true", func() {
				It("does not list the installed packages", func() {
					finalizer.ListDependencies()
				})
			})
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

	Describe("WarnUnmetDependencies", func() {
		var (
			logfile  *os.File
			contents string
		)

		JustBeforeEach(func() {
			logfile, err = ioutil.TempFile("", "nodejs-buildpack.log")
			Expect(err).To(BeNil())

			_, err = logfile.Write([]byte(contents))
			Expect(err).To(BeNil())
			Expect(logfile.Sync()).To(Succeed())

			finalizer.Logfile = logfile
			Expect(finalizer.WarnUnmetDependencies()).To(Succeed())
		})

		AfterEach(func() {
			Expect(logfile.Close()).To(Succeed())
			Expect(os.Remove(logfile.Name())).To(Succeed())
		})

		Context("package manager is yarn", func() {
			BeforeEach(func() {
				finalizer.UseYarn = true
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

	Describe("WarnUntrackedDependencies", func() {
		var (
			logfile  *os.File
			contents string
		)

		JustBeforeEach(func() {
			logfile, err = ioutil.TempFile("", "nodejs-buildpack.log")
			Expect(err).To(BeNil())

			_, err = logfile.Write([]byte(contents))
			Expect(err).To(BeNil())
			Expect(logfile.Sync()).To(Succeed())

			finalizer.Logfile = logfile
			Expect(finalizer.WarnUntrackedDependencies()).To(Succeed())
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
			logfile, err = ioutil.TempFile("", "nodejs-buildpack.log")
			Expect(err).To(BeNil())

			_, err = logfile.Write([]byte(contents))
			Expect(err).To(BeNil())
			Expect(logfile.Sync()).To(Succeed())

			finalizer.Logfile = logfile
			Expect(finalizer.WarnMissingDevDeps()).To(Succeed())
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
						finalizer.HasDevDependencies = true
					})

					It("warns the user", func() {
						Expect(buffer.String()).To(ContainSubstring("This module may be specified in 'devDependencies' instead of 'dependencies'"))
						Expect(buffer.String()).To(ContainSubstring("See: https://devcenter.heroku.com/articles/nodejs-support#devdependencies"))
					})
				})

				Context("package.json does not gave dev dependencies", func() {
					BeforeEach(func() {
						finalizer.HasDevDependencies = false
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
})
