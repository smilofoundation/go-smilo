import React, {Component, MouseEvent} from 'react'
import ReactDOM from 'react-dom'

// Components
import {Dropdown} from 'src/clockface'
import timePoints from 'src/logs/data/timePoints'
import {ErrorHandling} from 'src/shared/decorators/errors'
import {ClickOutside} from 'src/shared/components/ClickOutside'
import CustomSingularTime from 'src/shared/components/custom_singular_time/CustomSingularTime'

interface Props {
  customTime?: string
  relativeTime?: number
  onChooseCustomTime: (time: string) => void
  onChooseRelativeTime: (time: number) => void
}

interface State {
  isTimeSelectorOpen: boolean
}

@ErrorHandling
class TimeRangeDropdown extends Component<Props, State> {
  constructor(props) {
    super(props)

    this.state = {
      isTimeSelectorOpen: false,
    }
  }

  public render() {
    return (
      <div style={{display: 'inline'}}>
        <Dropdown
          selectedID={this.timeInputValue}
          onChange={this.handleSelection}
          widthPixels={100}
          customClass="time-range-dropdown logs-viewer--search-dropdown"
        >
          {this.timeItems}
        </Dropdown>
        {this.timeSelector}
      </div>
    )
  }

  private get timeItems(): JSX.Element[] {
    const relativeItems = timePoints.map(({text, value}) => (
      <Dropdown.Item key={text} value={value} id={text}>
        {text}
      </Dropdown.Item>
    ))

    return [this.customTimeItem, ...relativeItems]
  }

  private get customTimeItem(): JSX.Element {
    return (
      <Dropdown.Item key="custom" value={this.props.customTime} id="custom">
        <div onClick={this.handleOpenCustomTime}>Date Picker</div>
      </Dropdown.Item>
    )
  }

  private get timeSelector(): JSX.Element {
    const portalElement = document.querySelector(
      '.logs-viewer--search-dropdown'
    )

    if (!this.state.isTimeSelectorOpen || !portalElement) {
      return null
    }

    const overlay = (
      <ClickOutside onClickOutside={this.handleCloseCustomTime}>
        <div className="custom-time--overlay">
          <CustomSingularTime
            onSelected={this.handleCustomSelection}
            time={this.props.customTime}
          />
        </div>
      </ClickOutside>
    )

    return ReactDOM.createPortal(overlay, portalElement)
  }

  private handleCustomSelection = (time: string) => {
    this.handleCloseCustomTime()
    this.props.onChooseCustomTime(time)
    this.setState({isTimeSelectorOpen: false})
  }

  private handleSelection = (timeValue: number) => {
    this.props.onChooseRelativeTime(timeValue)
  }

  private get timeInputValue(): string {
    if (!this.props.customTime) {
      const point = timePoints.find(p => p.value === this.props.relativeTime)
      if (point) {
        return point.text
      }

      return 'now'
    }

    return 'custom'
  }

  private handleCloseCustomTime = () => {
    this.setState({isTimeSelectorOpen: false})
  }

  private handleOpenCustomTime = (e: MouseEvent<HTMLElement>) => {
    e.preventDefault()
    e.stopPropagation()

    this.setState({isTimeSelectorOpen: true})
  }
}
export default TimeRangeDropdown
