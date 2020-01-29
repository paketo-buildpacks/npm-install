module github.com/cloudfoundry/npm-cnb

require (
	code.cloudfoundry.org/lager v2.0.0+incompatible
	github.com/cloudfoundry/dagger v0.0.0-20200115142400-b69a9b4eabf4
	github.com/cloudfoundry/libcfbuildpack v1.91.22 // indirect
	github.com/cloudfoundry/occam v0.0.0-something-d8c017e53fe355918763302bd802e40fee551b64
	github.com/cloudfoundry/packit v0.0.0-20200117181238-c9fbc0a623ec
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/mattn/go-runewidth v0.0.8 // indirect
	github.com/onsi/gomega v1.8.1
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sclevine/spec v1.4.0
	golang.org/x/sys v0.0.0-20200124204421-9fbb57f87de9 // indirect
)

replace github.com/cloudfoundry/occam => /Users/pivotal/workspace/occam

go 1.13
