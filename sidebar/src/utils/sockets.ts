import MiddlewareSocket, { OnMessageDispatch } from './socket'
import { activeFileEvent } from '../actions/active-file'
import { autosearchEvent } from '../actions/search'
import { relatedCodeEvent } from "../store/related-code/related-code"

const ACTIVEFILE_URL = "ws://localhost:46624/active-file"
const AUTOSEARCH_URL = "ws://localhost:46624/autosearch"
const RELATED_CODE_URL = "ws://localhost:46624/codenav/subscribe"

export const activefile = new MiddlewareSocket(ACTIVEFILE_URL, activeFileEvent as OnMessageDispatch)
export const autosearch = new MiddlewareSocket(AUTOSEARCH_URL, autosearchEvent as OnMessageDispatch)
export const related_code = new MiddlewareSocket(RELATED_CODE_URL, relatedCodeEvent as OnMessageDispatch)
