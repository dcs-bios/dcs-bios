import React, { useState, useEffect } from 'react';
import SerialPortList from './SerialPortList'
import { ControlReferenceIndex } from './ControlReference'
import SetupUI from './SetupUI';
import { ConnectionStatus } from './Status'

export default function Dashboard() {
    return (
        <React.Fragment>
            <h1>DCS-BIOS Hub</h1>
            <ConnectionStatus />
            <h2>Serial Ports</h2>
            <SerialPortList />
            <ControlReferenceIndex showInstalledOnly />
            <SetupUI />
        </React.Fragment>
    );
}