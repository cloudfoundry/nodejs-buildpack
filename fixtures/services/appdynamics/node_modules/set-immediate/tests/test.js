typeof require === 'undefined' ? load('../setImmediate.js') : require('../setImmediate.js');

var s = false;

function printResult () {
	var msg = [].slice.call(arguments).join(' ');

	typeof print === 'undefined' ? console.log(msg) : print(msg);
}

var start = +new Date;

setImmediate(function (s) {
	printResult('It worked and took', +new Date - s, 'milliseconds.');
	s = true;
}, [start]);

var canceling = setImmediate(function () {
	printResult('ERROR: this should not have appeared.');
	s = false;
});

printResult('Attempting to cancel immediate with id:', canceling);

clearImmediate(canceling);

setTimeout(function () {
	s || process.exit(1);
}, 100);
