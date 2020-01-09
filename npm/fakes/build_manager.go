package fakes

import (
	"sync"

	"github.com/cloudfoundry/npm-cnb/npm"
)

type BuildManager struct {
	ResolveCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir string
		}
		Returns struct {
			BuildProcess npm.BuildProcess
			Error        error
		}
		Stub func(string) (npm.BuildProcess, error)
	}
}

func (f *BuildManager) Resolve(param1 string) (npm.BuildProcess, error) {
	f.ResolveCall.Lock()
	defer f.ResolveCall.Unlock()
	f.ResolveCall.CallCount++
	f.ResolveCall.Receives.WorkingDir = param1
	if f.ResolveCall.Stub != nil {
		return f.ResolveCall.Stub(param1)
	}
	return f.ResolveCall.Returns.BuildProcess, f.ResolveCall.Returns.Error
}
