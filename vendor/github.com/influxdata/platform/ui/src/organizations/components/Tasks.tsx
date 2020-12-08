// Libraries
import React, {PureComponent, ChangeEvent} from 'react'

// Components
import ProfilePageHeader from 'src/shared/components/profile_page/ProfilePageHeader'
import {Input, IconFont, ComponentSize, EmptyState} from 'src/clockface'
import TaskList from 'src/organizations/components/TaskList'
import FilterList from 'src/organizations/components/Filter'

// Types
import {Task, TaskStatus} from 'src/types/v2/tasks'

interface Props {
  tasks: Task[]
}

interface State {
  searchTerm: string
}

export default class Tasks extends PureComponent<Props, State> {
  constructor(props) {
    super(props)
    this.state = {
      searchTerm: '',
    }
  }

  public render() {
    const {searchTerm} = this.state

    return (
      <>
        <ProfilePageHeader>
          <Input
            icon={IconFont.Search}
            placeholder="Filter tasks..."
            widthPixels={290}
            value={searchTerm}
            onChange={this.handleFilterChange}
            onBlur={this.handleFilterBlur}
          />
        </ProfilePageHeader>
        <FilterList<Task>
          searchTerm={searchTerm}
          searchKeys={['name', 'owner.name']}
          list={this.tempTasks}
        >
          {ts => <TaskList tasks={ts} emptyState={this.emptyState} />}
        </FilterList>
      </>
    )
  }

  // TODO: use real tasks
  private get tempTasks(): Task[] {
    return [
      {
        every: '2m',
        id: '1',
        name: 'swoggles task',
        status: TaskStatus.Active,
        organizationId: '1',
        organization: {
          id: '1',
          name: 'swoggle org',
        },
        owner: {
          id: '1',
          name: 'watts',
        },
      },
      {
        every: '2m',
        id: '2',
        name: 'plerps task',
        status: TaskStatus.Active,
        organizationId: '1',
        organization: {
          id: '1',
          name: 'plerps org',
        },
        owner: {
          id: '1',
          name: 'alexxx',
        },
      },
    ]
  }

  private handleFilterBlur = (e: ChangeEvent<HTMLInputElement>): void => {
    this.setState({searchTerm: e.target.value})
  }

  private handleFilterChange = (e: ChangeEvent<HTMLInputElement>): void => {
    this.setState({searchTerm: e.target.value})
  }

  private get emptyState(): JSX.Element {
    return (
      <EmptyState size={ComponentSize.Medium}>
        <EmptyState.Text text="I see nay a task" />
      </EmptyState>
    )
  }
}
