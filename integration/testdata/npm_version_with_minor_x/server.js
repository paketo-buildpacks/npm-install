var port = Number(process.env.PORT || 8080);
var express = require("express");
var logfmt = require("logfmt");
var app = express();

app.use(logfmt.requestLogger());

const
    { spawnSync } = require( 'child_process' ),
    vers = spawnSync( 'npm', [ '-v' ] );

app.listen(port, function() {
  console.log("Listening on " + port);
});

app.get('/', function(req, res) {
  res.send('Hello, World! From npm version: ' + vers.stdout.toString());
});

