import React, { ReactElement, useState } from 'react';
import { apiPost } from './ApiConnection';
import './Status.css'

export function HooksLuaReload() {
    const [resultMsg, setResultMsg] = useState<string>("")
    const [buttonDisabled, setButtonDisabled] = useState<boolean>(false)

    const doReload = () => {
        setButtonDisabled(true)
        setResultMsg("")
        apiPost({
            "datatype": "reload_hooks_lua",
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
            <button disabled={buttonDisabled} onClick={doReload}>Reload hooks.lua</button>
            <br />
            {resultMsgDiv}
        </div>
    )
}
