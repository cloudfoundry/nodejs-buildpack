package integration_test

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/cloudfoundry/libbuildpack/cutlass"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

type haveUnchangedAppDirMatcher struct {
	checksumBefore, checksumAfter string
}

func HaveUnchangedAppDir() types.GomegaMatcher {
	return &haveUnchangedAppDirMatcher{}
}

func (m *haveUnchangedAppDirMatcher) Match(actual interface{}) (success bool, err error) {
	app, ok := actual.(*cutlass.App)
	if !ok {
		return false, fmt.Errorf("HaveUnchangedAppDir matcher requires a cutlass.App. Got:\n%s", format.Object(actual, 1))
	}

	re := regexp.MustCompile(`Checksum Before \(.*\): ([0-9a-f]+)`)
	matches := re.FindAllStringSubmatch(app.Stdout.String(), -1)
	if len(matches) < 1 || len(matches[0]) < 2 {
		return false, errors.New("HaveUnchangedAppDir matcher did not find 'Checksum Before' message")
	}

	if len(matches) > 1 {
		return false, errors.New("HaveUnchangedAppDir matcher found more than one 'Checksum Before' message")
	}
	m.checksumBefore = matches[0][1]

	re = regexp.MustCompile(`Checksum After \(.*\): ([0-9a-f]+)`)
	matches = re.FindAllStringSubmatch(app.Stdout.String(), -1)
	if len(matches) < 1 || len(matches[0]) < 2 {
		return false, fmt.Errorf("HaveUnchangedAppDir matcher did not find 'Checksum After' message")
	}

	if len(matches) > 1 {
		return false, errors.New("HaveUnchangedAppDir matcher found more than one 'Checksum After' message")
	}
	m.checksumAfter = matches[0][1]

	return m.checksumBefore != "" && m.checksumBefore == m.checksumAfter, nil
}

func (m *haveUnchangedAppDirMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(m.checksumBefore, "to be the same checksum as", m.checksumAfter)
}

func (m *haveUnchangedAppDirMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(m.checksumBefore, "to be a different checksum to", m.checksumAfter)
}
