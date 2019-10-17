import React from 'react';
import './App.css';

import {
  BrowserRouter as Router,
  Route,
  NavLink
} from "react-router-dom";


import SerialPortList from './SerialPortList'
import ControlReference from './ControlReference'
import { LuaSnippet } from './LuaConsole';

function App() {
  return (
    <Router basename="/app/hubconfig">
      <div className="app">
        <div className="nav">
         <img alt="" src="/dcs-bios-logo-4.png" style={{marginLeft: "auto", marginRight: "auto", display: "block"}}/>
          <ul>
            <li><NavLink to='/controlreference' activeStyle={{ fontWeight: "bold" }}>Control Reference</NavLink></li>
            <li><NavLink to='/comports' activeStyle={{ fontWeight: "bold" }}>Configure serial ports</NavLink></li>
            <li><NavLink to='/luaconsole' activeStyle={{ fontWeight: "bold" }}>Lua Console</NavLink></li>
          </ul>
        </div>
        <div className="content">
            <Route exact path='/'>
              Welcome Home
            </Route>
            <Route path='/controlreference' component={ControlReference}/>
            <Route path='/comports' component={SerialPortList} />
            <Route path='/luaconsole' component={LuaSnippet} />
        </div>
      </div>
    </Router>
  );
}

export default App;
