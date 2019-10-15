import 'codemirror/lib/codemirror.css'
import 'codemirror/theme/material.css'
import 'codemirror/mode/lua/lua'
import 'codemirror/mode/javascript/javascript'

import React from 'react';
import { Controlled as CodeMirror } from 'react-codemirror2';
import getApiConnection from './ApiConnection';

type LuaSnippetState = {
    luaEnvironment: string
    code: string
    responseStatus: string
    responseText: string
    readyToExecute: boolean
}

class LuaSnippet extends React.Component<{}, LuaSnippetState> {
    constructor(props: {}) {
        super(props)
        this.state = {
            luaEnvironment: "gui",
            code: "",
            responseStatus: "",
            responseText: "",
            readyToExecute: true,
        }
    }

    onKeyPress = (src: any, event:any) => {
        if (event.ctrlKey && event.keyCode === 13) {
            this.executeSnippet()
        }
    }

    executeSnippet = () => {
        if (!this.state.readyToExecute) return;
        this.setState({
            readyToExecute: false
        })
        let conn = getApiConnection()
        conn.onopen = () => {
            conn.send(JSON.stringify({
                datatype:"execute_lua_snippet",
                data: {
                    luaEnvironment: this.state.luaEnvironment,
                    luaCode: this.state.code
                }
            }))
        }
        conn.onmessage = (result) => {
            let msg = JSON.parse(result.data)
            conn.close()
            this.setState({
                responseStatus: msg.data.status,
                responseText: msg.data.result,
                readyToExecute: true
            })
        }
    }

    render() {
        return (
            <div>
                Lua Console:<br/>
                Env: <select value={this.state.luaEnvironment} onChange={(e) => {this.setState({ luaEnvironment: e.target.value });}}>
                    <option value="mission">mission</option>
                    <option value="export">export</option>
                    <option value="gui">gui</option>
                    </select> 
                Code: <CodeMirror
                    value={this.state.code}
                    options={{
                        mode: 'lua',
                        lineNumbers: true
                    }}
                    onBeforeChange={(editor, data, value) => {this.setState({code: value}); }}
                    onKeyPress={this.onKeyPress} /><br/>
                <button onClick={this.executeSnippet} disabled={!this.state.readyToExecute}>Execute</button>
                <hr/>
                Response: {this.state.responseStatus} 
                <CodeMirror
                    value={this.state.responseText}
                    options={{
                        lineNumbers: true
                    }}
                    onBeforeChange={(editor, data, value) => {}} />

            </div>
        )
    }
}

export { LuaSnippet }
