package matchers

import (
	"fmt"
	"net/http"

	"github.com/cloudfoundry/occam"
	"github.com/onsi/gomega/types"
)

func BeAvailable() types.GomegaMatcher {
	return &BeAvailableMatcher{}
}

type BeAvailableMatcher struct {
}

func (*BeAvailableMatcher) Match(actual interface{}) (bool, error) {
	container, ok := actual.(occam.Container)
	if !ok {
		return false, fmt.Errorf("BeAvailableMatcher expects an occam.Container, received %T", actual)
	}

	_, err := http.Get(fmt.Sprintf("http://localhost:%s", container.Ports["8080"]))
	if err != nil {
		return false, err
	}

	return true, nil
}

func (*BeAvailableMatcher) FailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected\n\t%#v\nto be available", actual)
}

func (*BeAvailableMatcher) NegatedFailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected\n\t%#v\nnot to be available", actual)
}
