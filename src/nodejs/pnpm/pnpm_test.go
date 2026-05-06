package pnpm_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/nodejs-buildpack/src/nodejs/pnpm"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=pnpm.go --destination=mocks_test.go --package=pnpm_test

var _ = Describe("PNPM", func() {
	var (
		err         error
		buildDir    string
		cacheDir    string
		p           *pnpm.PNPM
		logger      *libbuildpack.Logger
		buffer      *bytes.Buffer
		mockCtrl    *gomock.Controller
		mockCommand *MockCommand
	)

	BeforeEach(func() {
		buildDir, err = os.MkdirTemp("", "nodejs-buildpack.build.")
		Expect(err).NotTo(HaveOccurred())

		cacheDir, err = os.MkdirTemp("", "nodejs-buildpack.cache.")
		Expect(err).NotTo(HaveOccurred())

		buffer = new(bytes.Buffer)

		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))

		mockCtrl = gomock.NewController(GinkgoT())
		mockCommand = NewMockCommand(mockCtrl)

		p = &pnpm.PNPM{
			Log:     logger,
			Command: mockCommand,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())
		os.RemoveAll(cacheDir)
	})

	Describe("Build", func() {
		var oldNodeHome string
		var pnpmConfig map[string]string
		var pnpmInstallArgs []string

		AfterEach(func() {
			Expect(os.Setenv("NODE_HOME", oldNodeHome)).To(Succeed())
		})
		BeforeEach(func() {
			oldNodeHome = os.Getenv("NODE_HOME")
			Expect(os.Setenv("NODE_HOME", "test_node_home")).To(Succeed())

			pnpmConfig = map[string]string{}
			mockCommand.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), "pnpm", gomock.Any()).DoAndReturn(func(dir string, stdout, stderr io.Writer, program string, args ...string) error {
				switch args[0] {
				case "config":
					Expect(args[0:2]).To(Equal([]string{"config", "set"}))
					pnpmConfig[args[2]] = args[3]
				default:
					pnpmInstallArgs = append([]string{"pnpm"}, args...)
				}
				Expect(dir).To(Equal(buildDir))
				return nil
			}).AnyTimes()
		})

		It("runs pnpm config and install", func() {
			Expect(p.Build(buildDir, cacheDir)).To(Succeed())

			Expect(buffer.String()).To(ContainSubstring("Installing node modules (pnpm-lock.yaml)"))
			Expect(buffer.String()).To(ContainSubstring("Using pnpm store directory: " + filepath.Join(cacheDir, ".pnpm-store")))

			Expect(pnpmConfig).To(Equal(map[string]string{
				"store-dir": filepath.Join(cacheDir, ".pnpm-store"),
			}))

			Expect(pnpmInstallArgs).To(Equal([]string{
				"pnpm", "install",
				"--frozen-lockfile",
			}))
		})

		Context("when NPM_CONFIG_PRODUCTION is true", func() {
			BeforeEach(func() {
				Expect(os.Setenv("NPM_CONFIG_PRODUCTION", "true")).To(Succeed())
			})

			AfterEach(func() {
				Expect(os.Unsetenv("NPM_CONFIG_PRODUCTION")).To(Succeed())
			})

			It("runs pnpm install with --prod", func() {
				Expect(p.Build(buildDir, cacheDir)).To(Succeed())

				Expect(buffer.String()).To(ContainSubstring("NPM_CONFIG_PRODUCTION is true, installing only production dependencies"))
				Expect(pnpmInstallArgs).To(ContainElement("--prod"))
			})
		})

		Context("when a vendored pnpm store exists", func() {
			BeforeEach(func() {
				Expect(os.Mkdir(filepath.Join(buildDir, ".pnpm-store"), 0755)).To(Succeed())
			})

			It("runs pnpm install with --offline and uses the vendored store", func() {
				Expect(p.Build(buildDir, cacheDir)).To(Succeed())

				Expect(buffer.String()).To(ContainSubstring("Found vendored pnpm store"))
				Expect(buffer.String()).To(ContainSubstring("Running pnpm in offline mode"))

				Expect(pnpmConfig).To(HaveKeyWithValue("store-dir", filepath.Join(buildDir, ".pnpm-store")))
				Expect(pnpmInstallArgs).To(ContainElement("--offline"))
			})
		})
	})
})
