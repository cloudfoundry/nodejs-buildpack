package pnpm_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	p "github.com/cloudfoundry/nodejs-buildpack/src/nodejs/pnpm"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=pnpm.go --destination=mocks_test.go --package=pnpm_test

// MockCommand is a hand-written mock matching the pnpm.Command interface.
type MockCommand struct {
	ctrl     *gomock.Controller
	recorder *MockCommandMockRecorder
}

type MockCommandMockRecorder struct {
	mock *MockCommand
}

func NewMockCommand(ctrl *gomock.Controller) *MockCommand {
	mock := &MockCommand{ctrl: ctrl}
	mock.recorder = &MockCommandMockRecorder{mock}
	return mock
}

func (m *MockCommand) EXPECT() *MockCommandMockRecorder { return m.recorder }

func (m *MockCommand) Execute(dir string, stdout, stderr io.Writer, program string, args ...string) error {
	varargs := []interface{}{dir, stdout, stderr, program}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Execute", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockCommandMockRecorder) Execute(arg0, arg1, arg2, arg3 interface{}, arg4 ...interface{}) *gomock.Call {
	varargs := append([]interface{}{arg0, arg1, arg2, arg3}, arg4...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Execute", reflect.TypeOf((*MockCommand)(nil).Execute), varargs...)
}

func (m *MockCommand) Run(cmd *exec.Cmd) error {
	ret := m.ctrl.Call(m, "Run", cmd)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockCommandMockRecorder) Run(cmd interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockCommand)(nil).Run), cmd)
}

var _ = Describe("PNPM", func() {
	var (
		err         error
		buildDir    string
		cacheDir    string
		pnpm        *p.PNPM
		logger      *libbuildpack.Logger
		buffer      *bytes.Buffer
		mockCtrl    *gomock.Controller
		mockCommand *MockCommand
		oldNodeHome string
	)

	BeforeEach(func() {
		buildDir, err = os.MkdirTemp("", "nodejs-buildpack.build.")
		Expect(err).To(BeNil())
		cacheDir, err = os.MkdirTemp("", "nodejs-buildpack.cache.")
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))

		mockCtrl = gomock.NewController(GinkgoT())
		mockCommand = NewMockCommand(mockCtrl)

		pnpm = &p.PNPM{
			Log:     logger,
			Command: mockCommand,
		}

		oldNodeHome = os.Getenv("NODE_HOME")
		Expect(os.Setenv("NODE_HOME", "test_node_home")).To(Succeed())
	})

	AfterEach(func() {
		mockCtrl.Finish()
		Expect(os.Setenv("NODE_HOME", oldNodeHome)).To(Succeed())
		Expect(os.RemoveAll(buildDir)).To(Succeed())
		Expect(os.RemoveAll(cacheDir)).To(Succeed())
	})

	Describe("Rebuild", func() {
		It("runs pnpm rebuild with NODE_HOME set", func() {
			mockCommand.EXPECT().Run(gomock.Any()).DoAndReturn(func(cmd *exec.Cmd) error {
				Expect(cmd.Args).To(Equal([]string{"pnpm", "rebuild"}))
				Expect(cmd.Dir).To(Equal(buildDir))
				Expect(cmd.Env).To(ContainElement("npm_config_nodedir=test_node_home"))
				return nil
			})
			Expect(pnpm.Rebuild(buildDir)).To(Succeed())
			Expect(buffer.String()).To(ContainSubstring("Rebuilding any native modules (pnpm)"))
		})

		It("propagates command errors", func() {
			mockCommand.EXPECT().Run(gomock.Any()).Return(fmt.Errorf("pnpm rebuild failed"))
			Expect(pnpm.Rebuild(buildDir)).To(MatchError("pnpm rebuild failed"))
		})
	})

	Describe("Build", func() {
		Context("online mode (no .pnpm-store in buildDir)", func() {
			It("runs pnpm install with cache dir", func() {
				mockCommand.EXPECT().Run(gomock.Any()).DoAndReturn(func(cmd *exec.Cmd) error {
					Expect(cmd.Args).To(Equal([]string{"pnpm", "install", "--no-frozen-lockfile"}))
					Expect(cmd.Dir).To(Equal(buildDir))
					Expect(cmd.Env).To(ContainElement("PNPM_HOME=" + cacheDir + "/.pnpm-store"))
					return nil
				})
				Expect(pnpm.Build(buildDir, cacheDir)).To(Succeed())
			})
		})

		Context("offline mode (.pnpm-store exists in buildDir)", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(buildDir+"/.pnpm-store", 0755)).To(Succeed())
			})

			It("runs pnpm install --offline", func() {
				mockCommand.EXPECT().Run(gomock.Any()).DoAndReturn(func(cmd *exec.Cmd) error {
					Expect(cmd.Args).To(Equal([]string{"pnpm", "install", "--no-frozen-lockfile", "--offline"}))
					return nil
				})
				Expect(pnpm.Build(buildDir, cacheDir)).To(Succeed())
			})
		})
	})
})
