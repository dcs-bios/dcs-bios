import React from 'react';
import SerialPortList from './SerialPortList'
import { ControlReferenceIndex } from './ControlReference'

import { ConnectionStatus } from './Status'
import { HooksLuaReload } from './HooksLuaReload';

export default function Dashboard() {
    return (
        <React.Fragment>
            <h1>DCS-BIOS Hub</h1>
            <ConnectionStatus />
            <ControlReferenceIndex showInstalledOnly />
            <h2>Serial Ports</h2>
            <SerialPortList />
            <h2>Lua Scripting</h2>
            <HooksLuaReload />
        </React.Fragment>
    );
}