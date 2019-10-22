import React, { useState, useEffect } from 'react';
import { getApiConnection } from './ApiConnection';
import './Status.css'

type TStatusInfo = {
    version: string
    gitSHA1: string
    isDcsConnected: boolean
    isLuaConsoleConnected: boolean
    isLuaConsoleEnabled: boolean
    isExternalNetworkAccessEnabled: boolean
}

function useStatus() {
    let [status, setStatus] = useState<TStatusInfo>({
        version: "",
        gitSHA1: "",
        isDcsConnected: false,
        isLuaConsoleConnected: false,
        isLuaConsoleEnabled: false,
        isExternalNetworkAccessEnabled: false
    })

    useEffect(() => {
        const socket = getApiConnection()
        socket.onopen = () => {
            console.log("subscribing to status updates")
            socket.send(JSON.stringify({
                datatype: "get_status_updates",
                data: {}
            }))
        }
        socket.onmessage = (msg => {
            let json = JSON.parse(msg.data)
            if (json.datatype === "status_update") {
                console.log(json.data)
                setStatus(json.data)
            }
        })
    }, []);

    return status;

}

export function LuaConsoleStatus() {
    const status = useStatus();
    
    return (
        <div>
        <b>Status:</b>
        <span className={"status-indicator status-"+status.isLuaConsoleConnected}>DCS Connection</span>
        <span className={"status-indicator status-"+status.isLuaConsoleEnabled}>Enabled in Systray</span>
        </div>
    )
}

export function ConnectionStatus() {
    const status = useStatus();

    return (
        <div>
        <b>Connections:</b>
        <span className={"status-indicator status-"+status.isDcsConnected}>Virtual Cockpit</span>
        <span className={"status-indicator status-"+status.isLuaConsoleConnected}>Lua Console</span>
        
        <b style={{marginLeft: "3em"}}>Systray Settings:</b>
        <span className={"status-indicator status-"+status.isLuaConsoleEnabled}>Lua Console</span>
        <span className={"status-indicator status-"+status.isExternalNetworkAccessEnabled}>Access via Network</span>

            <br/>
            DCS-BIOS Version: {status.version} ({status.gitSHA1})
        </div>
    )
}
