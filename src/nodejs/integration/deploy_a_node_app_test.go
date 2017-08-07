package integration_test

import (
	"os/exec"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF NodeJS Buildpack", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	Context("when specifying a range for the nodeJS version in the package.json", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "node_version_range"))
		})

		It("resolves to a nodeJS version successfully", func() {
			PushAppAndConfirm(app)

			Expect(app.Stdout.String()).To(MatchRegexp("Installing node 4\\.\\d+\\.\\d+"))
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
		})
	})

	Context("when specifying a version 6 for the nodeJS version in the package.json", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "node_version_6"))
		})

		It("resolves to a nodeJS version successfully", func() {
			PushAppAndConfirm(app)

			Expect(app.Stdout.String()).To(MatchRegexp("Installing node 6\\.\\d+\\.\\d+"))
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))

			if ApiHasTask() {
				By("running a task", func() {
					By("can find node in the container", func() {
						command := exec.Command("cf", "run-task", app.Name, "echo \"RUNNING A TASK: $(node --version)\"")
						_, err := command.Output()
						Expect(err).To(BeNil())

						Eventually(func() string {
							return app.Stdout.String()
						}, "30s").Should(MatchRegexp("RUNNING A TASK: v6\\.\\d+\\.\\d+"))
					})
				})
			}
		})
	})

	Context("when not specifying a nodeJS version in the package.json", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "without_node_version"))
		})

		It("resolves to the stable nodeJS version successfully", func() {
			PushAppAndConfirm(app)

			Expect(app.Stdout.String()).To(MatchRegexp("Installing node 4\\.\\d+\\.\\d+"))
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
		})
	})

	Context("with an unreleased nodejs version", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "unreleased_node_version"))
		})

		It("displays a nice error messages and gracefully fails", func() {
			Expect(app.Push()).ToNot(BeNil())
			Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())

			Expect(app.Stdout.String()).To(ContainSubstring("Unable to install node: no match found for 9000.0.0"))
		})
	})

	Context("with an unsupported, but released, nodejs version", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "unsupported_node_version"))
		})

		It("displays a nice error messages and gracefully fails", func() {
			Expect(app.Push()).ToNot(BeNil())
			Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())

			Expect(app.Stdout.String()).To(ContainSubstring("Unable to install node: no match found for 4.1.1"))
		})
	})

	Context("with no Procfile and OPTIMIZE_MEMORY=true", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "simple_app"))
			app.SetEnv("OPTIMIZE_MEMORY", "true")
		})

		It("is running with autosized max_old_space_size", func() {
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("MaxOldSpace: 96")) // 128 * 75%
		})
	})

	Context("with no Procfile and OPTIMIZE_MEMORY is unset", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "simple_app"))
		})

		It("is running with autosized max_old_space_size", func() {
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("MaxOldSpace: undefined"))
		})
	})

	It("with an app that has vendored dependencies", func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "vendored_dependencies"))
		PushAppAndConfirm(app)

		By("does not output protip that recommends user vendors dependencies", func() {
			Expect(app.Stdout.String()).ToNot(MatchRegexp("PRO TIP:(.*) It is recommended to vendor the application's Node.js dependencies"))
		})

		if !cutlass.Cached {
			By("with an uncached buildpack", func() {
				By("successfully deploys and includes the dependencies", func() {
					Expect(app.GetBody("/")).To(ContainSubstring("0000000005"))
					Expect(app.Stdout.String()).To(ContainSubstring("Download [https://"))
				})
			})
		}

		if cutlass.Cached {
			By("with a cached buildpack", func() {
				By("deploys without hitting the internet", func() {
					Expect(app.GetBody("/")).To(ContainSubstring("0000000005"))
					Expect(app.Stdout.String()).To(ContainSubstring("Copy [/tmp/buildpacks/"))
				})
			})
		}
	})

	Context("with an app that has vendored dependencies", func() {
		BeforeEach(func() {
			if !cutlass.Cached {
				Skip("cached tests")
			}
		})

		AssertNoInternetTraffic("vendored_dependencies")
	})

	Context("with an app with a yarn.lock file", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_yarn"))
		})

		It("successfully deploys and vendors the dependencies via yarn", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).To(ContainSubstring("Running yarn in online mode"))

			Expect(filepath.Join(app.Path, "node_modules")).ToNot(BeADirectory())
			Expect(app.Files("app")).To(ContainElement("app/node_modules"))

			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
		})

		AssertUsesProxyDuringStagingIfPresent("with_yarn")
	})

	Context("with an app with a yarn.lock and vendored dependencies", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_yarn_vendored"))
		})

		It("deploys without hitting the internet", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).To(ContainSubstring("Running yarn in offline mode"))
			Expect(app.GetBody("/microtime")).To(MatchRegexp("native time: \\d+\\.\\d+"))
		})

		AssertNoInternetTraffic("with_yarn_vendored")
	})

	Context("with an app with an out of date yarn.lock", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "out_of_date_yarn_lock"))
		})

		It("warns that yarn.lock is out of date", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).To(ContainSubstring("yarn.lock is outdated"))
		})
	})

	Context("with an app with pre and post scripts", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "pre_post_commands"))
		})

		It("runs the scripts through npm run", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).To(ContainSubstring("Running heroku-prebuild (npm)"))
			Expect(app.Stdout.String()).To(ContainSubstring("Running heroku-postbuild (npm)"))
			Expect(app.GetBody("/")).To(ContainSubstring("Text: Hello Buildpacks Team"))
			Expect(app.GetBody("/")).To(ContainSubstring("Text: Goodbye Buildpacks Team"))
		})
	})

	Context("with an app with no vendored dependencies", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "no_vendored_dependencies"))
		})

		It("successfully deploys and vendors the dependencies", func() {
			PushAppAndConfirm(app)

			Expect(filepath.Join(app.Path, "node_modules")).ToNot(BeADirectory())
			Expect(app.Files("app")).To(ContainElement("app/node_modules"))

			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))

			By("outputs protip that recommends user vendors dependencies", func() {
				Expect(app.Stdout.String()).To(MatchRegexp("PRO TIP:(.*) It is recommended to vendor the application's Node.js dependencies"))
			})
		})

		AssertUsesProxyDuringStagingIfPresent("no_vendored_dependencies")
	})

	Context("with an incomplete node_modules directory", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "incomplete_node_modules"))
		})

		It("downloads missing dependencies from package.json", func() {
			PushAppAndConfirm(app)
			Expect(filepath.Join(app.Path, "node_modules", "hashish")).ToNot(BeADirectory())
			Expect(app.Files("app/node_modules")).To(ContainElement("app/node_modules/hashish"))
			Expect(app.Files("app/node_modules")).To(ContainElement("app/node_modules/express"))
		})
	})

	Context("with an incomplete package.json", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "incomplete_package_json"))
		})

		It("does not overwrite the vendored modules not listed in package.json", func() {
			PushAppAndConfirm(app)
			Expect(app.Files("app/node_modules")).To(ContainElement("app/node_modules/leftpad"))
			Expect(app.Files("app/node_modules")).To(ContainElement("app/node_modules/hashish"))
		})
	})

	PContext("with a cached buildpack in an air gapped environment", func() {
		BeforeEach(func() {
			if !cutlass.Cached {
				Skip("cached tests")
			}
		})
		// TODO :cached tag only
		//     before(:each) do
		//       `cf unbind-staging-security-group public_networks`
		//       `cf unbind-staging-security-group dns`
		//     end
		//
		//     after(:each) do
		//       `cf bind-staging-security-group public_networks`
		//       `cf bind-staging-security-group dns`
		//     end
		//
		//     context 'with no npm version specified' do
		//       let (:app_name) { 'airgapped_no_npm_version' }
		//
		//       subject(:app) do
		//         Machete.deploy_app(app_name)
		//       end
		//
		//       it 'is running with the default version of npm' do
		//         expect(app).to be_running
		//         expect(app).not_to have_internet_traffic
		//
		//         default_version = YAML.load_file(File.join(File.dirname(__FILE__), '..', '..', 'manifest.yml'))['default_versions'].find { |a| a['name'] == 'node' }['version']
		//         expect(app).to have_logged /Installing node #{default_version}/
		//         expect(app).to have_logged("Using default npm version")
		//       end
		//     end
		//
		//     context 'with invalid npm version specified' do
		//       let (:app_name) { 'airgapped_invalid_npm_version' }
		//
		//       it 'is not running and prints an error message' do
		//         expect(app).not_to be_running
		//         expect(app).to have_logged("We're unable to download the version of npm")
		//       end
		//     end
	})

	Describe("NODE_HOME and NODE_ENV", func() {
		BeforeEach(func() {
			if !cutlass.Cached {
				Skip("cached tests")
			}
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "logenv"))
		})

		It("sets the NODE_HOME to correct value", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).To(MatchRegexp("NODE_HOME=\\S*/0/node"))

			body, err := app.GetBody("/")
			Expect(err).To(BeNil())
			Expect(body).To(MatchRegexp(`"NODE_HOME":"[^"]*/0/node"`))
			Expect(body).To(ContainSubstring(`"NODE_ENV":"production"`))
			Expect(body).To(ContainSubstring(`"MEMORY_AVAILABLE":"128"`))
		})
	})
})
