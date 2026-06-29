const express = require("express");
const logfmt = require("logfmt");
const app = express();
app.use(logfmt.requestLogger());

app.get('/', (req, res) => {
  res.send('Hello, World!');
});

const port = Number(process.env.PORT || 5000);
app.listen(port, () => {
  console.log("Listening on " + port);
});
