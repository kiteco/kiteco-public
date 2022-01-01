import { Action, Dispatch } from 'redux'
import { ThunkDispatch, ThunkMiddleware, ThunkAction } from 'redux-thunk'
import { CHECK_IF_ONLINE } from '../actions/system'

interface ISocketHandlers {
   onMessage: (m: MessageEvent) => void,
   onOpen: (e: Event) => void,
   onClose: (c: CloseEvent) => void
}

export type OnMessageDispatch = (response: object) =>
                                (dispatch: ThunkDispatch<object, any, Action>) =>
                                ThunkAction<object, any, any, Action>

// Socket exposes middleware to use with redux
export default class MiddlewareSocket {
  private _onMessageDispatch: OnMessageDispatch
  private _url: string
  private _socket: WebSocket | null
  private _timerId: number | null

  constructor(url: string, onMessageDispatch: OnMessageDispatch) {
    this._onMessageDispatch = onMessageDispatch
    this._url = url
    this._socket = null
    this._timerId = null
  }

  get middleware(): ThunkMiddleware {
    return (store: { dispatch: ThunkDispatch<object, any, Action> }) => (next: Dispatch<Action>) => (action: Action) => {
      const socketHandlers = {
        onMessage: (event: MessageEvent) => {
          const response = JSON.parse(event.data)
          store.dispatch(this._onMessageDispatch(response))
        },
        onClose: (_: CloseEvent) => {
          if (this._timerId) {
            clearTimeout(this._timerId)
            this._timerId = null
          }
          //reestablish connection
          this._timerId = window.setTimeout(() => {
            this._open(socketHandlers)
          }, 5000)
        },
      } as ISocketHandlers
      switch (action.type) {
        // User request to connect
        case CHECK_IF_ONLINE:
          if (!this._socket || (this._socket.readyState !== WebSocket.OPEN && this._socket.readyState !== WebSocket.CONNECTING)) {
            this._open(socketHandlers)
          }
          break
        default:
          break
      }
      return next(action)
    }
  }

  _close = () => {
    if (this._socket) {
      this._socket.close()
      this._socket = null
    }
  }

 _open = (handlers: ISocketHandlers) => {
   if (this._socket) {
     this._close()
   }
   this._socket = new WebSocket(this._url)
   this._socket.onmessage = handlers.onMessage
   this._socket.onopen = handlers.onOpen
   this._socket.onclose = handlers.onClose
 }
}
