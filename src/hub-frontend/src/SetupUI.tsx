import React, { useState, useEffect, ReactElement } from 'react';
import { apiPost } from './ApiConnection';

import './SetupUI.css'

type DcsInstallation = {
    installDir: string
    profileDir: string
    variant: string
    luaScriptsInstalled: boolean
    luaConsoleHookInstalled: boolean
    autostartHubHookInstalled: boolean
}

export default function SetupUI() {
    const [ exportLuaSetupLine, setExportLuaSetupLine ] = useState<string>("")
    const [ installs, setInstalls ] = useState<DcsInstallation[]>([])
    const [ lastSetupLog, setLastSetupLog ] = useState<string>("")

    const updateSetupInfoTable = () => {
        apiPost({
            datatype: "get_setup_info",
            data: {}
        }).then(response => {
            setInstalls(response.data.installs)
            setExportLuaSetupLine(response.data.exportLuaSetupLine)
        })
    }

    useEffect(() => {
        updateSetupInfoTable()
    }, [])


    const modifyExportLua = (install: DcsInstallation, shouldBeInstalled: boolean) => {
        apiPost({
            datatype: "modify_export_lua",
            data: {
                profileDir: install.profileDir,
                shouldBeInstalled: shouldBeInstalled
            }
        }).then(response => {
            setLastSetupLog(response.data.message)
            updateSetupInfoTable()
        })
    }

    const modifyHook = (hookType: string, install: DcsInstallation, shouldBeInstalled: boolean) => {
        apiPost({
            datatype: "modify_hook",
            data: {
                profileDir: install.profileDir,
                shouldBeInstalled: shouldBeInstalled,
                hookType: hookType
            }
        }).then(response => {
            setLastSetupLog(response.data.message)
            updateSetupInfoTable()
        })
    }

    let setupLog: ReactElement | null = null;
    if (lastSetupLog.length > 0) {
        setupLog = (<div className="setupLog">
            Export.lua modifications:<br/>
            <pre>{lastSetupLog}</pre>
        </div>)
    }

    return (
        <React.Fragment>
            <h2>Setup Scripts</h2>
            <table>
                <tbody>
                <tr>
                    <th>Installation Path</th>
                    <th>User Profile Path</th>
                    <th>Virtual Cockpit Connection</th>
                    <th>Autostart DCS-BIOS</th>
                    <th>Lua Console</th>

                    </tr>
                {installs.map(i => (
                    <tr key={i.installDir} className="dcs-installation">
                        <td><DirPath path={i.installDir}/></td>
                        <td><DirPath path={i.profileDir}/></td>
                        <td align="center" onClick={(e) => modifyExportLua(i, !i.luaScriptsInstalled)} className={"setup-td-"+i.luaScriptsInstalled.toString()}><input type="checkbox" checked={i.luaScriptsInstalled} readOnly/></td>
                        <td align="center" onClick={(e) => modifyHook("autostart", i, !i.autostartHubHookInstalled)} className={"setup-td-"+i.autostartHubHookInstalled.toString()}><input type="checkbox" checked={i.autostartHubHookInstalled} readOnly/></td>
                        <td align="center" onClick={(e) => modifyHook("luaconsole", i, !i.luaConsoleHookInstalled)} className={"setup-td-"+i.luaConsoleHookInstalled.toString()}><input type="checkbox" checked={i.luaConsoleHookInstalled} readOnly/></td>
                    </tr>
                ))}
                </tbody>
            </table>
            {setupLog}
                    <br/><br/>
                Use the table above to enable or disable DCS-BIOS features for each DCS: World installation.
                <ul>
                    <li>Check <b>Virtual Cockpit Connection</b> to hook into Export.lua so DCS-BIOS can receive cockpit data and send commands to DCS.</li>
                    <li>Check <b>Autostart DCS-BIOS</b> if you want to start the DCS-BIOS Hub automatically whenever you start DCS: World.</li>
                    <li>If you are a developer, you may want to check <b>Lua Console</b>. This is a prerequisite to use the Lua Console feature, which is useful when developing Lua scripts for DCS: World.</li>
                </ul>
            <br/>
            
            If your DCS installation is not shown above, you can manually add the following line to your <span style={{font: "monospace"}}>Export.lua</span>:<br/>
                <br/>
                <code>{exportLuaSetupLine}</code>

        </React.Fragment>
    );
}

// display the last element of a directory path,
// but provide the full path as a tooltip
function DirPath(props: { path: string }) {
    const parts = props.path.split("\\")
    const shortPath = parts[parts.length-1]
    return (
        <span title={props.path}>{shortPath}</span>
    )
}