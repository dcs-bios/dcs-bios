import React, { useState, useEffect, ReactElement } from 'react';
import { apiPost } from './ApiConnection';

type DcsInstallation = {
    installDir: string
    profileDir: string
    variant: string
    luaScriptsInstalled: boolean
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

                To be able to communicate with DCS: World, the DCS-BIOS Lua Scripts need to be installed.
            
                The table below shows the auto-detected DCS: World installations on your machine (Release and Open Beta). You can use the buttons on the right
                to enable or disable DCS-BIOS for that installation.

            <table>
                <tbody>
                <tr>
                    <th>Installation Path</th>
                    <th>User Profile Path</th>
                    <th>DCS-BIOS Scripts installed?</th>
                    <th></th>
                    <th></th>
                    </tr>
                {installs.map(i => (
                    <tr key={i.installDir}>
                        <td>{i.installDir}</td>
                        <td>{i.profileDir}</td>
                        <td style={{width: "13em", textAlign: "center", backgroundColor: i.luaScriptsInstalled ? "lightgreen" : "white"}}>{i.luaScriptsInstalled ? "OK" : "not installed"}</td>
                        <td><button onClick={() => modifyExportLua(i, true)}>Install Scripts</button></td>
                        <td><button onClick={() => modifyExportLua(i, false)}>Remove Scripts</button></td>
                    </tr>
                ))}
                </tbody>
            </table>
            <br/>
            {setupLog}
            <br/>
            If your DCS installation is not shown above, you can manually add the following line to your <span style={{font: "monospace"}}>Export.lua</span>:<br/>
                <br/>
                <code>{exportLuaSetupLine}</code>

        </React.Fragment>
    );
}