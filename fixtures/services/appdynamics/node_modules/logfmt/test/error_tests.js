var logfmt = require('../logfmt'),
    assert = require('assert');

var logfmt = new logfmt;
var OutStream = require('./outstream');

suite('logfmt.error', function() {
  test('logs an error', function() {
    var err = new Error('testing');
    logfmt.stream = new(require('./outstream'));
    logfmt.error(err);
    var id = logfmt.stream.lines[0].match(/id=(\d+)/)[1];
    var line1 = logfmt.parse(logfmt.stream.lines[0])
    assert.equal(line1.error, true);
    assert.equal(line1.id, id);
    assert(line1.now);
    assert.equal(line1.message, 'testing');

    var line2 = logfmt.parse(logfmt.stream.lines[1])
    assert.equal(line2.error, true);
    assert.equal(line2.id, id);
    assert(line2.now);
    assert.equal(line2.line, '0');
    assert.equal(line2.trace, 'Error: testing');
  });

  test('sends only a max number of log lines', function() {
    var err = new Error('testing');
    logfmt.stream = new(require('./outstream'));
    logfmt.maxErrorLines = 2;
    logfmt.error(err);
    assert.equal(logfmt.stream.lines.length, 3);
  });

  test("doesn't blow up on a bad error object", function(){
    logfmt.error({})
  })
})
