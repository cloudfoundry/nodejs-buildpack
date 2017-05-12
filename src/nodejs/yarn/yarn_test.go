package yarn_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"nodejs/yarn"
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
		y           *yarn.Yarn
		logger      *libbuildpack.Logger
		buffer      *bytes.Buffer
		mockCtrl    *gomock.Controller
		mockCommand *MockCommand
		yarnCheck   error
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "nodejs-buildpack.build.")
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)

		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))

		mockCtrl = gomock.NewController(GinkgoT())
		mockCommand = NewMockCommand(mockCtrl)

		y = &yarn.Yarn{
			BuildDir: buildDir,
			Log:      logger,
			Command:  mockCommand,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())
	})

	Describe("Build", func() {
		var oldNodeHome string

		BeforeEach(func() {
			oldNodeHome = os.Getenv("NODE_HOME")
			Expect(os.Setenv("NODE_HOME", "test_node_home")).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.Setenv("NODE_HOME", oldNodeHome)).To(Succeed())
		})

		Context("has npm-packages-offline-cache", func() {
			JustBeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(buildDir, "npm-packages-offline-cache"), 0755)).To(Succeed())

				gomock.InOrder(
					mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "yarn", "config", "set", "yarn-offline-mirror", filepath.Join(buildDir, "npm-packages-offline-cache")).Return(nil),
					mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "yarn", "install", "--pure-lockfile", "--ignore-engines", "--cache-folder", filepath.Join(buildDir, ".cache/yarn"), "--offline").Do(
						func(_ string, _, _ io.Writer, _, _, _, _, _, _, _ string) {
							Expect(os.Getenv("npm_config_nodedir")).To(Equal("test_node_home"))
						}).Return(nil),
					mockCommand.EXPECT().Execute(buildDir, ioutil.Discard, gomock.Any(), "yarn", "check", "--offline").Return(yarnCheck),
				)
			})

			It("tells the user it is running in offline mode", func() {
				Expect(y.Build()).To(Succeed())
				Expect(buffer.String()).To(ContainSubstring("Installing node modules (yarn.lock)"))
				Expect(buffer.String()).To(ContainSubstring("Found yarn mirror directory " + filepath.Join(buildDir, "npm-packages-offline-cache")))
				Expect(buffer.String()).To(ContainSubstring("Running yarn in offline mode"))
			})

			It("runs yarn config", func() {
				Expect(y.Build()).To(Succeed())
			})

			It("runs yarn install with npm_config_nodedir", func() {
				Expect(y.Build()).To(Succeed())
			})

			Context("package.json matches yarn.lock", func() {
				BeforeEach(func() {
					yarnCheck = nil
				})

				It("reports the fact", func() {
					Expect(y.Build()).To(Succeed())
					Expect(buffer.String()).To(ContainSubstring("yarn.lock and package.json match"))
				})
			})

			Context("package.json does not match yarn.lock", func() {
				BeforeEach(func() {
					yarnCheck = &exec.ExitError{}
				})

				It("warns the user", func() {
					Expect(y.Build()).To(Succeed())
					Expect(buffer.String()).To(ContainSubstring("**WARNING** yarn.lock is outdated"))
				})
			})
		})

		Context("NO npm-packages-offline-cache directory", func() {
			JustBeforeEach(func() {
				gomock.InOrder(
					mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "yarn", "install", "--pure-lockfile", "--ignore-engines", "--cache-folder", filepath.Join(buildDir, ".cache/yarn")).Do(
						func(_ string, _, _ io.Writer, _, _, _, _, _, _ string) {
							Expect(os.Getenv("npm_config_nodedir")).To(Equal("test_node_home"))
						}).Return(nil),
					mockCommand.EXPECT().Execute(buildDir, ioutil.Discard, gomock.Any(), "yarn", "check").Return(yarnCheck),
				)
			})

			It("tells the user it is running in online mode", func() {
				Expect(y.Build()).To(Succeed())
				Expect(buffer.String()).To(ContainSubstring("Installing node modules (yarn.lock)"))
				Expect(buffer.String()).To(ContainSubstring("Running yarn in online mode"))
				Expect(buffer.String()).To(ContainSubstring("To run yarn in offline mode, see: https://yarnpkg.com/blog/2016/11/24/offline-mirror"))
			})

			It("runs yarn install", func() {
				Expect(y.Build()).To(Succeed())
			})

			Context("package.json matches yarn.lock", func() {
				BeforeEach(func() {
					yarnCheck = nil
				})

				It("reports the fact", func() {
					Expect(y.Build()).To(Succeed())
					Expect(buffer.String()).To(ContainSubstring("yarn.lock and package.json match"))
				})
			})

			Context("package.json does not match yarn.lock", func() {
				BeforeEach(func() {
					yarnCheck = &exec.ExitError{}
				})

				It("warns the user", func() {
					Expect(y.Build()).To(Succeed())
					Expect(buffer.String()).To(ContainSubstring("**WARNING** yarn.lock is outdated"))
				})
			})
		})
	})
})
