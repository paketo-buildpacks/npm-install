package fakes

import "sync"

type Concat struct {
	ConcatCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Files []string
		}
		Returns struct {
			String string
			Error  error
		}
		Stub func(...string) (string, error)
	}
}

func (f *Concat) Concat(param1 ...string) (string, error) {
	f.ConcatCall.Lock()
	defer f.ConcatCall.Unlock()
	f.ConcatCall.CallCount++
	f.ConcatCall.Receives.Files = param1
	if f.ConcatCall.Stub != nil {
		return f.ConcatCall.Stub(param1...)
	}
	return f.ConcatCall.Returns.String, f.ConcatCall.Returns.Error
}
