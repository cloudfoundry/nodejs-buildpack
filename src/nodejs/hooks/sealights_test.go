package hooks_test

import (
	"bytes"
	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/nodejs-buildpack/src/nodejs/hooks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var _ = Describe("Sealights hook", func() {
	var (
		err               error
		buildDir          string
		logger            *libbuildpack.Logger
		buffer            *bytes.Buffer
		stager            *libbuildpack.Stager
		sealights         *hooks.SealightsHook
		token             string
		build             string
		proxy             string
		labId             string
		projectRoot       string
		testStage         string
		procfile          string
		testProcfile      = "web: node index.js --build 192 --name Good"
		expected          = strings.ReplaceAll("web: node ./node_modules/.bin/slnodejs run  --useinitialcolor true --token good_token --buildsessionid goodBsid --proxy http://localhost:1886 --labid Roni's --projectroot project/root --teststage \"Unit Tests\" index.js --build 192 --name Good", " ", "")
		expectedWithFiles = strings.ReplaceAll("web: node ./node_modules/.bin/slnodejs run  --useinitialcolor true --tokenfile application/token/file --buildsessionidfile build/id/file --proxy http://localhost:1886 --labid Roni's --projectroot project/root --teststage \"Unit Tests\" index.js --build 192 --name Good", " ", "")
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "nodejs-buildpack.build.")
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(buffer)
		args := []string{buildDir, ""}
		stager = libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})

		token = os.Getenv("SL_TOKEN_FILE")
		build = os.Getenv("SL_BUILD_SESSION_ID_FILE")
		proxy = os.Getenv("SL_PROXY")
		labId = os.Getenv("SL_LAB_ID")
		projectRoot = os.Getenv("SL_PROJECT_ROOT")
		testStage = os.Getenv("SL_TEST_STAGE")
		err = ioutil.WriteFile(filepath.Join(stager.BuildDir(), "Procfile"), []byte(testProcfile), 0755)
		Expect(err).To(BeNil())

		sealights = &hooks.SealightsHook{
			libbuildpack.DefaultHook{},
			logger,
			&libbuildpack.Command{},
		}
	})

	AfterEach(func() {
		err = os.Setenv("SL_TOKEN", token)
		Expect(err).To(BeNil())
		err = os.Setenv("SL_BUILD_SESSION_ID", build)
		Expect(err).To(BeNil())
		err = os.Setenv("SL_PROXY", proxy)
		Expect(err).To(BeNil())
		err = os.Setenv("SL_LAB_ID", labId)
		Expect(err).To(BeNil())
		err = os.Setenv("SL_PROJECT_ROOT", projectRoot)
		Expect(err).To(BeNil())
		err = os.Setenv("SL_TEST_STAGE", testStage)
		Expect(err).To(BeNil())

		err = ioutil.WriteFile(filepath.Join(stager.BuildDir(), "Procfile"), []byte(procfile), 0755)
		Expect(err).To(BeNil())

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())
	})

	Describe("AfterCompile", func() {
		var (
			token     = "good_token"
			tokenFile = "application/token/file"
			bsid      = "goodBsid"
			bsidFile  = "build/id/file"
			proxy     = "http://localhost:1886"
			lab       = "Roni's"
			root      = "project/root"
			stage     = "Unit Tests"
		)
		Context("build new application run command in Procfile", func() {
			BeforeEach(func() {
				err = os.Setenv("SL_TOKEN", token)
				Expect(err).To(BeNil())
				err = os.Setenv("SL_TOKEN_FILE", tokenFile)
				Expect(err).To(BeNil())
				err = os.Setenv("SL_BUILD_SESSION_ID", bsid)
				Expect(err).To(BeNil())
				err = os.Setenv("SL_BUILD_SESSION_ID_FILE", bsidFile)
				Expect(err).To(BeNil())
				err = os.Setenv("SL_PROXY", proxy)
				Expect(err).To(BeNil())
			})
			It("test application run cmd creation", func() {
				err = os.Setenv("SL_LAB_ID", lab)
				Expect(err).To(BeNil())
				err = os.Setenv("SL_PROJECT_ROOT", root)
				Expect(err).To(BeNil())
				err = os.Setenv("SL_TEST_STAGE", stage)
				Expect(err).To(BeNil())
				err = sealights.SetApplicationStart(stager)
				Expect(err).To(BeNil())
				bytes, err := ioutil.ReadFile(filepath.Join(stager.BuildDir(), "Procfile"))
				Expect(err).To(BeNil())
				cleanResult := strings.ReplaceAll(string(bytes), " ", "")
				Expect(cleanResult).To(Equal(expectedWithFiles))
			})
			It("hook fails with empty token", func() {
				err = os.Setenv("SL_TOKEN", "")
				Expect(err).To(BeNil())
				err = os.Setenv("SL_TOKEN_FILE", "")
				Expect(err).To(BeNil())
				err = sealights.SetApplicationStart(stager)
				Expect(err).To(MatchError(ContainSubstring(hooks.EmptyTokenError)))
			})
			It("hook fails with empty build session id", func() {
				err = os.Setenv("SL_BUILD_SESSION_ID", "")
				Expect(err).NotTo(HaveOccurred())
				err = os.Setenv("SL_BUILD_SESSION_ID_FILE", "")
				Expect(err).NotTo(HaveOccurred())
				err = sealights.SetApplicationStart(stager)
				Expect(err).To(MatchError(ContainSubstring(hooks.EmptyBuildError)))
			})
			It("hook fails with empty build session id", func() {
				err = os.Setenv("SL_LAB_ID", lab)
				Expect(err).To(BeNil())
				err = os.Setenv("SL_PROJECT_ROOT", root)
				Expect(err).To(BeNil())
				err = os.Setenv("SL_TEST_STAGE", stage)
				Expect(err).To(BeNil())
				err = os.Setenv("SL_BUILD_SESSION_ID_FILE", "")
				Expect(err).NotTo(HaveOccurred())
				err = os.Setenv("SL_TOKEN_FILE", "")
				Expect(err).To(BeNil())
				err = sealights.SetApplicationStart(stager)
				bytes, err := ioutil.ReadFile(filepath.Join(stager.BuildDir(), "Procfile"))
				Expect(err).To(BeNil())
				cleanResult := strings.ReplaceAll(string(bytes), " ", "")
				Expect(cleanResult).To(Equal(expected))
			})
		})
	})
})
