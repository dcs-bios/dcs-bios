import React, { useState, useEffect } from 'react';

import { getApiConnection } from './ApiConnection'

import './SerialPorts.css';
import { StatusIndicator } from './Status';

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

type SerialPortListState = {
  ports: Map<string, PortState>
}

function describeState(portState: PortState) {
  let connectedState: string = ""
  let stateClasses = []
  if (portState.autoConnect) {
    stateClasses.push("autoconnect-enabled")
  } else {
    stateClasses.push("autoconnect-disabled")
  }
  const { shouldBeConnected, isConnected } = portState
  if (shouldBeConnected && isConnected) {
    connectedState = "connected";
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
  if (!portState.isPresent) {
    connectedState = "missing"
    stateClasses.push("state-missing");
  }
  return { connectedState, stateClasses }
}

function SerialPortList() {
  const [ ports, setPorts ] = useState<Map<string, PortState>>(() => new Map<string, PortState>())

  useEffect(() => {
    const monitorPortsWebsocket = getApiConnection()
    
    monitorPortsWebsocket.onopen = () => {
      monitorPortsWebsocket.send(JSON.stringify({
        datatype: "monitor_serial_ports",
        data: {}
      }))
    }
    monitorPortsWebsocket.onmessage = msg => {
      msg = JSON.parse(msg.data)
      let newPortStates = new Map<string, PortState>();
      let sortedPortNames = Object.keys(msg.data);
      sortedPortNames.sort()
      for (let portName of sortedPortNames) {
        newPortStates.set(portName, msg.data[portName] as PortState)
      }
      setPorts(newPortStates)
    }
    return () => monitorPortsWebsocket.close()
  }, [])

  const updatePortPref = (portName: string, newState: PortPreference) => {
    let ws = getApiConnection()
    ws.onopen = () => {
      ws.send(JSON.stringify({
        datatype:"set_port_pref",
        data: {
          portName: portName,
          shouldBeConnected: newState.shouldBeConnected,
          autoConnect: newState.autoConnect
        }
      }))
      ws.close()
    }
  }

  const disconnectAll = () => {
    ports.forEach((port, portName) => {
      updatePortPref(
          portName,
         {
          autoConnect: port.autoConnect,
          shouldBeConnected: false
        }
      )
    });
  }
  const reconnectAutoPorts = () => {
    ports.forEach((port, portName) => {
      if (!port.autoConnect) return;
      updatePortPref(
          portName,
         {
          autoConnect: port.autoConnect,
          shouldBeConnected: true
        }
      )
    });
    
  }

  let sortedPortNames: Array<string> = Array.from(ports.keys())
  sortedPortNames.sort()

  return (
    <div>
      <button onClick={disconnectAll}>Disconnect All</button>
      <button onClick={reconnectAutoPorts}>Connect All Auto</button>
      <br/><br/>

      <table className="serial-port-table">
        <tbody>
          <tr>
            <th>Name</th>
            <th>State</th>
            <th>Autoconnect?</th>
            <th></th>
          </tr>
          {sortedPortNames.map(portName => <SerialPortTableRow key={portName} portName={portName} portState={ports.get(portName) as PortState} updatePortPref={updatePortPref}/>)}
        </tbody>
      </table>
    </div>
  )
}

function SerialPortTableRow(props: {portName: string, portState: PortState, updatePortPref: (portName: string, newState: PortPreference) => void}) {
  const { portName, portState, updatePortPref } = props

  const { connectedState, stateClasses } = describeState(portState)

  return (
    <tr className={"serial-port-row "+stateClasses.join(" ")}>
      <td className="serial-port-name">{portName}</td>
      <td className={"serial-port-connection-state "+stateClasses.join(" ")}><StatusIndicator text={connectedState} active={portState.isConnected}/></td>
      <td className="serial-port-autoconnect" onClick={(e) => props.updatePortPref(portName, {
        shouldBeConnected: portState.shouldBeConnected,
        autoConnect: !portState.autoConnect
      })}>
        <input type="checkbox" readOnly checked={portState.autoConnect}/>
      </td>
      <td className="serial-port-button"><button onClick={(e) => updatePortPref(portName, {
                  shouldBeConnected: !portState.shouldBeConnected,
                  autoConnect: portState.autoConnect
                  })}
          >{portState.shouldBeConnected ? "Disconnect" : "Connect"} {portName}</button></td>
    </tr>
  )
}



export default SerialPortList
