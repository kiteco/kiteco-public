/*
  Previously, we used Segment to route metrics to Mixpanel and Customer.io.
  Now, we override Segment's analytics.js to directly call Mixpanel and Customer.io's API.
*/
import { Store, Action, Dispatch } from "redux";
import { LOCATION_CHANGE } from "connected-react-router";
import * as mixpanel from "./mixpanel";

export interface ITrackEvent {
  event: string;
  props?: any;
}

declare var _cio: any;
const SOURCE = "webapp";

const getCurrentTime = () => {
  const d = new Date();
  return Math.floor(d.getTime() / 1000);
};

export const track = ({ event, props = null }: ITrackEvent) => {
  if (process.env.REACT_APP_ENV === "development") {
    console.log(`tracking ${event}`, props);
  }

  _cio.track(event, {
    source: SOURCE,
    sent_at: getCurrentTime(),
    ...props
  });

  mixpanel.track({
    event,
    props: {
      source: SOURCE,
      sent_at: getCurrentTime(),
      user_id: mixpanel.get_distinct_id(),
      ...props
    }
  });
};

export const identify = (id: string): void => {
  _cio.identify({ id });
  mixpanel.identify(id);
};

export const reset = () => {
  mixpanel.reset();
};

export const alias = (id: string) => {
  // Not supported by Customer.io
  mixpanel.alias(id);
};

export const analyticsMiddleware = (store: Store<any>) => (next: Dispatch<Action>) => (action: any) => {
  if (action.type === LOCATION_CHANGE) {
    // Manually implement Segment's analytics.page() event for backwards compatability.
    // See https://segment.com/docs/spec/page/#properties for Schema
    const { pathname: path, search } = action.payload;
    const { referrer, title } = document;
    const url = window.location.href;
    track({
      event: "Loaded a Page",
      props: { path, search, referrer, title, url }
    });
  }

  if (!action.meta || !action.meta.analytics || !action.meta.analytics.event) {
    return next(action);
  }

  try {
    const { event, props } = action.meta.analytics;
    const submission = {
      accountStatus: store.getState().account.status,
      ...props
    };
    track({ event: `webapp: ${event}`, props: submission });
  } catch (error) { }
  return next(action);
};

export function getTrackingParamsFromURL(targetUrl: string, click: string): string {
  const url: string = window.location.href;

  if (!url.includes(targetUrl)) return '';

  return `?cta_source=${encodeURIComponent(targetUrl)}&cta_content=${encodeURIComponent(url.split(targetUrl)[1])}&click_cta=${encodeURIComponent(click)}`;
}
