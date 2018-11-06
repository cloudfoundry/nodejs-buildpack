package yarn_test

import (
	"bytes"
	"io/ioutil"
	"github.com/cloudfoundry/nodejs-buildpack/src/nodejs/yarn"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=yarn.go --destination=mocks_test.go --package=yarn_test

var _ = Describe("Yarn", func() {
	var (
		err         error
		buildDir    string
		cacheDir    string
		y           *yarn.Yarn
		logger      *libbuildpack.Logger
		buffer      *bytes.Buffer
		mockCtrl    *gomock.Controller
		mockCommand *MockCommand
		yarnCheck   error
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "nodejs-buildpack.build.")
		cacheDir, err = ioutil.TempDir("", "nodejs-buildpack.cache.")
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)

		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))

		mockCtrl = gomock.NewController(GinkgoT())
		mockCommand = NewMockCommand(mockCtrl)

		y = &yarn.Yarn{
			Log:     logger,
			Command: mockCommand,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())
	})

	Describe("Build", func() {
		var oldNodeHome string
		var yarnConfig map[string]string
		var yarnInstallArgs []string

		AfterEach(func() {
			Expect(os.Setenv("NODE_HOME", oldNodeHome)).To(Succeed())
		})
		BeforeEach(func() {
			oldNodeHome = os.Getenv("NODE_HOME")
			Expect(os.Setenv("NODE_HOME", "test_node_home")).To(Succeed())

			yarnConfig = map[string]string{}
			mockCommand.EXPECT().Run(gomock.Any()).Do(func(cmd *exec.Cmd) error {
				switch cmd.Args[1] {
				case "config":
					Expect(cmd.Args[0:3]).To(Equal([]string{"yarn", "config", "set"}))
					yarnConfig[cmd.Args[3]] = cmd.Args[4]
				default:
					yarnInstallArgs = cmd.Args
					Expect(cmd.Env).To(ContainElement("npm_config_nodedir=test_node_home"))
				}
				Expect(cmd.Dir).To(Equal(buildDir))
				return nil
			}).AnyTimes()
		})

		Context("has npm-packages-offline-cache", func() {
			JustBeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(buildDir, "npm-packages-offline-cache"), 0755)).To(Succeed())

				mockCommand.EXPECT().Execute(buildDir, ioutil.Discard, gomock.Any(), "yarn", []string{"check", "--offline"}).Return(yarnCheck)
			})

			It("tells the user it is running in offline mode", func() {
				Expect(y.Build(buildDir, cacheDir)).To(Succeed())
				Expect(buffer.String()).To(ContainSubstring("Installing node modules (yarn.lock)"))
				Expect(buffer.String()).To(ContainSubstring("Found yarn mirror directory " + filepath.Join(buildDir, "npm-packages-offline-cache")))
				Expect(buffer.String()).To(ContainSubstring("Running yarn in offline mode"))
			})

			It("runs yarn config", func() {
				Expect(y.Build(buildDir, cacheDir)).To(Succeed())
				Expect(yarnConfig).To(Equal(map[string]string{
					"yarn-offline-mirror":         filepath.Join(buildDir, "npm-packages-offline-cache"),
					"yarn-offline-mirror-pruning": "false",
				}))
			})

			It("runs yarn install with offline arguments and npm_config_nodedir", func() {
				Expect(y.Build(buildDir, cacheDir)).To(Succeed())
				Expect(yarnInstallArgs).To(Equal([]string{"yarn", "install", "--pure-lockfile", "--ignore-engines", "--cache-folder", filepath.Join(cacheDir, ".cache/yarn"), "--modules-folder", filepath.Join(buildDir, "node_modules"), "--offline"}))
			})

			Context("package.json matches yarn.lock", func() {
				BeforeEach(func() {
					yarnCheck = nil
				})

				It("reports the fact", func() {
					Expect(y.Build(buildDir, cacheDir)).To(Succeed())
					Expect(buffer.String()).To(ContainSubstring("yarn.lock and package.json match"))
				})
			})

			Context("package.json does not match yarn.lock", func() {
				BeforeEach(func() {
					yarnCheck = &exec.ExitError{}
				})

				It("warns the user", func() {
					Expect(y.Build(buildDir, cacheDir)).To(Succeed())
					Expect(buffer.String()).To(ContainSubstring("**WARNING** yarn.lock is outdated"))
				})
			})
		})

		Context("NO npm-packages-offline-cache directory", func() {
			JustBeforeEach(func() {
				mockCommand.EXPECT().Execute(buildDir, ioutil.Discard, gomock.Any(), "yarn", []string{"check"}).Return(yarnCheck)
			})

			It("tells the user it is running in online mode", func() {
				Expect(y.Build(buildDir, cacheDir)).To(Succeed())
				Expect(buffer.String()).To(ContainSubstring("Installing node modules (yarn.lock)"))
				Expect(buffer.String()).To(ContainSubstring("Running yarn in online mode"))
				Expect(buffer.String()).To(ContainSubstring("To run yarn in offline mode, see: https://yarnpkg.com/blog/2016/11/24/offline-mirror"))
			})

			It("runs yarn config", func() {
				Expect(y.Build(buildDir, cacheDir)).To(Succeed())
				Expect(yarnConfig).To(Equal(map[string]string{
					"yarn-offline-mirror":         filepath.Join(cacheDir, "npm-packages-offline-cache"),
					"yarn-offline-mirror-pruning": "true",
				}))
			})

			It("runs yarn install", func() {
				Expect(y.Build(buildDir, cacheDir)).To(Succeed())
				Expect(yarnInstallArgs).To(Equal([]string{"yarn", "install", "--pure-lockfile", "--ignore-engines", "--cache-folder", filepath.Join(cacheDir, ".cache/yarn"), "--modules-folder", filepath.Join(buildDir, "node_modules")}))
			})

			Context("package.json matches yarn.lock", func() {
				BeforeEach(func() {
					yarnCheck = nil
				})

				It("reports the fact", func() {
					Expect(y.Build(buildDir, cacheDir)).To(Succeed())
					Expect(buffer.String()).To(ContainSubstring("yarn.lock and package.json match"))
				})
			})

			Context("package.json does not match yarn.lock", func() {
				BeforeEach(func() {
					yarnCheck = &exec.ExitError{}
				})

				It("warns the user", func() {
					Expect(y.Build(buildDir, cacheDir)).To(Succeed())
					Expect(buffer.String()).To(ContainSubstring("**WARNING** yarn.lock is outdated"))
				})
			})
		})
	})
})
