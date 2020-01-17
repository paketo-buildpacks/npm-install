const http = require('http');
const port = process.env.PORT || 8080;
const sqlite3 = require('sqlite3').verbose();
const childProcess = require('child_process');

const requestHandler = (request, response) => {
  childProcess.exec('cat preinstall.log && cat postinstall.log', {}, function(err, stdout, stderr) {
    if (err) {
      response.end(err);
    }

    response.end(stdout);
  });
};

const server = http.createServer(requestHandler);

server.listen(port, (err) => {
  if (err) {
    return console.log('something bad happened', err);
  }

  console.log(`vendored server is listening on ${port}`);
});
