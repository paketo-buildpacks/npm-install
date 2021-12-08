package fakes

import (
	"sync"
)

type BuildProcess struct {
	RunCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			ModulesDir string
			CacheDir   string
			WorkingDir string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string, string) error
	}
	ShouldRunCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir string
			Metadata   map[string]interface {
			}
		}
		Returns struct {
			Run bool
			Sha string
			Err error
		}
		Stub func(string, map[string]interface {
		}) (bool, string, error)
	}
}

func (f *BuildProcess) Run(param1 string, param2 string, param3 string) error {
	f.RunCall.mutex.Lock()
	defer f.RunCall.mutex.Unlock()
	f.RunCall.CallCount++
	f.RunCall.Receives.ModulesDir = param1
	f.RunCall.Receives.CacheDir = param2
	f.RunCall.Receives.WorkingDir = param3
	if f.RunCall.Stub != nil {
		return f.RunCall.Stub(param1, param2, param3)
	}
	return f.RunCall.Returns.Error
}
func (f *BuildProcess) ShouldRun(param1 string, param2 map[string]interface {
}) (bool, string, error) {
	f.ShouldRunCall.mutex.Lock()
	defer f.ShouldRunCall.mutex.Unlock()
	f.ShouldRunCall.CallCount++
	f.ShouldRunCall.Receives.WorkingDir = param1
	f.ShouldRunCall.Receives.Metadata = param2
	if f.ShouldRunCall.Stub != nil {
		return f.ShouldRunCall.Stub(param1, param2)
	}
	return f.ShouldRunCall.Returns.Run, f.ShouldRunCall.Returns.Sha, f.ShouldRunCall.Returns.Err
}
