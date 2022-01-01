/*
  Wrapper for Mixpanel analytics client.
*/
import mixpanel from "mixpanel-browser";
import { ITrackEvent } from "./analytics";

const TOKENS = {
  live: "XXXXXXX",
  test: "XXXXXXX"
};

const getMixpanelToken = () => {
  if (process.env.REACT_APP_ENV === "development") {
    return TOKENS.test;
  }
  if (process.env.NODE_ENV === "production") {
    return TOKENS.live;
  }
  return TOKENS.test;
};

mixpanel.init(
  getMixpanelToken(),
  {
    "ignore_dnt": true,
    "api_host": 'https://kiteweblibs.b-cdn.net',
  },
);

export const get_distinct_id = () => {
  mixpanel.get_distinct_id();
};

export const track = ({ event, props }: ITrackEvent) => {
  mixpanel.track(event, props);
};

export const identify = (id: string): void => {
  mixpanel.identify(id);
};

export const reset = () => {
  mixpanel.reset();
};

export const alias = (id: string): void => {
  mixpanel.alias(id);
};
