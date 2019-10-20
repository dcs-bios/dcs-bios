import React from 'react';
import SerialPortList from './SerialPortList'
import { ControlReferenceIndex } from './ControlReference'
import SetupUI from './SetupUI';

export default function Dashboard() {
    return (
        <React.Fragment>
            <h1>DCS-BIOS Hub</h1>
            Welcome to DCS-BIOS Hub v0.8.
            <h2>Serial Ports</h2>
            <SerialPortList />
            <ControlReferenceIndex showInstalledOnly/>
            <SetupUI/>
        </React.Fragment>
    );
}