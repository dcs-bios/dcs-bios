import React from 'react';

import { w3cwebsocket as W3CWebSocket } from "websocket";
import { getApiConnection } from './ApiConnection'

type PortState = {
    shouldBeConnected: boolean,
    autoConnect: boolean,
    isConnected: boolean,
    isPresent: boolean
}

type PortPreference = {
  shouldBeConnected: boolean
  autoConnect: boolean
}

type SerialPortProps = {
  portName: string
  portState: PortState
}

class SerialPort extends React.Component<SerialPortProps, {}> {
    updatePortPref(newState: PortPreference) {
      let ws = getApiConnection()
      ws.onopen = () => {
        ws.send(JSON.stringify({
          datatype:"set_port_pref",
          data: {
            portName: this.props.portName,
            shouldBeConnected: newState.shouldBeConnected,
            autoConnect: newState.autoConnect
          }
        }))
        ws.close()
      }
    }
  
    render() {
      let toggleConnectButton = (<button onClick={(e) => this.updatePortPref({
        shouldBeConnected: !this.props.portState.shouldBeConnected,
        autoConnect: this.props.portState.autoConnect
      })}>{this.props.portState.shouldBeConnected ? "Disconnect" : "Connect"} {this.props.portName}</button>)
  
      let connectOnStartupCheckbox = (
        <input type="checkbox" 
        checked={this.props.portState.autoConnect}
        onChange={(e) => {this.updatePortPref({
          shouldBeConnected: this.props.portState.shouldBeConnected,
          autoConnect: !this.props.portState.autoConnect
        }
        )}}
        ></input>
      )
  
      var connectedState
      let shouldBeConnected = this.props.portState.shouldBeConnected
      let isConnected = this.props.portState.isConnected
  
      if (shouldBeConnected && isConnected) {
        connectedState = <b>connected</b>
      }
      if (shouldBeConnected && (!isConnected)) {
        connectedState = "connecting..."
      }
      if ((!shouldBeConnected) && isConnected) {
        connectedState = "disconnecting..."
      }
      if ((!shouldBeConnected) && (!isConnected)) {
        connectedState = "not connected"
      }
      if (!this.props.portState.isPresent) {
        connectedState = "missing"
      }

      return (
        <div style={{border: "1px solid gray", padding: "3px"}}>
        <b>{this.props.portName}</b> ({connectedState})<br/>
          {toggleConnectButton}<br/>
          {connectOnStartupCheckbox}connect automatically<br/>
        </div>
      )
    }
  }
  
type SerialPortListState = {
  ports: Map<string, PortState>
}

  class SerialPortList extends React.Component<{}, SerialPortListState> {
    private monitorPortsWebsocket: W3CWebSocket | undefined

    constructor(props: any) {
      super(props)
      this.state = {
        ports: new Map<string, PortState>()
      }
    }
  
  componentDidMount() {
    this.monitorPortsWebsocket = getApiConnection()
    
    this.monitorPortsWebsocket.onopen = () => {
      if (!this.monitorPortsWebsocket) return;
      this.monitorPortsWebsocket.send(JSON.stringify({
        datatype: "monitor_serial_ports",
        data: {}
      }))
    }
    this.monitorPortsWebsocket.onmessage = msg => {
      msg = JSON.parse(msg.data)
      let newPortStates = new Map<string, PortState>();
      let sortedPortNames = Object.keys(msg.data);
      sortedPortNames.sort()
      for (let portName of sortedPortNames) {
        newPortStates.set(portName, msg.data[portName] as PortState)
      }
      this.setState({
        ports: newPortStates
      })
    }
  }
  componentWillUnmount() {
    if (!this.monitorPortsWebsocket) return;
    this.monitorPortsWebsocket.close()
  }
  
    render() {
      let sortedPortNames = Object.keys(this.state.ports)
      sortedPortNames.sort()
      return (<div>
        <ul>
          {Array.from(this.state.ports.entries()).map(
            ([portName, portState]) => <SerialPort key={portName} portName={portName} portState={portState}/>
          )}
        </ul>
      </div>);
    }
  }

  export default SerialPortList
