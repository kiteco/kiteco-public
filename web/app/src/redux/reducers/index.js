import { combineReducers } from "redux";
import { connectRouter } from "connected-react-router";
import answers from "./answers";
import docs from "./docs";
import examples from "./examples";
import errors from "./errors";
import account from "./account";
import { reducer as license } from '../store/license'
import notifications from "./notifications";
import promotions from "./promotions";
import comments from "./comments";
import history from "./history";
import loading from "./loading";
import cachedCompletions from "./cached-completions";
import signinPopup from "./signin-popup";
import starred from "./starred";
import stylePopup from "./style-popup";
import style from "./style";
import usages from "./usages";

const reducer = (hist) => combineReducers({
  answers,
  comments,
  router: connectRouter(hist),
  notifications,
  docs,
  examples,
  errors,
  account,
  license,
  promotions,
  history,
  loading,
  cachedCompletions,
  signinPopup,
  starred,
  stylePopup,
  style,
  usages
});

export default reducer;
