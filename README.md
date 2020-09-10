# Paketo NPM Install Cloud Native Buildpack

The NPM Install CNB makes use of the [`npm`](https://www.npmjs.com/) tooling
installed within the [Node Engine CNB](https://github.com/paketo-buildpacks/node-engine)
to manage application dependencies.

## Integration

The NPM Install CNB provides `node_modules` as a dependency. Downstream
buildpacks can require the `node_modules` dependency by generating a [Build
Plan TOML](https://github.com/buildpacks/spec/blob/master/buildpack.md#build-plan-toml)
file that looks like the following:

```toml
[[requires]]

  # The name of the NPM Install dependency is "node_modules". This value is
  # considered part of the public API for the buildpack and will not change
  # without a plan for deprecation.
  name = "node_modules"

  # Note: The version field is unsupported as there is no version for a set of
  # npm.

  # The NPM Install buildpack supports some non-required metadata options.
  [requires.metadata]

    # Setting the build flag to true will ensure that the node modules are
    # available for subsequent buildpacks during their build phase.
    # If you are writing a buildpack that needs to run a node module during its build
    # process, this flag should be set to true.
    build = true

    # Setting the launch flag to true will ensure that the packages managed by
    # NPM will be available for the running application. If you
    # are writing an application that needs to run node modules at runtime, this
    # flag should be set to true.
    launch = true
```

## Usage

To package this buildpack for consumption:

```
$ ./scripts/package.sh
```

## `buildpack.yml` Configurations

The NPM Install buildpack does not support configurations using `buildpack.yml`.
