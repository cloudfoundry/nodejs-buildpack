package integration_test

import (
	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("pushing an app a second time", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	BeforeEach(func() {
		if cutlass.Cached {
			Skip("running uncached tests")
		}

		app = cutlass.New(Fixtures("simple_app"))
		app.Buildpacks = []string{"nodejs_buildpack"}
	})

	Regexp := `\[.*\/node[\-_][\d.]+[\-_]linux[\-_](amd64)?(x64)?[\-_]cflinuxfs\d[\-_][\da-f]+\.tgz\]`
	DownloadRegexp := "Download " + Regexp
	CopyRegexp := "Copy " + Regexp

	It("uses the cache for manifest dependencies", func() {
		PushAppAndConfirm(app)
		Expect(app.Stdout.String()).To(MatchRegexp(DownloadRegexp))
		Expect(app.Stdout.String()).ToNot(MatchRegexp(CopyRegexp))

		app.Stdout.Reset()
		PushAppAndConfirm(app)
		Expect(app.Stdout.String()).To(MatchRegexp(CopyRegexp))
		Expect(app.Stdout.String()).ToNot(MatchRegexp(DownloadRegexp))
	})
})
