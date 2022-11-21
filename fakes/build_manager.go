package fakes

import (
	"sync"

	npminstall "github.com/paketo-buildpacks/npm-install"
)

type BuildManager struct {
	ResolveCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir string
		}
		Returns struct {
			BuildProcess npminstall.BuildProcess
			Bool         bool
			Error        error
		}
		Stub func(string) (npminstall.BuildProcess, bool, error)
	}
}

func (f *BuildManager) Resolve(param1 string) (npminstall.BuildProcess, bool, error) {
	f.ResolveCall.mutex.Lock()
	defer f.ResolveCall.mutex.Unlock()
	f.ResolveCall.CallCount++
	f.ResolveCall.Receives.WorkingDir = param1
	if f.ResolveCall.Stub != nil {
		return f.ResolveCall.Stub(param1)
	}
	return f.ResolveCall.Returns.BuildProcess, f.ResolveCall.Returns.Bool, f.ResolveCall.Returns.Error
}
