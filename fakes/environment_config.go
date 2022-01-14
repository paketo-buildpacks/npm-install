package fakes

import (
	"sync"

	"github.com/paketo-buildpacks/packit/v2"
)

type EnvironmentConfig struct {
	ConfigureCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Layer packit.Layer
		}
		Returns struct {
			Error error
		}
		Stub func(packit.Layer) error
	}
	GetValueCall struct {
		sync.Mutex
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

func (f *EnvironmentConfig) Configure(param1 packit.Layer) error {
	f.ConfigureCall.Lock()
	defer f.ConfigureCall.Unlock()
	f.ConfigureCall.CallCount++
	f.ConfigureCall.Receives.Layer = param1
	if f.ConfigureCall.Stub != nil {
		return f.ConfigureCall.Stub(param1)
	}
	return f.ConfigureCall.Returns.Error
}
func (f *EnvironmentConfig) GetValue(param1 string) string {
	f.GetValueCall.Lock()
	defer f.GetValueCall.Unlock()
	f.GetValueCall.CallCount++
	f.GetValueCall.Receives.Key = param1
	if f.GetValueCall.Stub != nil {
		return f.GetValueCall.Stub(param1)
	}
	return f.GetValueCall.Returns.String
}
