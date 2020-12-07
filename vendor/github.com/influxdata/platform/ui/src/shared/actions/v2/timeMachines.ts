// Types
import {TimeMachineState} from 'src/shared/reducers/v2/timeMachines'
import {TimeRange, ViewType} from 'src/types/v2'
import {Axes, DecimalPlaces} from 'src/types/v2/dashboards'
import {Color} from 'src/types/colors'

export type Action =
  | SetActiveTimeMachineAction
  | SetNameAction
  | SetTimeRangeAction
  | SetTypeAction
  | SetDraftScriptAction
  | SubmitScriptAction
  | SetDecimalPlaces
  | SetAxes
  | SetStaticLegend
  | SetColors
  | SetYAxisLabel
  | SetYAxisMinBound
  | SetYAxisMaxBound
  | SetYAxisPrefix
  | SetYAxisSuffix
  | SetYAxisBase
  | SetYAxisScale

interface SetActiveTimeMachineAction {
  type: 'SET_ACTIVE_TIME_MACHINE'
  payload: {
    activeTimeMachineID: string
    initialState: Partial<TimeMachineState>
  }
}

export const setActiveTimeMachine = (
  activeTimeMachineID: string,
  initialState: Partial<TimeMachineState> = {}
): SetActiveTimeMachineAction => ({
  type: 'SET_ACTIVE_TIME_MACHINE',
  payload: {activeTimeMachineID, initialState},
})

interface SetNameAction {
  type: 'SET_VIEW_NAME'
  payload: {name: string}
}

export const setName = (name: string): SetNameAction => ({
  type: 'SET_VIEW_NAME',
  payload: {name},
})

interface SetTimeRangeAction {
  type: 'SET_TIME_RANGE'
  payload: {timeRange: TimeRange}
}

export const setTimeRange = (timeRange: TimeRange): SetTimeRangeAction => ({
  type: 'SET_TIME_RANGE',
  payload: {timeRange},
})

interface SetTypeAction {
  type: 'SET_VIEW_TYPE'
  payload: {type: ViewType}
}

export const setType = (type: ViewType): SetTypeAction => ({
  type: 'SET_VIEW_TYPE',
  payload: {type},
})

interface SetDraftScriptAction {
  type: 'SET_DRAFT_SCRIPT'
  payload: {draftScript: string}
}

export const setDraftScript = (draftScript: string): SetDraftScriptAction => ({
  type: 'SET_DRAFT_SCRIPT',
  payload: {draftScript},
})

interface SubmitScriptAction {
  type: 'SUBMIT_SCRIPT'
}

export const submitScript = (): SubmitScriptAction => ({
  type: 'SUBMIT_SCRIPT',
})
interface SetAxes {
  type: 'SET_AXES'
  payload: {axes: Axes}
}

export const setAxes = (axes: Axes): SetAxes => ({
  type: 'SET_AXES',
  payload: {axes},
})

interface SetYAxisLabel {
  type: 'SET_Y_AXIS_LABEL'
  payload: {label: string}
}

export const setYAxisLabel = (label: string): SetYAxisLabel => ({
  type: 'SET_Y_AXIS_LABEL',
  payload: {label},
})

interface SetYAxisMinBound {
  type: 'SET_Y_AXIS_MIN_BOUND'
  payload: {min: string}
}

export const setYAxisMinBound = (min: string): SetYAxisMinBound => ({
  type: 'SET_Y_AXIS_MIN_BOUND',
  payload: {min},
})

interface SetYAxisMaxBound {
  type: 'SET_Y_AXIS_MAX_BOUND'
  payload: {max: string}
}

export const setYAxisMaxBound = (max: string): SetYAxisMaxBound => ({
  type: 'SET_Y_AXIS_MAX_BOUND',
  payload: {max},
})

interface SetYAxisPrefix {
  type: 'SET_Y_AXIS_PREFIX'
  payload: {prefix: string}
}

export const setYAxisPrefix = (prefix: string): SetYAxisPrefix => ({
  type: 'SET_Y_AXIS_PREFIX',
  payload: {prefix},
})

interface SetYAxisSuffix {
  type: 'SET_Y_AXIS_SUFFIX'
  payload: {suffix: string}
}

export const setYAxisSuffix = (suffix: string): SetYAxisSuffix => ({
  type: 'SET_Y_AXIS_SUFFIX',
  payload: {suffix},
})

interface SetYAxisBase {
  type: 'SET_Y_AXIS_BASE'
  payload: {base: string}
}

export const setYAxisBase = (base: string): SetYAxisBase => ({
  type: 'SET_Y_AXIS_BASE',
  payload: {base},
})

interface SetYAxisScale {
  type: 'SET_Y_AXIS_SCALE'
  payload: {scale: string}
}

export const setYAxisScale = (scale: string): SetYAxisScale => ({
  type: 'SET_Y_AXIS_SCALE',
  payload: {scale},
})

interface SetStaticLegend {
  type: 'SET_STATIC_LEGEND'
  payload: {staticLegend: boolean}
}

export const setStaticLegend = (staticLegend: boolean): SetStaticLegend => ({
  type: 'SET_STATIC_LEGEND',
  payload: {staticLegend},
})

interface SetColors {
  type: 'SET_COLORS'
  payload: {colors: Color[]}
}

export const setColors = (colors: Color[]): SetColors => ({
  type: 'SET_COLORS',
  payload: {colors},
})

interface SetDecimalPlaces {
  type: 'SET_DECIMAL_PLACES'
  payload: {decimalPlaces: DecimalPlaces}
}

export const setDecimalPlaces = (
  decimalPlaces: DecimalPlaces
): SetDecimalPlaces => ({
  type: 'SET_DECIMAL_PLACES',
  payload: {decimalPlaces},
})
