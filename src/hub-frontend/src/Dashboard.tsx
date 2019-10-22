import React from 'react';
import SerialPortList from './SerialPortList'
import { ControlReferenceIndex } from './ControlReference'

import { ConnectionStatus } from './Status'

export default function Dashboard() {
    return (
        <React.Fragment>
            <h1>DCS-BIOS Hub</h1>
            <ConnectionStatus />
            <ControlReferenceIndex showInstalledOnly />
            <h2>Serial Ports</h2>
            <SerialPortList />
        </React.Fragment>
    );
}