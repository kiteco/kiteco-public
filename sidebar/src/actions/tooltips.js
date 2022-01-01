/* ===={ TOOLTIPS }==== */

export const SHOW_TOOLTIP = "show tooltip"
export const showToolTip = ({ kind, bounds, data }) => ({
  type: SHOW_TOOLTIP,
  kind,
  bounds,
  data,
})

export const HIDE_TOOLTIP = "hide tooltip"
export const hideToolTip = ({ kind, bounds, data }) => ({
  type: HIDE_TOOLTIP,
  kind,
  bounds,
  data,
})
