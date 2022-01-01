/* WEBDOCS DOMAIN */

import { Domains } from './domains'

const webdocsDomain = `https://${Domains.DocsAPI}`;

/* EDITOR */

export const membersPath = (identifier, page, limit, type = "symbol") => {
  return [
    `${webdocsDomain}/api/editor/value/${identifier}/members`,
    [`offset=${page}`, `limit=${limit}`, `type=${type}`].join("&")
  ].join("?");
};

export const definitionSourcePath = id => {
  return `${webdocsDomain}/api/editor/symbol/${id}/definition-source`;
};

export const symbolPath = id => {
  return `${webdocsDomain}/api/editor/symbol/${id}`;
};

export const sandboxCompletionsPath = id => {
  return `${webdocsDomain}/api/websandbox/completions${id ? `?id=${id}` : ""}`;
};

export const signaturedCompletionsPath = id => {
  return `${webdocsDomain}/api/websandbox/signatured-completions${id ? `?id=${id}` : ""}`;
};

/* SEARCH + QUERIES */
const DEFAULT_OFFSET = 0;
export const DEFAULT_QUERY_LIMIT = 6;
export const queryCompletionPath = ({
  query,
  limit = DEFAULT_QUERY_LIMIT,
  offset = DEFAULT_OFFSET
}) => `${webdocsDomain}/api/editor/search?q=${query}&offset=${offset}&limit=${limit}`;

/* EXAMPLES */
export const examplesPath = (language, id) => {
  return `${webdocsDomain}/api/${language}/curation/${id}`;
};

export const multipleExamplesPath = (language, ids) => {
  return `${webdocsDomain}/api/${language}/curation/examples?id=${ids.join(",")}`;
};

/* KITE ANSWERS */
export const answersPath = slug => {
  // TODO: Swap python for languages var
  return `${webdocsDomain}/api/python/answers/${slug}`;
};

/* USAGES */
const DEFAULT_USAGES_OFFSET = 0;
const DEFAULT_USAGES_LIMIT = 999;
export const usagesPath = (
  identifier,
  offset = DEFAULT_USAGES_OFFSET,
  limit = DEFAULT_USAGES_LIMIT
) => {
  return `${webdocsDomain}/api/editor/value/${identifier}/usages?offset=${offset}&limit=${limit}`;
};

/* ACCOUNTS */
export const signupsPath = "/api/signups";

export const userAccountPath = "/api/account/user";
export const createAccountPath = "/api/account/create-web";

export const pricingPath = "/api/account/pricing";
export const calcPricePath = "/api/account/calculate-price";

export const loginPath = "/api/account/login-web";
export const logoutPath = "/api/account/logout";

export const checkEmailPath = "/api/account/check-email";
export const checkPasswordPath = "/api/account/check-password";

export const passwordResetPath = action => {
  return `/api/account/reset-password/${action}`;
};

export const verifyEmailPath = "/api/account/verify-email";
export const unsubscribePath = "/unsubscribe";
export const invitePath = medium => {
  return `/api/account/invite-${medium}`;
};
export const forumLoginPath = "/api/account/forum-login";

export const commentPath = "/api/account/comment";


/* NEWSLETTER */
export const newsletterSignUp = "/api/account/newsletter";

/* MOBILE DOWNLOAD */
export const mobileDownloadPath = "/api/account/mobile-download";

/* EMAIL VERIFICATION */
export const emailVerificationPath = "/api/account/verify-newsletter";

/* PYCON SIGNUP */
export const pyconSignupPath = "/api/account/pycon-signup";


/* LICENSING */
export const licenseInfoPath = "/api/account/license-info";
export const subscriptionsPath = "/api/account/subscriptions";
