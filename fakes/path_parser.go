package fakes

import "sync"

type PathParser struct {
	GetCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Path string
		}
		Returns struct {
			ProjectPath string
			Err         error
		}
		Stub func(string) (string, error)
	}
}

func (f *PathParser) Get(param1 string) (string, error) {
	f.GetCall.Lock()
	defer f.GetCall.Unlock()
	f.GetCall.CallCount++
	f.GetCall.Receives.Path = param1
	if f.GetCall.Stub != nil {
		return f.GetCall.Stub(param1)
	}
	return f.GetCall.Returns.ProjectPath, f.GetCall.Returns.Err
}
