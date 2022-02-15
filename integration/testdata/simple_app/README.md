This file here to suppress "npm WARN package.json node_web_app@0.0.0 No README data"

This app contains a development dependency (`spellchecker-cli`), which is only
used to test `npmrc` service binding support. The dependency will only be installed
if `NODE_ENV=development` or an `.npmrc` config file explicitly includes dev dependencies.
