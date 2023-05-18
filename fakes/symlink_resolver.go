package fakes

import (
	"sync"

	npminstall "github.com/paketo-buildpacks/npm-install"
)

type SymlinkResolver struct {
	CopyCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			LockfilePath    string
			SourceLayerPath string
			TargetLayerPath string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string, string) error
	}
	ParseLockfileCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			LockfilePath string
		}
		Returns struct {
			Lockfile npminstall.Lockfile
			Error    error
		}
		Stub func(string) (npminstall.Lockfile, error)
	}
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

func (f *SymlinkResolver) Copy(param1 string, param2 string, param3 string) error {
	f.CopyCall.mutex.Lock()
	defer f.CopyCall.mutex.Unlock()
	f.CopyCall.CallCount++
	f.CopyCall.Receives.LockfilePath = param1
	f.CopyCall.Receives.SourceLayerPath = param2
	f.CopyCall.Receives.TargetLayerPath = param3
	if f.CopyCall.Stub != nil {
		return f.CopyCall.Stub(param1, param2, param3)
	}
	return f.CopyCall.Returns.Error
}
func (f *SymlinkResolver) ParseLockfile(param1 string) (npminstall.Lockfile, error) {
	f.ParseLockfileCall.mutex.Lock()
	defer f.ParseLockfileCall.mutex.Unlock()
	f.ParseLockfileCall.CallCount++
	f.ParseLockfileCall.Receives.LockfilePath = param1
	if f.ParseLockfileCall.Stub != nil {
		return f.ParseLockfileCall.Stub(param1)
	}
	return f.ParseLockfileCall.Returns.Lockfile, f.ParseLockfileCall.Returns.Error
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
