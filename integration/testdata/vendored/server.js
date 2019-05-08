const http = require('http');
const port = process.env.PORT || 8080;
const leftpad = require('leftpad');

const requestHandler = (request, response) => {
  response.end("Hello, World!");
};

const server = http.createServer(requestHandler);

server.listen(port, (err) => {
  if (err) {
    return console.log('something bad happened', err);
  }

  console.log(`vendored server is listening on ${port}`);
});
