import docs from './docs'
import * as actions from '../actions/docs'


describe('docs reducer', () => {
  it('should handle initial state', () => {
    expect(
      docs(undefined, {})
    ).toEqual({
        status: null,
        data: null,
      })
  })

  it('should handle SET_DOCS_STATUS', () => {
    expect(
      docs(undefined, {
        type: actions.SET_DOCS_STATUS,
        status: "status"
      })
    ).toEqual({
        status: "status",
        data: null
      })
  })

  it('should handle SHOW_DOCS', () => {
    expect(
      docs(undefined, {
        type: actions.SHOW_DOCS,
        data: {data: ""},
      })
    ).toEqual({
        status: null,
        data: {data: ""},
      })
  })
})
