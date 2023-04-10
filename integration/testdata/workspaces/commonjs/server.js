const http = require('http');
const leftpad = require('leftpad');
const greeting = require('@paketo-buildpacks/hello-world');

const server = http.createServer((request, response) => {
  switch (request.url) {
    case '/process':
      response.end(JSON.stringify(process.env))
      break;

    default:
      response.end(greeting);
  }
});

const port = process.env.PORT || 8080;
server.listen(port, (err) => {
  if (err) {
    return console.log('something bad happened', err);
  }

  console.log(`NOT vendored server is listening on ${port}`);
});
