# Paketo Buildpack for NPM Install Cloud Native

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
$ ./scripts/package.sh --version <version-number>
```

This will create a `buildpackage.cnb` file under the `build` directory which you
can use to build your app as follows:
`pack build <app-name> -p <path-to-app> -b <path/to/node-engine.cnb> -b build/buildpackage.cnb`

## Specifying a project path

To specify a project subdirectory to be used as the root of the app, please use
the `BP_NODE_PROJECT_PATH` environment variable at build time either directly
(e.g. `pack build my-app --env BP_NODE_PROJECT_PATH=./src/my-app`) or through a
[`project.toml`
file](https://github.com/buildpacks/spec/blob/main/extensions/project-descriptor.md).
This could be useful if your app is a part of a monorepo.

## Run Tests

To run all unit tests, run:
```
./scripts/unit.sh
```

To run all integration tests, run:
```
/scripts/integration.sh
```

## Stack support

For most apps, the NPM Install Buildpack runs fine on the [Base
builder](https://github.com/paketo-buildpacks/stacks#metadata-for-paketo-buildrun-stack-images).
But when the app requires compilation of native extensions using `node-gyp`,
the buildpack requires that you use the [Full
builder](https://github.com/paketo-buildpacks/stacks#metadata-for-paketo-buildrun-stack-images).
This is because `node-gyp` requires `python` that's absent on the Base builder,
and the module may require other shared objects.
