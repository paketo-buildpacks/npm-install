package fakes

import "sync"

type ScriptsParser struct {
	ParseScriptsCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Path string
		}
		Returns struct {
			ScriptsMap map[string]string
			Err        error
		}
		Stub func(string) (map[string]string, error)
	}
}

func (f *ScriptsParser) ParseScripts(param1 string) (map[string]string, error) {
	f.ParseScriptsCall.Lock()
	defer f.ParseScriptsCall.Unlock()
	f.ParseScriptsCall.CallCount++
	f.ParseScriptsCall.Receives.Path = param1
	if f.ParseScriptsCall.Stub != nil {
		return f.ParseScriptsCall.Stub(param1)
	}
	return f.ParseScriptsCall.Returns.ScriptsMap, f.ParseScriptsCall.Returns.Err
}
