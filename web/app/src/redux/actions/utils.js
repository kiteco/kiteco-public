
/**
 * helper method to create an action
 * that also sends to analytics
 */
export const record = (name) => ({
  type: name,
  meta: {
    analytics: {
      event: name,
    },
  },
});
