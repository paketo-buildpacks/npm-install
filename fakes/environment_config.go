package fakes

import "sync"

type EnvironmentConfig struct {
	LookupCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Key string
		}
		Returns struct {
			Value string
			Found bool
		}
		Stub func(string) (string, bool)
	}
	LookupBoolCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Key string
		}
		Returns struct {
			Bool  bool
			Error error
		}
		Stub func(string) (bool, error)
	}
}

func (f *EnvironmentConfig) Lookup(param1 string) (string, bool) {
	f.LookupCall.mutex.Lock()
	defer f.LookupCall.mutex.Unlock()
	f.LookupCall.CallCount++
	f.LookupCall.Receives.Key = param1
	if f.LookupCall.Stub != nil {
		return f.LookupCall.Stub(param1)
	}
	return f.LookupCall.Returns.Value, f.LookupCall.Returns.Found
}
func (f *EnvironmentConfig) LookupBool(param1 string) (bool, error) {
	f.LookupBoolCall.mutex.Lock()
	defer f.LookupBoolCall.mutex.Unlock()
	f.LookupBoolCall.CallCount++
	f.LookupBoolCall.Receives.Key = param1
	if f.LookupBoolCall.Stub != nil {
		return f.LookupBoolCall.Stub(param1)
	}
	return f.LookupBoolCall.Returns.Bool, f.LookupBoolCall.Returns.Error
}
