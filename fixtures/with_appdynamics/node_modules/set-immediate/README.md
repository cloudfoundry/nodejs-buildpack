# setImmediate

A simple and lightweight shim for the setImmediate [W3C Draft](https://dvcs.w3.org/hg/webperf/raw-file/tip/specs/setImmediate/Overview.html) API, for use in any browsers and NodeJS.

## Example usage

You can include the script any way you want in a browser, in NodeJS environment, it is suggested that you do so by installing ```set-immediate``` from npm, and then including it using ```require('set-immediate')```, preferably before any other dependencies, so that they can use it too (it is injected on the global object for browser < - > node compatibility).

```javascript

setImmediate(function (foo, bar) {
	console.log(foo, bar);
}, ['foo', 'bar']);

var id = setImmediate(function () {});

clearImmediate(id);

```
