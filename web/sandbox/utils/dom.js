export function onRemove(element, onDetachCallback) {
  const observer = new MutationObserver(function () {
      function isDetached(el) {
          if (el.parentNode === document) {
              return false;
          } else if (el.parentNode === null) {
              return true;
          } else {
              return isDetached(el.parentNode);
          }
      }

      if (isDetached(element)) {
          observer.disconnect();
          onDetachCallback();
      }
  })

  observer.observe(document, {
       childList: true,
       subtree: true
  });
}