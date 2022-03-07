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
			CacheDir   string
		}
		Returns struct {
			BuildProcess npminstall.BuildProcess
			Error        error
		}
		Stub func(string, string) (npminstall.BuildProcess, error)
	}
}

func (f *BuildManager) Resolve(param1 string, param2 string) (npminstall.BuildProcess, error) {
	f.ResolveCall.mutex.Lock()
	defer f.ResolveCall.mutex.Unlock()
	f.ResolveCall.CallCount++
	f.ResolveCall.Receives.WorkingDir = param1
	f.ResolveCall.Receives.CacheDir = param2
	if f.ResolveCall.Stub != nil {
		return f.ResolveCall.Stub(param1, param2)
	}
	return f.ResolveCall.Returns.BuildProcess, f.ResolveCall.Returns.Error
}
