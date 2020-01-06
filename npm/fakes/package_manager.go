package fakes

import "sync"

type PackageManager struct {
	InstallCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Dir string
		}
		Returns struct {
			Error error
		}
		Stub func(string) error
	}
}

func (f *PackageManager) Install(param1 string) error {
	f.InstallCall.Lock()
	defer f.InstallCall.Unlock()
	f.InstallCall.CallCount++
	f.InstallCall.Receives.Dir = param1
	if f.InstallCall.Stub != nil {
		return f.InstallCall.Stub(param1)
	}
	return f.InstallCall.Returns.Error
}
