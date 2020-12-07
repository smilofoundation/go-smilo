// Constants
import {StepStatus} from 'src/clockface/constants/wizard'

// Types
import {SetupParams} from 'src/onboarding/apis'
import {DataSource} from 'src/types/v2/dataSources'

export type Action =
  | SetSetupParams
  | IncrementCurrentStepIndex
  | DecrementCurrentStepIndex
  | SetCurrentStepIndex
  | SetStepStatus
  | AddDataSource
  | RemoveDataSource

interface SetSetupParams {
  type: 'SET_SETUP_PARAMS'
  payload: {setupParams: SetupParams}
}

export const setSetupParams = (setupParams: SetupParams): SetSetupParams => ({
  type: 'SET_SETUP_PARAMS',
  payload: {setupParams},
})

interface SetCurrentStepIndex {
  type: 'SET_CURRENT_STEP_INDEX'
  payload: {index: number}
}

export const setCurrentStepIndex = (index: number): SetCurrentStepIndex => ({
  type: 'SET_CURRENT_STEP_INDEX',
  payload: {index},
})

interface IncrementCurrentStepIndex {
  type: 'INCREMENT_CURRENT_STEP_INDEX'
}

export const incrementCurrentStepIndex = (): IncrementCurrentStepIndex => ({
  type: 'INCREMENT_CURRENT_STEP_INDEX',
})

interface DecrementCurrentStepIndex {
  type: 'DECREMENT_CURRENT_STEP_INDEX'
}

export const decrementCurrentStepIndex = (): DecrementCurrentStepIndex => ({
  type: 'DECREMENT_CURRENT_STEP_INDEX',
})

interface SetStepStatus {
  type: 'SET_STEP_STATUS'
  payload: {index: number; status: StepStatus}
}

export const setStepStatus = (
  index: number,
  status: StepStatus
): SetStepStatus => ({
  type: 'SET_STEP_STATUS',
  payload: {
    index,
    status,
  },
})

interface AddDataSource {
  type: 'ADD_DATA_SOURCE'
  payload: {dataSource: DataSource}
}

export const addDataSource = (dataSource: DataSource): AddDataSource => ({
  type: 'ADD_DATA_SOURCE',
  payload: {dataSource},
})

interface RemoveDataSource {
  type: 'REMOVE_DATA_SOURCE'
  payload: {dataSource: string}
}

export const removeDataSource = (dataSource: string): RemoveDataSource => ({
  type: 'REMOVE_DATA_SOURCE',
  payload: {dataSource},
})
