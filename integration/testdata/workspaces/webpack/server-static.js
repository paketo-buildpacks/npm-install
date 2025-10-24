const http = require("http");
const fs = require("fs");
const path = require("path");

const PORT = process.env.PORT || 8080;
const STATIC_CONTENT_DIR = path.join(__dirname, process.env.STATIC_CONTENT_DIR || "build");

const server = http.createServer((req, res) => {
  let filePath = path.join(STATIC_CONTENT_DIR, req.url === "/" ? "index.html" : req.url);

  fs.readFile(filePath, (_, content) => {
    res.writeHead(200, { "Content-Type": getContentType(filePath) });
    res.end(content, "utf-8");
  });
});

server.listen(PORT);

function getContentType(filePath) {
  const extname = path.extname(filePath);
  switch (extname) {
    case ".html":
      return "text/html";
    case ".js":
      return "text/javascript";
    case ".css":
      return "text/css";
    case ".json":
      return "application/json";
    case ".png":
      return "image/png";
    case ".jpg":
      return "image/jpg";
    default:
      return "text/html";
  }
}
