package fakes

import "sync"

type SymlinkResolver struct {
	ResolveCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			LockfilePath string
			LayerPath    string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string) error
	}
}

func (f *SymlinkResolver) Resolve(param1 string, param2 string) error {
	f.ResolveCall.mutex.Lock()
	defer f.ResolveCall.mutex.Unlock()
	f.ResolveCall.CallCount++
	f.ResolveCall.Receives.LockfilePath = param1
	f.ResolveCall.Receives.LayerPath = param2
	if f.ResolveCall.Stub != nil {
		return f.ResolveCall.Stub(param1, param2)
	}
	return f.ResolveCall.Returns.Error
}
