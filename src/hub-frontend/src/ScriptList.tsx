import React, { ReactElement, useState, useEffect } from 'react';
import { apiPost, getApiConnection } from './ApiConnection';
import './ScriptList.css'


type ScriptListEntry = {
    path: string,
    enabled: boolean
}
export function ScriptList() {
    const [scriptList, setScriptList] = useState<ScriptListEntry[]>([])
    useEffect(() => {
        const monitorScriptListWebsocket = getApiConnection()

        monitorScriptListWebsocket.onopen = () => {
            monitorScriptListWebsocket.send(JSON.stringify({
                datatype: "monitor_script_list",
                data: {}
            }))
        }
        monitorScriptListWebsocket.onmessage = msg => {
            let parsedMsg = JSON.parse(msg.data)
            let newScriptList = parsedMsg.data as ScriptListEntry[]
            setScriptList(newScriptList)
        }
        return () => monitorScriptListWebsocket.close()
    }, [])

    // transform '"foo bar"' to 'foo bar'
    const unquote = (s: string) => {
        const match = s.match(/^"(.*)"$/)
        if (match == null || match.length != 2) {
            return s
        }
        return match[1]
    }

    const addEntry = () => {
        let path = prompt("Path to script file:")
        if (path == null) return;
        path = unquote(path)

        let newScriptList = scriptList.slice()
        newScriptList.push({
            path: path,
            enabled: true
        })
        apiPost({
            datatype: "set_script_list",
            data: newScriptList
        })
    }

    const toggleEnabled = (i: number) => {
        let newScriptList = scriptList.slice()
        newScriptList[i].enabled = !newScriptList[i].enabled
        apiPost({
            datatype: "set_script_list",
            data: newScriptList
        })
    }

    const removeEntry = (i: number) => {
        let newScriptList = scriptList.slice()
        newScriptList.splice(i, 1)
        apiPost({
            datatype: "set_script_list",
            data: newScriptList
        })
    }

    return (
        <div>
            <table>
                <tbody>
                    {scriptList.map((entry, i) => <tr key={i.toString()} className={"script-list-entry"+(entry.enabled ? "" : " disabled")}>
                        <td><input onClick={e => toggleEnabled(i)} type="checkbox" checked={entry.enabled} readOnly={true} /></td>
                        <td>{entry.path}</td>
                        <td><button onClick={e => removeEntry(i)}>x</button></td>
                    </tr>)}
                </tbody>
            </table>
            <button onClick={addEntry}>Add...</button>
        </div>
    )
}

export function ReloadScripts() {
    const [resultMsg, setResultMsg] = useState<string>("")
    const [buttonDisabled, setButtonDisabled] = useState<boolean>(false)

    const doReload = () => {
        setButtonDisabled(true)
        setResultMsg("")
        apiPost({
            "datatype": "reload_scripts",
            "data": {}
        }).then(result => {
            setResultMsg(result.data.message)
            setButtonDisabled(false)
        })
    }

    let resultMsgDiv: ReactElement | null = null
    if (resultMsg !== "") {
        resultMsgDiv = <pre>{resultMsg}</pre>
    }


    return (
        <div>
            <button disabled={buttonDisabled} onClick={doReload}>Reload Scripts</button>
            <br />
            {resultMsgDiv}
        </div>
    )
}
