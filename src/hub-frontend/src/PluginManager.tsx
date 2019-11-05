import React, { useState, useEffect } from 'react';

import { getApiConnection, apiPost } from './ApiConnection'
import { useHistory, Link } from 'react-router-dom'
import './PluginManager.css'

type PluginState = {
    localName: string
    remoteURL: string
    checkedOutBranch: string
    canApplyUpdates: boolean
    checkedOutCommitHash: string
    checkedOutTags: string[]
    isManagedManually: boolean
    tags: string[]
    branches: string[]
    progressMessage: string
    progressState: string
}

function usePluginSelection() {
    let [selectedPlugins, setSelectedPlugins] = useState<string[]>([])

    const setPluginSelected = (cloneURL: string, status: boolean) => {
        let newSelectedPlugins = selectedPlugins.filter(x => true)
        if (status) {
            if (selectedPlugins.indexOf(cloneURL) === -1) {
                newSelectedPlugins.push(cloneURL)
            }
        } else {
            newSelectedPlugins = selectedPlugins.filter(x => x !== cloneURL)
        }
        setSelectedPlugins(newSelectedPlugins)
    }
    const isPluginSelected = (cloneURL: string) => {
        return selectedPlugins.indexOf(cloneURL) >= 0
    }

    return { selectedPlugins, setSelectedPlugins, setPluginSelected, isPluginSelected }
}

export function PluginManager() {
    const { selectedPlugins, setSelectedPlugins, setPluginSelected, isPluginSelected } = usePluginSelection()
    const [pluginList, setPluginList] = useState<PluginState[]>(() => [])
    // remove deleted plugins from selection
    let filteredSelection = selectedPlugins.filter(remoteURL => {
        for (let p of pluginList) {
            if (p.remoteURL === remoteURL) return true;
        }
        return false;
    })
    if (filteredSelection.length !== selectedPlugins.length) {
        setSelectedPlugins(filteredSelection)
    }

    useEffect(() => {
        const monitorModuleListWebsocket = getApiConnection()

        monitorModuleListWebsocket.onopen = () => {
            monitorModuleListWebsocket.send(JSON.stringify({
                datatype: "monitor_plugin_list",
                data: {}
            }))
        }
        monitorModuleListWebsocket.onmessage = msg => {
            msg = JSON.parse(msg.data)
            let newModules = msg.data as PluginState[]
            setPluginList(newModules)
        }
        return () => monitorModuleListWebsocket.close()
    }, [])

    const checkForUpdates = () => {
        apiPost({
            datatype: "check_for_plugin_updates",
            data: {}
        })
    }

    const countPlugins = (pluginList: PluginState[], predicate: (p: PluginState) => boolean) => {
        let c = 0
        for (let p of pluginList) {
            if (predicate(p)) c++;
        }
        return c
    }
    const workingCount = countPlugins(pluginList, p => p.progressState === "working")
    const selectedAvailableUpdateCount = countPlugins(pluginList, p => p.canApplyUpdates && isPluginSelected(p.remoteURL))

    const selectUnselectAll = () => {
        if (selectedPlugins.length === pluginList.length) {
            setSelectedPlugins([])
        } else {
            let newSelectedPlugins = []
            for (let p of pluginList) {
                newSelectedPlugins.push(p.remoteURL)
            }
            setSelectedPlugins(newSelectedPlugins)
        }
    }

    const deleteSelectedPlugins = () => {
        if (!window.confirm("Delete selected plugins?")) return;

        for (let p of pluginList) {
            if (!isPluginSelected(p.remoteURL)) continue;
            if (p.progressState === "working") continue;
            if (p.isManagedManually) continue;

            apiPost({
                datatype: "remove_plugin",
                data: {
                    localName: p.localName
                }
            })
        }
    }

    const applyAvailableUpdates = () => {
        for (let p of pluginList) {
            if (!isPluginSelected(p.remoteURL)) continue;
            if (!p.canApplyUpdates) continue;
            if (p.progressState === "working") continue;
            apiPost({
                datatype: "apply_plugin_updates",
                data: {
                    localName: p.localName
                }
            })
        }
    }

    return (
        <div>
            <h2>Plugins</h2>

            <ul>
                <li>
                    <Link to="/plugincatalog">Open the plugin catalog</Link> to browse available plugins and see recommended plugins for your installed DCS: World modules<br />
                </li>
                <li>
                    or install a specific plugin if you know its installation URL: <InstallModuleForm />
                </li>
                <li>
                    <button onClick={checkForUpdates}>Check for updates</button>
                </li>
            </ul>

            {workingCount > 0 ? <b>working on {workingCount.toString()} plugins...</b> : null}
            
            <br /><br />
            <button onClick={selectUnselectAll}>{selectedPlugins.length > 0 && selectedPlugins.length === pluginList.length ? "Unselect all" : "Select all"}</button>
            &nbsp;({selectedPlugins.length.toString()} plugins selected.)
            <br />
            <button onClick={applyAvailableUpdates} disabled={selectedAvailableUpdateCount === 0}>Apply {selectedAvailableUpdateCount.toString()} available updates</button> <button onClick={deleteSelectedPlugins} disabled={selectedPlugins.length === 0}>Delete selected plugins</button>
            <table>
                <thead>
                    <tr>
                        <td></td>
                        <th>Name</th>
                        <th>Version</th>
                    </tr>
                </thead>
                <tbody>
                    {pluginList.length === 0 ? <tr><td colSpan={3}>No plugins installed.<br/><br/><Link to="/plugincatalog">Open the plugin catalog</Link> to browse available plugins and see recommended plugins for your installed DCS: World modules.</td></tr> : null}
                    {pluginList.map(mod => <PluginState key={mod.localName} module={mod} selected={isPluginSelected(mod.remoteURL)} setPluginSelected={setPluginSelected} />)}
                </tbody>
            </table>

        </div>
    )
}

function InstallModuleForm() {
    const [remoteURL, setRemoteURL] = useState("")

    const install = () => {
        apiPost({
            datatype: "install_plugin",
            data: {
                remoteURL
            }
        }).then(msg => {
            console.log("install result", msg)
            if (msg.datatype === "error") {
                window.alert("Error: " + msg.data.message)
            } else if (msg.datatype === "success") {
                setRemoteURL("")
            }
        })
    }

    return (
        <React.Fragment>
            <input placeholder="Remote URL" type="text" onChange={e => setRemoteURL(e.target.value)} value={remoteURL} />
            <button onClick={install}>Install</button>
        </React.Fragment>
    )
}

function PluginState(props: { module: PluginState, selected: boolean, setPluginSelected: (remoteURL: string, state: boolean) => void }) {
    const mod = props.module

    const workInProgress = (mod.progressState === "working")

    const progressInfo = (
        <div>

            <span>{mod.progressState === "working" ? mod.progressMessage : null}</span>
            {mod.progressState === "error" ? <span>Error: {mod.progressMessage}</span> : null}


        </div>
    )
    const versionInfo = (
        <div>

            <span className="commit-hash">{mod.checkedOutCommitHash}</span><br />
            {mod.checkedOutBranch === "" ? null : <span className="git-branch">{mod.checkedOutBranch}</span>}
            {mod.checkedOutTags.map((tag) => { return <span className="git-tag" key={tag}>{tag}</span> })}
            {mod.canApplyUpdates ? <span className="update-available">update available</span> : null}
            {mod.isManagedManually ? <span>(installed manually, ignored by plugin manager)</span> : null}
        </div>

    )

    return (
        <tr className={"plugin-state" + (props.selected ? " selected" : "")} onClick={e => { props.setPluginSelected(mod.remoteURL, !props.selected) }}>
            <td>
                <input type="checkbox" checked={props.selected} readOnly={true}></input>
            </td>
            <td>
                <b>{mod.localName}</b>  <br />
                <span className="remote-url">{mod.remoteURL}</span><br />

            </td>
            <td>

                {workInProgress ? progressInfo : versionInfo}

            </td>

        </tr>
    )
}


type pluginInfo = {
    cloneURL: string
    description: string
    website: string
    recommendForModPath: string
    isRecommended: boolean
    isAlreadyInstalled: boolean
    localName: string
}

export function PluginCatalog() {
    const { selectedPlugins, setSelectedPlugins, setPluginSelected, isPluginSelected } = usePluginSelection()

    const [availablePlugins, setAvailablePlugins] = useState<pluginInfo[]>([])
    const [loading, setLoading] = useState(true)
    const [showOnlyRecommended, setShowOnlyRecommended] = useState(true)

    useEffect(() => {
        setLoading(true)
        apiPost({
            datatype: "get_plugin_catalog",
            data: {}
        }).then(response => {
            if (response.datatype === "plugin_catalog") {
                let newSelectedPlugins = []
                for (let plugin of response.data) {
                    if (plugin.isRecommended && !plugin.isAlreadyInstalled) {
                        newSelectedPlugins.push(plugin.cloneURL)
                    }
                }
                setAvailablePlugins(response.data)
                setSelectedPlugins(newSelectedPlugins)
                setLoading(false)
            } else if (response.datatype === "error") {
                window.alert(response.data.message)
            }
        })
    }, [setSelectedPlugins])

    const filteredPlugins = availablePlugins.filter(p => {
        if (p.isAlreadyInstalled) return false;
        if (p.isRecommended) return true;
        return !showOnlyRecommended;
    })

    const countPlugins = (pluginList: pluginInfo[], predicate: (p: pluginInfo) => boolean) => {
        let c = 0
        for (let p of pluginList) {
            if (predicate(p)) c++;
        }
        return c
    }

    const history = useHistory()
    const installSelectedPlugins = () => {
        for (let cloneURL of selectedPlugins) {
            apiPost({
                datatype: "install_plugin",
                data: {
                    remoteURL: cloneURL
                }
            })
            history.push("/pluginmanager")
        }

    }

    const recommendedCount = countPlugins(availablePlugins, p => p.isRecommended && !p.isAlreadyInstalled)
    const selectedCount = selectedPlugins.length

    return (
        <div>
            {loading ? "loading..." : null}
            {availablePlugins.length > 0 ? <span>
                Showing <b>{filteredPlugins.length.toString()}/{availablePlugins.length.toString()}</b> plugins from catalog. (Already installed plugins are not shown.)<br />

                <input type="checkbox" readOnly={true} defaultChecked={showOnlyRecommended} onClick={e => setShowOnlyRecommended(!showOnlyRecommended)} />only show recommended plugins ({recommendedCount.toString()})
            </span> : null}
            <br />
            <button onClick={installSelectedPlugins} disabled={selectedCount === 0}>Install {selectedCount} Plugins</button> or <Link to='/pluginmanager'>go back to plugin list</Link>

            <table>
                <tbody>
                {filteredPlugins.map(p => <PluginCatalogEntry key={p.cloneURL} plugin={p} selected={isPluginSelected(p.cloneURL)} setPluginSelected={setPluginSelected} />)}
                </tbody>
            </table>
            
        </div>
    )
}

function PluginCatalogEntry(props: { plugin: pluginInfo, selected: boolean, setPluginSelected: (cloneURL: string, status: boolean) => void }) {
    const p = props.plugin

    return (
        <tr className={"plugin-catalog-entry" + (props.selected ? " selected" : "")} onClick={e => props.setPluginSelected(props.plugin.cloneURL, !props.selected)}>
            <td className="plugin-catalog-entry-checkbox">
                <input type="checkbox" readOnly={true} checked={props.selected} />
            </td>

            <td className="plugin-catalog-entry-description">
                <b>{p.localName}</b> {p.isRecommended ? <span>(recommended{p.recommendForModPath !== "" ? <span> for {p.recommendForModPath}</span> : null})</span> : null}<br />
                <span className="remote-url">{p.cloneURL}</span><br />
                {p.description}
            </td>
        </tr>
    )
}
