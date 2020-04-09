package integration_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cloudfoundry/dagger"
	"github.com/cloudfoundry/occam"
	"github.com/cloudfoundry/packit/pexec"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

var (
	bpDir         string
	npmURI        string
	npmCachedURI  string
	nodeURI       string
	nodeCachedURI string
)

func TestIntegration(t *testing.T) {
	var (
		Expect = NewWithT(t).Expect
		err    error
	)

	bpDir, err = dagger.FindBPRoot()
	Expect(err).NotTo(HaveOccurred())

	npmURI, err = dagger.PackageBuildpack(bpDir)
	Expect(err).ToNot(HaveOccurred())

	npmCachedURI, _, err = dagger.PackageCachedBuildpack(bpDir)
	Expect(err).ToNot(HaveOccurred())

	nodeURI, err = dagger.GetLatestBuildpack("node-engine-cnb")
	Expect(err).ToNot(HaveOccurred())

	nodeRepo, err := dagger.GetLatestUnpackagedBuildpack("node-engine-cnb")
	Expect(err).ToNot(HaveOccurred())

	nodeCachedURI, _, err = dagger.PackageCachedBuildpack(nodeRepo)
	Expect(err).ToNot(HaveOccurred())

	// HACK: we need to fix dagger and the package.sh scripts so that this isn't required
	npmURI = fmt.Sprintf("%s.tgz", npmURI)
	npmCachedURI = fmt.Sprintf("%s.tgz", npmCachedURI)
	nodeCachedURI = fmt.Sprintf("%s.tgz", nodeCachedURI)

	defer dagger.DeleteBuildpack(npmURI)
	defer dagger.DeleteBuildpack(npmCachedURI)
	defer dagger.DeleteBuildpack(nodeURI)
	defer os.RemoveAll(nodeRepo)
	defer dagger.DeleteBuildpack(nodeCachedURI)

	SetDefaultEventuallyTimeout(5 * time.Second)

	suite := spec.New("Integration", spec.Random(), spec.Report(report.Terminal{}))
	suite("Caching", testCaching)
	suite("EmptyNodeModules", testEmptyNodeModules, spec.Parallel())
	suite("Logging", testLogging, spec.Parallel())
	suite("NoNodeModules", testNoNodeModules, spec.Parallel())
	suite("PrePostScriptsRebuild", testPrePostScriptRebuild, spec.Parallel())
	suite("SimpleApp", testSimpleApp, spec.Parallel())
	suite("UnmetDependencies", testUnmetDependencies, spec.Parallel())
	suite("Vendored", testVendored, spec.Parallel())
	suite("VendoredWithBinaries", testVendoredWithBinaries, spec.Parallel())
	suite("Versioning", testVersioning, spec.Parallel())

	dagger.SyncParallelOutput(func() { suite.Run(t) })
}

func ContainerLogs(id string) func() string {
	docker := occam.NewDocker()

	return func() string {
		logs, _ := docker.Container.Logs.Execute(id)
		return logs.String()
	}
}

func GetBuildLogs(raw string) []string {
	var lines []string
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "[builder]") {
			lines = append(lines, strings.TrimPrefix(line, "[builder] "))
		}
	}

	return lines
}

func GetGitVersion() (string, error) {
	gitExec := pexec.NewExecutable("git")
	stdout := bytes.NewBuffer(nil)
	err := gitExec.Execute(pexec.Execution{
		Args:   []string{"describe", "--abbrev=0", "--tags"},
		Stdout: stdout,
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(strings.TrimPrefix(stdout.String(), "v")), nil
}

func ContainSequence(expected interface{}) types.GomegaMatcher {
	return &containSequenceMatcher{
		expected: expected,
	}
}

type containSequenceMatcher struct {
	expected interface{}
}

func (matcher *containSequenceMatcher) Match(actual interface{}) (success bool, err error) {
	if reflect.TypeOf(actual).Kind() != reflect.Slice {
		return false, errors.New("not a slice")
	}

	expectedLength := reflect.ValueOf(matcher.expected).Len()
	actualLength := reflect.ValueOf(actual).Len()
	for i := 0; i < (actualLength - expectedLength + 1); i++ {
		aSlice := reflect.ValueOf(actual).Slice(i, i+expectedLength)
		eSlice := reflect.ValueOf(matcher.expected).Slice(0, expectedLength)

		match := true
		for j := 0; j < eSlice.Len(); j++ {
			aValue := aSlice.Index(j)
			eValue := eSlice.Index(j)

			if eMatcher, ok := eValue.Interface().(types.GomegaMatcher); ok {
				m, err := eMatcher.Match(aValue.Interface())
				if err != nil {
					return false, err
				}

				if !m {
					match = false
				}
			} else if !reflect.DeepEqual(aValue.Interface(), eValue.Interface()) {
				match = false
			}
		}

		if match {
			return true, nil
		}
	}

	return false, nil
}

func (matcher *containSequenceMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to contain sequence", matcher.expected)
}

func (matcher *containSequenceMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to contain sequence", matcher.expected)
}
