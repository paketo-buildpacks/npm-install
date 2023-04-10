package fakes

import (
	"sync"

	npminstall "github.com/paketo-buildpacks/npm-install"
)

type Symlinker struct {
	LinkCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Source string
			Target string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string) error
	}
	WithPathCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Path string
		}
		Returns struct {
			Symlinker npminstall.Symlinker
		}
		Stub func(string) npminstall.Symlinker
	}
}

func (f *Symlinker) Link(param1 string, param2 string) error {
	f.LinkCall.mutex.Lock()
	defer f.LinkCall.mutex.Unlock()
	f.LinkCall.CallCount++
	f.LinkCall.Receives.Source = param1
	f.LinkCall.Receives.Target = param2
	if f.LinkCall.Stub != nil {
		return f.LinkCall.Stub(param1, param2)
	}
	return f.LinkCall.Returns.Error
}
func (f *Symlinker) WithPath(param1 string) npminstall.Symlinker {
	f.WithPathCall.mutex.Lock()
	defer f.WithPathCall.mutex.Unlock()
	f.WithPathCall.CallCount++
	f.WithPathCall.Receives.Path = param1
	if f.WithPathCall.Stub != nil {
		return f.WithPathCall.Stub(param1)
	}
	return f.WithPathCall.Returns.Symlinker
}
