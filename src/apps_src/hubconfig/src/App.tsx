import React from 'react';
import './App.css';

import SerialPortList from './SerialPortList'
import { LuaSnippet } from './LuaConsole';

function App() {
  return (
    <div className="App">
      <h1>DCS-BIOS Hub</h1>
      <h3>Lua Console</h3>
      <LuaSnippet />
      <h3>Serial Port Connections</h3>
      <SerialPortList />

    </div>
  );
}

export default App;
