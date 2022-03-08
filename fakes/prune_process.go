package fakes

import "sync"

type PruneProcess struct {
	RunCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			ModulesDir string
			CacheDir   string
			WorkingDir string
			NpmrcPath  string
			Launch     bool
		}
		Returns struct {
			Error error
		}
		Stub func(string, string, string, string, bool) error
	}
	ShouldRunCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir string
			Metadata   map[string]interface {
			}
			NpmrcPath string
		}
		Returns struct {
			Run bool
			Sha string
			Err error
		}
		Stub func(string, map[string]interface {
		}, string) (bool, string, error)
	}
}

func (f *PruneProcess) Run(param1 string, param2 string, param3 string, param4 string, param5 bool) error {
	f.RunCall.mutex.Lock()
	defer f.RunCall.mutex.Unlock()
	f.RunCall.CallCount++
	f.RunCall.Receives.ModulesDir = param1
	f.RunCall.Receives.CacheDir = param2
	f.RunCall.Receives.WorkingDir = param3
	f.RunCall.Receives.NpmrcPath = param4
	f.RunCall.Receives.Launch = param5
	if f.RunCall.Stub != nil {
		return f.RunCall.Stub(param1, param2, param3, param4, param5)
	}
	return f.RunCall.Returns.Error
}
func (f *PruneProcess) ShouldRun(param1 string, param2 map[string]interface {
}, param3 string) (bool, string, error) {
	f.ShouldRunCall.mutex.Lock()
	defer f.ShouldRunCall.mutex.Unlock()
	f.ShouldRunCall.CallCount++
	f.ShouldRunCall.Receives.WorkingDir = param1
	f.ShouldRunCall.Receives.Metadata = param2
	f.ShouldRunCall.Receives.NpmrcPath = param3
	if f.ShouldRunCall.Stub != nil {
		return f.ShouldRunCall.Stub(param1, param2, param3)
	}
	return f.ShouldRunCall.Returns.Run, f.ShouldRunCall.Returns.Sha, f.ShouldRunCall.Returns.Err
}
