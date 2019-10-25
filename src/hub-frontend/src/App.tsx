import React from 'react';
import './App.css';

import {
  BrowserRouter as Router,
  Route,
  NavLink,
} from "react-router-dom";


import { ControlReference } from './ControlReference'
import Dashboard from './Dashboard'
import { LuaSnippet } from './LuaConsole';
import SetupUI from './SetupUI';

const RootUrlContext = React.createContext("")

function App() {
  let rootUrl = "/app/hubconfig"
  if (window.location.port === "3000") {
    // development mode
    rootUrl = ""
  }

  return (
    <Router basename="/app/hubconfig">
      <RootUrlContext.Provider value={rootUrl}>
      <div className="app">
        <div className="nav">
         <img alt="" src={rootUrl+"/dcs-bios-logo-4.png"} style={{marginLeft: "auto", marginRight: "auto", display: "block"}}/>
          <ul>
          <li><NavLink exact to='/' activeStyle={{ fontWeight: "bold" }}>Dashboard</NavLink></li>
            <li><NavLink to='/controlreference' activeStyle={{ fontWeight: "bold" }}>Control Reference</NavLink></li>
            <li><NavLink to='/luaconsole' activeStyle={{ fontWeight: "bold" }}>Lua Console</NavLink></li>
            <li><NavLink to='/setup' activeStyle={{ fontWeight: "bold" }}>DCS Connection</NavLink></li>
            <li><a target="_blank" href="/app/doc/">Documentation</a></li>
          </ul>
        </div>
        <div className="content">
            <Route exact path='/' component={Dashboard}/>
            <Route path='/setup' component={SetupUI}/>
            <Route path='/controlreference' component={ControlReference}/>
            <Route path='/luaconsole' component={LuaSnippet} />
        </div>
      </div>
      </RootUrlContext.Provider>
    </Router>
  );
}

export default App;
