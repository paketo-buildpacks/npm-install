package fakes

import "sync"

type ConfigurationManager struct {
	DeterminePathCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Typ         string
			PlatformDir string
			Entry       string
		}
		Returns struct {
			Path string
			Err  error
		}
		Stub func(string, string, string) (string, error)
	}
}

func (f *ConfigurationManager) DeterminePath(param1 string, param2 string, param3 string) (string, error) {
	f.DeterminePathCall.mutex.Lock()
	defer f.DeterminePathCall.mutex.Unlock()
	f.DeterminePathCall.CallCount++
	f.DeterminePathCall.Receives.Typ = param1
	f.DeterminePathCall.Receives.PlatformDir = param2
	f.DeterminePathCall.Receives.Entry = param3
	if f.DeterminePathCall.Stub != nil {
		return f.DeterminePathCall.Stub(param1, param2, param3)
	}
	return f.DeterminePathCall.Returns.Path, f.DeterminePathCall.Returns.Err
}
