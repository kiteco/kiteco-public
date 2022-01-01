// Global event system

(function() {

function Dispatcher() {

  this.listeners = {};

  // registers a callback function to call when an event named `eventName` is triggered.
  // returns an id which can be passed to the `deregister` function
  this.register = function(eventName, callback) {
    if (typeof this.listeners[eventName] === 'undefined') {
      this.listeners[eventName] = [];
    }
    var callbackIndex;
    // attempt to find a null in the this.listeners[eventName] array
    for (callbackIndex = 0;  callbackIndex < this.listeners[eventName].length;  callbackIndex++) {
      if (this.listeners[eventName][callbackIndex] === null) {
        // populate a null spot with callback & return this callbackIndex
        this.listeners[eventName][callbackIndex] = callback;
        return {
          eventName: eventName,
          callbackIndex: callbackIndex
        }
      }
    }
    // if we didn't find an empty spot in this.listeners[eventName], just
    // append the callback
    this.listeners[eventName].push(callback);
    return {
      eventName: eventName,
      callbackIndex: this.listeners[eventName].length - 1
    };
  };

  // pass the object returned by register into deregister to remove the callback
  this.deregister = function(callbackId) {
    this.listeners[callbackId.eventName][callbackId.callbackIndex] = null;
  };

  this.trigger = function(eventName, obj) {
    if (typeof this.listeners[eventName] != 'undefined') {
      this.listeners[eventName].forEach(function(callback) {
        if (callback != null) {
          callback(obj);
        }
      });
    }
  };
}

if (typeof module !== 'undefined' && module.exports) { // node (for tests)
  module.exports.Dispatcher = Dispatcher;
} else if (typeof window !== 'undefined') { // browser
  window.Dispatcher = Dispatcher;
} else {
  console.log("I have no idea what environment we are running in.");
}

})();
