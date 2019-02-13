var port = Number(process.env.PORT || 8080);
var express = require("express");
var logfmt = require("logfmt");
var app = express();

app.use(logfmt.requestLogger());

app.listen(port, function() {
  console.log("Listening on " + port);
});

app.get('/', function(req, res) {
  res.send('Hello, World!');
});
