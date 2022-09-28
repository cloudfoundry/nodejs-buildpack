const express = require('express');
const app = express();
const config = require('@sample/sample-config');

app.get('/check', (req, res) => {
    res.send({
        config: config(),
    });
});

const port = process.env.PORT || 3000;

app.listen(port, () => console.log(`Sample app listening on port ${ port }!`));