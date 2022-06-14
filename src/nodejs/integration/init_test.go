package integration_test

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/cloudfoundry/switchblade"
	"github.com/onsi/gomega/format"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var (
	ChecksumRegexp = regexp.MustCompile(`Checksum (Before|After) \(.*\): ([0-9a-f]+)`)

	settings struct {
		Buildpack struct {
			Version string
			Path    string
		}

		Cached      bool
		Serial      bool
		GitHubToken string
		Platform    string
	}
)

func init() {
	flag.BoolVar(&settings.Cached, "cached", false, "run cached buildpack tests")
	flag.BoolVar(&settings.Serial, "serial", false, "run serial buildpack tests")
	flag.StringVar(&settings.Platform, "platform", "cf", `switchblade platform to test against ("cf" or "docker")`)
	flag.StringVar(&settings.GitHubToken, "github-token", "", "use the token to make GitHub API requests")
}

func TestIntegration(t *testing.T) {
	var Expect = NewWithT(t).Expect

	format.MaxLength = 0
	SetDefaultEventuallyTimeout(10 * time.Second)

	root, err := filepath.Abs("./../../..")
	Expect(err).NotTo(HaveOccurred())

	fixtures := filepath.Join(root, "fixtures")

	platform, err := switchblade.NewPlatform(settings.Platform, settings.GitHubToken)
	Expect(err).NotTo(HaveOccurred())

	err = platform.Initialize(
		switchblade.Buildpack{
			Name: "nodejs_buildpack",
			URI:  os.Getenv("BUILDPACK_FILE"),
		},
		switchblade.Buildpack{
			Name: "override_buildpack",
			URI:  filepath.Join(fixtures, "util", "override_buildpack"),
		},
	)
	Expect(err).NotTo(HaveOccurred())

	proxyName, err := switchblade.RandomName()
	Expect(err).NotTo(HaveOccurred())

	proxyDeployment, _, err := platform.Deploy.
		WithBuildpacks("go_buildpack").
		Execute(proxyName, filepath.Join(fixtures, "util", "proxy"))
	Expect(err).NotTo(HaveOccurred())

	dynatraceName, err := switchblade.RandomName()
	Expect(err).NotTo(HaveOccurred())

	dynatraceDeployment, _, err := platform.Deploy.
		WithBuildpacks("go_buildpack").
		Execute(dynatraceName, filepath.Join(fixtures, "util", "dynatrace"))
	Expect(err).NotTo(HaveOccurred())

	suite := spec.New("integration", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Default", testDefault(platform, fixtures))
	suite("HTTPS", testHTTPS(platform, fixtures))
	suite("Memory", testMemory(platform, fixtures))
	suite("Multibuildpack", testMultibuildpack(platform, fixtures))
	suite("NPM", testNPM(platform, fixtures))
	suite("Override", testOverride(platform, fixtures))
	suite("Vendored", testVendored(platform, fixtures))
	suite("Versions", testVersions(platform, fixtures))
	suite("Yarn", testYarn(platform, fixtures))

	suite("Appdynamics", testAppdynamics(platform, fixtures))
	suite("ContrastSecurity", testContrastSecurity(platform, fixtures))
	suite("Dynatrace", testDynatrace(platform, fixtures, dynatraceDeployment.InternalURL))
	suite("NewRelic", testNewRelic(platform, fixtures))
	suite("Sealights", testSealights(platform, fixtures))
	suite("Seeker", testSeeker(platform, fixtures))
	suite("Snyk", testSnyk(platform, fixtures))

	if settings.Cached {
		suite("Offline", testOffline(platform, fixtures))
	} else {
		suite("Cache", testCache(platform, fixtures))
		suite("Proxy", testProxy(platform, fixtures, proxyDeployment.InternalURL))
	}

	suite.Run(t)

	Expect(platform.Delete.Execute(proxyName)).To(Succeed())
	Expect(platform.Delete.Execute(dynatraceName)).To(Succeed())
	Expect(os.Remove(os.Getenv("BUILDPACK_FILE"))).To(Succeed())
}
