package integration_test

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/blang/semver"
	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/cutlass"
	"github.com/cloudfoundry/libbuildpack/packager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var bpDir, bpFile string
var buildpackVersion string

func init() {
	flag.StringVar(&buildpackVersion, "version", "", "version to use (builds if empty)")
	flag.BoolVar(&cutlass.Cached, "cached", true, "cached buildpack")
	flag.StringVar(&cutlass.DefaultMemory, "memory", "128M", "default memory for pushed apps")
	flag.StringVar(&cutlass.DefaultDisk, "disk", "256M", "default disk for pushed apps")
	flag.Parse()
}

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

func PushAppAndConfirm(app *cutlass.App) {
	Expect(app.Push()).To(Succeed())
	Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
	Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())
}

func ApiHasTask() bool {
	apiVersionString, err := cutlass.ApiVersion()
	Expect(err).To(BeNil())
	apiVersion, err := semver.Make(apiVersionString)
	Expect(err).To(BeNil())
	apiHasTask, err := semver.ParseRange("> 2.75.0")
	Expect(err).To(BeNil())
	return apiHasTask(apiVersion)
}

func findRoot() string {
	file := "VERSION"
	for {
		files, err := filepath.Glob(file)
		Expect(err).To(BeNil())
		if len(files) == 1 {
			file, err = filepath.Abs(filepath.Dir(file))
			Expect(err).To(BeNil())
			return file
		}
		file = filepath.Join("..", file)
	}
}

var _ = SynchronizedBeforeSuite(func() []byte {
	// Run once
	bpDir = findRoot()

	if buildpackVersion == "" {
		data, err := ioutil.ReadFile(filepath.Join(bpDir, "VERSION"))
		Expect(err).NotTo(HaveOccurred())
		buildpackVersion = string(data)
		buildpackVersion = fmt.Sprintf("%s.%s", buildpackVersion, time.Now().Format("20060102150405"))

		file, err := packager.Package(bpDir, packager.CacheDir, buildpackVersion, cutlass.Cached)
		Expect(err).To(BeNil())

		var manifest struct {
			Language string `yaml:"language"`
		}
		Expect(libbuildpack.NewYAML().Load(filepath.Join(bpDir, "manifest.yml"), &manifest)).To(Succeed())
		Expect(cutlass.UpdateBuildpack(manifest.Language, file)).To(Succeed())

		return []byte(file)
	}

	return []byte{}
}, func(localBpFile []byte) {
	// Run on all nodes
	bpFile = ""
	if len(localBpFile) > 0 {
		bpFile = string(localBpFile)
	}
	bpDir = findRoot()
	cutlass.DefaultStdoutStderr = GinkgoWriter
})

var _ = SynchronizedAfterSuite(func() {
	// Run on all nodes
}, func() {
	// Run once
	if bpFile != "" {
		os.Remove(bpFile)
	}
	cutlass.DeleteOrphanedRoutes()
})

func AssertUsesProxyDuringStagingIfPresent(fixtureName string) {
	Context("with an uncached buildpack", func() {
		BeforeEach(func() {
			if cutlass.Cached {
				Skip("Running cached tests")
			}
		})

		It("uses a proxy during staging if present", func() {
			proxy, err := cutlass.NewProxy()
			Expect(err).To(BeNil())
			defer proxy.Close()

			traffic, err := cutlass.InternetTraffic(
				bpDir,
				filepath.Join("fixtures", fixtureName),
				bpFile,
				[]string{"HTTP_PROXY=" + proxy.URL, "HTTPS_PROXY=" + proxy.URL},
			)
			Expect(err).To(BeNil())

			destUrl, err := url.Parse(proxy.URL)
			Expect(err).To(BeNil())

			Expect(cutlass.UniqueDestination(
				traffic, fmt.Sprintf("%s.%s", destUrl.Hostname(), destUrl.Port()),
			)).To(BeNil())
		})
	})
}

func AssertNoInternetTraffic(fixtureName string) {
	It("does not call out over the internet", func() {
		if !cutlass.Cached {
			Skip("Running uncached tests")
		}

		traffic, err := cutlass.InternetTraffic(
			bpDir,
			filepath.Join("fixtures", fixtureName),
			bpFile,
			[]string{},
		)
		Expect(err).To(BeNil())
		Expect(traffic).To(BeEmpty())
	})
}
