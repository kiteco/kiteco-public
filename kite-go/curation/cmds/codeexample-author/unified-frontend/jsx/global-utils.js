global.map = (x, f) => {
  if (typeof x === 'undefined' || x === null) {
    return null;
  } else if (typeof x.map === 'function') {
    return x.map(f);
  } else {
    return f(x);
  }
};

global.maybeCall = function (f) {
  return typeof f === 'function' ? f.apply(null, Array.prototype.slice(arguments, 1)) : undefined;
};

// from https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Array/includes
if (![].includes) {
  Array.prototype.includes = function(searchElement /*, fromIndex*/ ) {
    'use strict';
    var O = Object(this);
    var len = parseInt(O.length) || 0;
    if (len === 0) {
      return false;
    }
    var n = parseInt(arguments[1]) || 0;
    var k;
    if (n >= 0) {
      k = n;
    } else {
      k = len + n;
      if (k < 0) {k = 0;}
    }
    var currentElement;
    while (k < len) {
      currentElement = O[k];
      if (searchElement === currentElement ||
         (searchElement !== searchElement && currentElement !== currentElement)) {
        return true;
      }
      k++;
    }
    return false;
  };
}
