package matchers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"

	"github.com/cloudfoundry/switchblade"
	"github.com/onsi/gomega/types"
)

type ServeMatcher struct {
	expected interface{}
	endpoint string
	response string
}

func Serve(expected interface{}) *ServeMatcher {
	return &ServeMatcher{
		expected: expected,
	}
}

func (sm *ServeMatcher) WithEndpoint(endpoint string) *ServeMatcher {
	sm.endpoint = endpoint
	return sm
}

func (sm *ServeMatcher) Match(actual interface{}) (success bool, err error) {
	deployment, ok := actual.(switchblade.Deployment)
	if !ok {
		return false, fmt.Errorf("ServeMatcher expects a switchblade.Deployment, received %T", actual)
	}

	uri, err := url.Parse(deployment.ExternalURL)
	if err != nil {
		return false, err
	}

	uri.Path = sm.endpoint

	response, err := http.Get(uri.String())
	if err != nil {
		return false, err
	}
	defer response.Body.Close()

	content, err := io.ReadAll(response.Body)
	if err != nil {
		return false, err
	}

	sm.response = string(content)

	if response.StatusCode != http.StatusOK {
		return false, nil
	}

	return sm.compare(string(content), sm.expected)
}

func (sm *ServeMatcher) compare(actual string, expected interface{}) (bool, error) {
	if m, ok := expected.(types.GomegaMatcher); ok {
		match, err := m.Match(actual)
		if err != nil {
			return false, err
		}

		return match, nil
	}

	return reflect.DeepEqual(actual, expected), nil
}

func (sm *ServeMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected the response from deployment:\n\n\t%s\n\nto contain:\n\n\t%s", sm.response, sm.expected)
}

func (sm *ServeMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected the response from deployment:\n\n\t%s\n\nnot to contain:\n\n\t%s", sm.response, sm.expected)
}
