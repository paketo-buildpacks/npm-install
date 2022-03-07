package fakes

import "sync"

type EnvironmentConfig struct {
	GetValueCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Key string
		}
		Returns struct {
			String string
		}
		Stub func(string) string
	}
}

func (f *EnvironmentConfig) GetValue(param1 string) string {
	f.GetValueCall.mutex.Lock()
	defer f.GetValueCall.mutex.Unlock()
	f.GetValueCall.CallCount++
	f.GetValueCall.Receives.Key = param1
	if f.GetValueCall.Stub != nil {
		return f.GetValueCall.Stub(param1)
	}
	return f.GetValueCall.Returns.String
}
