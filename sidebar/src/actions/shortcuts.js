import { push } from 'react-router-redux';

export const handleShortcuts = dispatch => (action, event) => {
  switch (action) {
  case 'SEARCH':
    return dispatch(push('/'));
  default:
  }
};
