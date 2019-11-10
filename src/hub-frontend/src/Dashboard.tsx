import React from 'react';
import SerialPortList from './SerialPortList'

import { ConnectionStatus } from './Status'
import { ReloadScripts, ScriptList } from './ScriptList';

export default function Dashboard() {
    return (
        <React.Fragment>
            <h1>DCS-BIOS Hub</h1>
            <ConnectionStatus />
            <h2>Serial Ports</h2>
            <SerialPortList />
            <h2>Lua Scripting</h2>
            <ReloadScripts /><br/>
            <ScriptList/>
        </React.Fragment>
    );
}