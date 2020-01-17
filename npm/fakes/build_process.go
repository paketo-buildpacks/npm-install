package fakes

import "sync"

type BuildProcess struct {
	RunCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			LayerDir   string
			CacheDir   string
			WorkingDir string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string, string) error
	}
}

func (f *BuildProcess) Run(param1 string, param2 string, param3 string) error {
	f.RunCall.Lock()
	defer f.RunCall.Unlock()
	f.RunCall.CallCount++
	f.RunCall.Receives.LayerDir = param1
	f.RunCall.Receives.CacheDir = param2
	f.RunCall.Receives.WorkingDir = param3
	if f.RunCall.Stub != nil {
		return f.RunCall.Stub(param1, param2, param3)
	}
	return f.RunCall.Returns.Error
}
