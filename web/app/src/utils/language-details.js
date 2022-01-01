import {head} from './functional'

export const evalPath = (o, ...path) =>
  path.reduce((m, k) => {
    if (k === '*' && m) { k = head(Object.keys(m)); }
    return m && typeof m[k] !== 'undefined' ? m[k] : undefined;
  }, o);

export const detailLang = o =>
  o && o.language_details
    ? head(Object.keys(o.language_details)).toLowerCase()
    : 'python';

export const detailGet = (o, k) =>
  o[k] || evalPath(o, 'language_details', '*', k);

export const detailExist = (o, k) => detailGet(o, k) != null;

export const detailNotEmpty = (o, k) => {
  const v = detailGet(o, k);
  return v != null && v.length > 0;
};
