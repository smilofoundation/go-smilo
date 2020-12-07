import {AppState} from 'src/types/v2'
import {push} from 'react-router-redux'

import {Task} from 'src/types/v2/tasks'
import {
  submitNewTask,
  updateTaskFlux,
  getUserTasks,
  getTask,
  updateTaskStatus as updateTaskStatusAPI,
  deleteTask as deleteTaskAPI,
} from 'src/tasks/api/v2'
import {getMe} from 'src/shared/apis/v2/user'
import {notify} from 'src/shared/actions/notifications'
import {
  taskNotCreated,
  tasksFetchFailed,
  taskDeleteFailed,
  taskNotFound,
  taskUpdateFailed,
} from 'src/shared/copy/v2/notifications'

export type Action =
  | SetNewScript
  | SetTasks
  | SetSearchTerm
  | SetCurrentScript
  | SetCurrentTask
  | SetShowInactive
  | SetDropdownOrgID

type GetStateFunc = () => AppState

export enum ActionTypes {
  SetNewScript = 'SET_NEW_SCRIPT',
  SetTasks = 'SET_TASKS',
  SetSearchTerm = 'SET_TASKS_SEARCH_TERM',
  SetCurrentScript = 'SET_CURRENT_SCRIPT',
  SetCurrentTask = 'SET_CURRENT_TASK',
  SetShowInactive = 'SET_TASKS_SHOW_INACTIVE',
  SetDropdownOrgID = 'SET_DROPDOWN_ORG_ID',
}

export interface SetNewScript {
  type: ActionTypes.SetNewScript
  payload: {
    script: string
  }
}
export interface SetCurrentScript {
  type: ActionTypes.SetCurrentScript
  payload: {
    script: string
  }
}
export interface SetCurrentTask {
  type: ActionTypes.SetCurrentTask
  payload: {
    task: Task
  }
}

export interface SetTasks {
  type: ActionTypes.SetTasks
  payload: {
    tasks: Task[]
  }
}

export interface SetSearchTerm {
  type: ActionTypes.SetSearchTerm
  payload: {
    searchTerm: string
  }
}

export interface SetShowInactive {
  type: ActionTypes.SetShowInactive
  payload: {}
}

export interface SetDropdownOrgID {
  type: ActionTypes.SetDropdownOrgID
  payload: {
    dropdownOrgID: string
  }
}

export const setNewScript = (script: string): SetNewScript => ({
  type: ActionTypes.SetNewScript,
  payload: {script},
})

export const setCurrentScript = (script: string): SetCurrentScript => ({
  type: ActionTypes.SetCurrentScript,
  payload: {script},
})

export const setCurrentTask = (task: Task): SetCurrentTask => ({
  type: ActionTypes.SetCurrentTask,
  payload: {task},
})

export const setTasks = (tasks: Task[]): SetTasks => ({
  type: ActionTypes.SetTasks,
  payload: {tasks},
})

export const setSearchTerm = (searchTerm: string): SetSearchTerm => ({
  type: ActionTypes.SetSearchTerm,
  payload: {searchTerm},
})

export const setShowInactive = (): SetShowInactive => ({
  type: ActionTypes.SetShowInactive,
  payload: {},
})

export const setDropdownOrgID = (dropdownOrgID: string): SetDropdownOrgID => ({
  type: ActionTypes.SetDropdownOrgID,
  payload: {dropdownOrgID},
})

export const updateTaskStatus = (task: Task) => async (
  dispatch,
  getState: GetStateFunc
) => {
  try {
    const {
      links: {tasks: url},
    } = getState()
    await updateTaskStatusAPI(url, task.id, task.status)

    dispatch(populateTasks())
  } catch (e) {
    console.error(e)
    dispatch(notify(taskDeleteFailed()))
  }
}

export const deleteTask = (task: Task) => async (
  dispatch,
  getState: GetStateFunc
) => {
  try {
    const {
      links: {tasks: url},
    } = getState()

    await deleteTaskAPI(url, task.id)

    dispatch(populateTasks())
  } catch (e) {
    console.error(e)
    dispatch(notify(taskDeleteFailed()))
  }
}

export const populateTasks = () => async (
  dispatch,
  getState: GetStateFunc
): Promise<void> => {
  try {
    const {
      orgs,
      links: {tasks: url, me: meUrl},
    } = getState()

    const user = await getMe(meUrl)
    const tasks = await getUserTasks(url, user)

    const mappedTasks = tasks.map(task => {
      return {
        ...task,
        organization: orgs.find(org => org.id === task.organizationId),
      }
    })

    dispatch(setTasks(mappedTasks))
  } catch (e) {
    console.error(e)
    dispatch(notify(tasksFetchFailed()))
  }
}

export const selectTaskByID = (id: string) => async (
  dispatch,
  getState: GetStateFunc
): Promise<void> => {
  try {
    const {
      links: {tasks: url},
    } = getState()

    const task = await getTask(url, id)

    return dispatch(setCurrentTask(task))
  } catch (e) {
    console.error(e)
    dispatch(goToTasks())
    dispatch(notify(taskNotFound()))
  }
}

export const selectTask = (task: Task) => async dispatch => {
  dispatch(push(`/tasks/${task.id}`))
}

export const goToTasks = () => async dispatch => {
  dispatch(push('/tasks'))
}

export const cancelUpdateTask = () => async dispatch => {
  dispatch(setCurrentTask(null))
  dispatch(goToTasks())
}

export const updateScript = () => async (dispatch, getState: GetStateFunc) => {
  try {
    const {
      links: {tasks: url},
      tasks: {currentScript: script, currentTask: task},
    } = getState()

    await updateTaskFlux(url, task.id, script)

    dispatch(setCurrentTask(null))
    dispatch(goToTasks())
  } catch (e) {
    console.error(e)
    dispatch(notify(taskUpdateFailed()))
  }
}

export const saveNewScript = () => async (
  dispatch,
  getState: GetStateFunc
): Promise<void> => {
  try {
    const {
      orgs,
      links: {tasks: url, me: meUrl},
      tasks: {newScript: script},
    } = getState()

    const user = await getMe(meUrl)

    await submitNewTask(url, user, orgs[0], script)

    dispatch(setNewScript(''))
    dispatch(goToTasks())
  } catch (e) {
    console.error(e)
    dispatch(notify(taskNotCreated(e.headers['x-influx-error'])))
  }
}
