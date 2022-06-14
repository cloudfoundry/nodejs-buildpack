(function (global, prefixes, timeouts, id, i) {

if (!global.setImmediate && typeof process !== 'undefined' && process && process.nextTick) {
	global.setImmediate = function (callback, args) {
		var i = id;

		timeouts[id] = {
			c: callback,
			a: args || [],
		};

		process.nextTick(function () {
			var t = timeouts[i];
			if (t) {
				t.c.apply(global, t.a);
				delete timeouts[i];
			}
		});
		return id++;
	};

	global.clearImmediate = function (id) {
		timeouts[id] && delete timeouts[id];
	};
	return;
}

for (i=0; !global.setImmediate && i<prefixes.length; i++) {
	global.setImmediate	= global[prefixes[i] + 'SetImmediate'];
	global.clearImmediate	= global[prefixes[i] + 'ClearImmediate'];
}

if (!global.setImmediate && !global.importScripts || global.postMessage && global.addEventListener) {
	global.addEventListener('message', function (e) {
		if (!e.data || e.source !== global || !e.data.setImmediate) return;

		e.preventDefault && e.preventDefault();
		e.stopPropagation && e.stopPropagation();

		var t = timeouts[e.data.setImmediate];

		if (t) {
			t.c.apply(global, t.a);
			delete timeouts[e.data.setImmediate];
		}
	}, true);

	global.setImmediate = function (callback, args) {
		timeouts[id] = {
			c: callback,
			a: args || [],
		};
		global.postMessage({
			setImmediate: id,
		}, '*');
		return id++;
	};

	global.clearImmediate = function (id) {
		timeouts[id] && delete timeouts[id];
	};
	return;
}

if (!global.setImmediate && global.setInterval) {
	global.setImmediate = function (callback, args) {
		return global.setTimeout(callback, 0, args);
	};
	global.clearImmediate = global.clearTimeout;
	return;
}

}(typeof global !== 'undefined' && global ? global : this, ['webkit', 'moz', 'ms', 'o'], {}, +new Date));

