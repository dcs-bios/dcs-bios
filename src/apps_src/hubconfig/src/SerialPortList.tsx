import React from 'react';

import { w3cwebsocket as W3CWebSocket } from "websocket";
import { getApiConnection } from './ApiConnection'

import './SerialPorts.css';

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
      let toggleConnectButton = (
      <button onClick={(e) => this.updatePortPref({
                  shouldBeConnected: !this.props.portState.shouldBeConnected,
                  autoConnect: this.props.portState.autoConnect
                  })}
              className="serial-port-connect-disconnect-button"
          >{this.props.portState.shouldBeConnected ? "Disconnect" : "Connect"} {this.props.portName}</button>)
  
      let connectOnStartupCheckbox = (
        <span className="autoconnect-checkbox"><input type="checkbox"
        checked={this.props.portState.autoConnect}
        onChange={(e) => {this.updatePortPref({
          shouldBeConnected: this.props.portState.shouldBeConnected,
          autoConnect: !this.props.portState.autoConnect
        }
        )}}
        ></input>connect automatically</span>
      )
  
      let stateClasses = [];
      if (this.props.portState.autoConnect) stateClasses.push("autoconnect-enabled");

      var connectedState: any = ""
      var connectedClass: string = ""
      let shouldBeConnected = this.props.portState.shouldBeConnected
      let isConnected = this.props.portState.isConnected
      
      
      if (shouldBeConnected && isConnected) {
        connectedState = <b>connected</b>;
        stateClasses.push("state-connected");
      }
      if (shouldBeConnected && (!isConnected)) {
        connectedState = "connecting..."
        stateClasses.push("state-connecting");
      }
      if ((!shouldBeConnected) && isConnected) {
        connectedState = "disconnecting..."
        stateClasses.push("state-disconnecting");
      }
      if ((!shouldBeConnected) && (!isConnected)) {
        connectedState = "not connected"
        stateClasses.push("state-disconnected");
      }
      if (!this.props.portState.isPresent) {
        connectedState = "missing"
        stateClasses.push("state-missing");
      }

      return (
        <div className={"serial-port "+stateClasses.join(' ')}>
        <b className="serial-port-name">{this.props.portName}</b><br/>
        <span className={"serial-port-state "+stateClasses.join(' ')}>{connectedState}</span><br/>
          {toggleConnectButton}<br/>
          {connectOnStartupCheckbox}<br/>
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
