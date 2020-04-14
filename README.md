# NPM Cloud Native Buildpack

The NPM CNB makes use of the [`npm`](https://www.npmjs.com/) tooling installed
within the [Node Engine CNB](https://github.com/paketo-buildpacks/node-engine)
to manage application dependencies, then writes the start command for the given
application.

## Integration

The NPM CNB provides node_modules as a dependency. Downstream buildpacks can
require the node dependency by generating a [Build Plan
TOML](https://github.com/buildpacks/spec/blob/master/buildpack.md#build-plan-toml)
file that looks like the following:

```toml
[[requires]]

  # The name of the NPM dependency is "node_modules". This value is considered
  # part of the public API for the buildpack and will not change without a plan
  # for deprecation.
  name = "node_modules"

  # Note: The version field is unsupported as there is no version for a set of
  # npm.

  # The NPM buildpack supports some non-required metadata options.
  [requires.metadata]

    # Setting the build flag to true will ensure that the Node modules are
    # available for subsequent buildpacks during their build phase.
    # If you are writing a buildpack that needs to run a node module during its build
    # process, this flag should be set to true.
    build = true

    # Setting the launch flag to true will ensure that the packages managed by
    # NPM will be available for the running application. If you
    # are writing an application that needs to run Node modules at runtime, this
    # flag should be set to true.
    launch = true
```

## Usage

To package this buildpack for consumption:

```
$ ./scripts/package.sh
```

This builds the buildpack's Go source using `GOOS=linux` by default. You can supply another value as the first argument to `package.sh`.