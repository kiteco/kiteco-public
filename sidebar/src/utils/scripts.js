import { load as loadSegment } from './analytics'
import { load as doorbellLoad } from './doorbell'

const SEGMENT_ANALYTICS = {
  name: 'segment',
  needsMetricsEnabled: true,
  loadFn: loadSegment
}
const DOORBELL = {
  name: 'doorbell',
  needsMetricsEnabled: false,
  loadFn: doorbellLoad
}

export const SCRIPTS = [
  SEGMENT_ANALYTICS,
  DOORBELL,
]
