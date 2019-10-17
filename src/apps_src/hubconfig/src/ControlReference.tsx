import React, { useState, useEffect, ReactElement } from 'react';

import {
  Route,
  Link,
  useParams,
  useRouteMatch
} from "react-router-dom";

import './ControlReference.css';

import { apiPost } from './ApiConnection';
import { stripTrailingSlash } from 'history/PathUtils';

type TIOElement = {
  name: string
  module: string
  category: string
  type: string
  description: string
  inputs: TInputElement[]
  outputs: TOutputElement[]
}
type TOutputElement = {
  address: number
  mask: number
  length: number
  description: string
  max_value: number
  shift_by: number
  suffix: string
  type: string
}
type TInputElement = {
  description: string
  interface: string
  max_value: number
  argument: string
}

class ControlReference extends React.Component<{ match: any }, { moduleNames: string[], moduleToCategory: any }> {
  constructor(props: any) {
    super(props)
    this.state = {
      moduleNames: [],
      moduleToCategory: {}
    }

  }

  componentDidMount() {
    apiPost({
      datatype: "control_reference_get_modules",
      data: {}
    }).then((msg: any) => {
      let names = Object.keys(msg.data)
      names.sort()
      this.setState({
        moduleNames: names,
        moduleToCategory: msg.data
      })
    })
  }

  componentWillUnmount() {
  }

  getCategoriesFromModuleName = (moduleName: string) => this.state.moduleToCategory[moduleName];

  render() {
    return (
      <div>
        <Route exact path={`${this.props.match.path}`} component={ControlReferenceIndex} />
        <Route exact path={`${this.props.match.path}/:moduleName`} render={(props) => <ControlReferenceForModule parentUrl={this.props.match.url} moduleNameToCategoryList={this.state.moduleToCategory} />} />
        <Route exact path={`${this.props.match.path}/:moduleName/:categoryName`} render={(props) => <ControlReferenceCategory controlReferenceUrl={this.props.match.url} />} />
      </div>
    )
  }
}

class ControlReferenceIndex extends React.Component<{}, { moduleNames: string[], modules: any }> {
  constructor(props: any) {
    super(props)
    this.state = {
      moduleNames: [],
      modules: {}
    }
  }
  componentDidMount() {
    apiPost({
      datatype: "control_reference_get_modules",
      data: {}
    }).then((msg: any) => {
      let names = Object.keys(msg.data)
      names.sort()
      this.setState({
        moduleNames: names,
        modules: msg.data
      })
    })
  }
  render() {
    return (
      <div>
        {
          this.state.moduleNames.map(name => <IndexCard key={name} moduleName={name} categories={this.state.modules[name]} />)
        }
      </div>
    )
  }
}

class IndexCard extends React.Component<{ moduleName: string, categories: string[] }, {}> {
  render() {
    return (

      <Route render={({ match }) =>

        <div className="" style={{ display: "block", float: "left", padding: "1em" }}>
          <Link to={match.path + '/' + encodeURIComponent(this.props.moduleName)}><h4>{this.props.moduleName}</h4></Link>
        </div>

      } />

    )
  }
}

function ControlReferenceForModule(props: { moduleNameToCategoryList: any, parentUrl: string }) {

  let params = useParams<{ moduleName: string }>();
  let match: any = useRouteMatch() || {}
  let categoryNames: string[] = props.moduleNameToCategoryList[params.moduleName] || []

  return (
    <div><h3><Link to={`${props.parentUrl}`}>Control Reference:</Link> {params.moduleName}</h3>
      <ul>
        {categoryNames.map(catName =>
          <li key={catName}><Link to={match.url + '/' + encodeURIComponent(catName)}>{catName}</Link></li>
        )}
      </ul>

    </div>
  )
}

function ControlReferenceCategory(props: { controlReferenceUrl: string }) {
  let params = useParams<{ moduleName: string, categoryName: string }>()
  let [ioElements, setIOElements] = useState<any>([]);

  // load list of IOElements when the component is loaded
  useEffect(() => {
    apiPost({
      datatype: "control_reference_query_ioelements",
      data: {
        module: params.moduleName,
        category: params.categoryName
      }
    }).then((msg: any) => {
      setIOElements(msg.data);
    })
  }, [params.moduleName, params.categoryName])


  return (
    <div>
      <h3><Link to='/controlreference'>Control Reference:</Link> <Link to={'/controlreference/' + encodeURIComponent(params.moduleName)}>{params.moduleName}</Link>: {params.categoryName}</h3>

      {ioElements.map((elem: any) => <IOElement key={elem.name} item={elem} />)}

    </div>
  )
}

function IOElement(props: { item: TIOElement }) {
  let inputs = props.item.inputs.map((input, idx) => <InputCommand control={props.item} input={input} key={idx} />);
  let outputs = props.item.outputs.map((output, idx) => <OutputElement control={props.item} output={output} key={idx} />);
  return (
    <div className="control">
      <div className="controlheader">
        <b>{props.item.description}</b>
        <span className="controlidentifier">{props.item.module}/{props.item.name}</span>
      </div>
      <div className="controlbody">
        <div className="inputs">
          {inputs}
        </div>
        <div className="outputs">
          {outputs}
        </div>
      </div>
    </div>
  )
}

function InputCommand(props: { control: TIOElement, input: TInputElement }) {
  let { control, input } = props;

  let codeSnippets: ReactElement[] = [];


  if (input.interface === "set_state" && input.max_value == 1) {
    codeSnippets.push(<Switch2PosSnippet key="Switch2Pos" {...props} />)
  }
  if (input.interface === "set_state" && input.max_value == 2) {
    codeSnippets.push(<Switch3PosSnippet key="Switch3Pos" {...props} />)
  }
  if (input.interface === "set_state" && input.max_value < 33) {
    codeSnippets.push(<SwitchMultiPosSnippet key="SwitchMultiPos" {...props} />)
  }
  if (input.interface === "set_state" && input.max_value === 65535) {
    codeSnippets.push(<PotentiometerSnippet key="Potentiometer" {...props} />)
  }
  if (input.interface === "variable_step") {
    codeSnippets.push(<RotaryEncoderVariableStepSnippet key="RotaryEncoderVariableStep" {...props} />)
  }
  if (input.interface === "fixed_step") {
    codeSnippets.push(<RotaryEncoderFixedStepSnippet key="RotaryEncoderFixedStep" {...props} />)
  }



  return (
    <div><CodeSnippetSelector snippets={codeSnippets} /></div>
  )
}



function OutputElement(props: { control: TIOElement, output: TOutputElement }) {
  let { control, output } = props;

  let codeSnippets: ReactElement[] = [];

  let x = output.type;

  if (output.type === "integer" && output.max_value === 1) {
    codeSnippets.push(<LEDSnippet key="LED" {...props} />);
  }
  if (output.type === "integer" && output.max_value === 65535) {
    codeSnippets.push(<ServoOutputSnippet key="ServoOutput" {...props}/>);
  }
  if (output.type === "integer") {
    codeSnippets.push(<IntegerBufferSnippet key="IntegerBuffer" {...props}/>);
  }
  if (output.type == "string") {
    codeSnippets.push(<StringBufferSnippet key="StringBuffer" {...props}/>);
  }

  return (
    <div><CodeSnippetSelector snippets={codeSnippets} /></div>
  )
}




function CodeSnippetSelector(props: { snippets: ReactElement[] }) {
  let { snippets } = props;
  let initialSelectedTab = (snippets.length > 0) ? snippets[0].key : "";
  let [activeTabKey, setActiveTab] = useState(initialSelectedTab)
  if (snippets.length === 0) {
    return null;
  }

  let tabSelectors: ReactElement[] = [];
  for (let snippet of snippets) {
    let isSelected = snippet.key == activeTabKey;
    let style: React.CSSProperties = {
      cursor: "hand"
    }
    if (isSelected) {
      style.fontWeight = "bold";
    }
    tabSelectors.push(<a className="snippet-tab-handle" style={style} onClick={() => setActiveTab(snippet.key)}>{snippet.key}</a>)
  }

  if (tabSelectors.length === 1) tabSelectors = [];

  return (
    <React.Fragment>
      <div>{tabSelectors}</div>
      <div className="current-snippet">{snippets.find(x => x.key == activeTabKey)}</div>
    </React.Fragment>
  )
}

const idCamelCase = function (input: string) {
  var ret = "";
  var capitalize = false;
  for (var i = 0; i < input.length; i++) {
    if (input[i] == '_') {
      capitalize = true;
    } else {
      if (capitalize) {
        ret = ret + input[i].toUpperCase();
        capitalize = false;
      } else {
        ret = ret + input[i].toLowerCase();
      }
    }
  }
  return ret;
};

// input code snippets:

function Switch2PosSnippet(props: { control: TIOElement, input: TInputElement }) {
  let { control, input } = props;
  return <code>DcsBios::Switch2Pos {idCamelCase(control.name)}("{control.name}", <b className="pinNo">PIN</b>);</code>;
}

function Switch3PosSnippet(props: { control: TIOElement, input: TInputElement }) {
  let { control, input } = props;
  return <code>DcsBios::Switch3Pos {idCamelCase(control.name)}("{control.name}", <b className="pinNo">PIN_A</b>, <b className="pinNo">PIN_B</b>);</code>;
}

function SwitchMultiPosSnippet(props: { control: TIOElement, input: TInputElement }) {
  let { control, input } = props;
  let pins: ReactElement[] = [];
  for (let i = 0; i <= input.max_value; i++) {
    let pinText = "PIN_" + i.toString();
    pins.push(<b className="pinNo" key={pinText}>{pinText}</b>);
    console.log("pushed", pinText)
    if (i < input.max_value) {
      pins.push(<span key={"comma" + pinText}>, </span>)
    }
  }
  console.log("number of pins:", pins.length, pins, input)
  return <code>const byte {idCamelCase(control.name)}Pins[{pins.length.toString()}] = {'{'}{pins.map(x => x)}{'}'}<br />DcsBios::SwitchMultiPos {idCamelCase(control.name)}("{control.name}", {idCamelCase(control.name)}Pins, {pins.length.toString()});</code>;
}

function RotaryEncoderVariableStepSnippet(props: { control: TIOElement, input: TInputElement }) {
  let { control, input } = props;
  return <code>DcsBios::RotaryEncoder {idCamelCase(control.name)}("{control.name}", "-3200", "+3200", <b className="pinNo">PIN_A</b>, <b className="pinNo">PIN_B</b>);</code>;
}

function RotaryEncoderFixedStepSnippet(props: { control: TIOElement, input: TInputElement }) {
  let { control, input } = props;
  return <code>DcsBios::RotaryEncoder {idCamelCase(control.name)}("{control.name}", "DEC", "INC", <b className="pinNo">PIN_A</b>, <b className="pinNo">PIN_B</b>);</code>;
}

function PotentiometerSnippet(props: { control: TIOElement, input: TInputElement }) {
  let { control, input } = props;
  return <code>DcsBios::Potentiometer {idCamelCase(control.name)}("{control.name}", <b className="pinNo">PIN</b>);</code>;
}

function ActionButtonSnippet(props: { control: TIOElement, input: TInputElement }) {
  let { control, input } = props;
  return <code>DcsBios::ActionButton {idCamelCase(control.name)}("{control.name}", "{input.argument}", <b className="pinNo">PIN</b>);</code>;
}

// output code snippets:

const hex = function (input: number) {
  if (input === 0)
    return "0x0000";
  var padTo = 4;
  if (!input)
    return "";
  if (!padTo)
    padTo = 4;
  var hex = input.toString(16)
  while (hex.length < padTo)
    hex = "0" + hex;
  return "0x" + hex;
};


function LEDSnippet(props: { control: TIOElement, output: TOutputElement }) {
  let { control, output } = props;
  return <code>DcsBios::LED {idCamelCase(control.name)}({hex(output.address)}, {hex(output.mask)}, <b className="pinNo">PIN</b>);</code>
}

function ServoOutputSnippet(props: { control: TIOElement, output: TOutputElement }) {
  let { control, output } = props;
  return <code>DcsBios::ServoOutput {idCamelCase(control.name)}({hex(output.address)},<b className="pinNo">PIN</b>, <b className="pinNo">544</b>, <b className="pinNo">2400</b>);</code>
}

function StringBufferSnippet(props: { control: TIOElement, output: TOutputElement }) {
  let { control, output } = props;
  return <code>void on{idCamelCase(control.name)}Change(char* newValue) {'{'}<br/>
&nbsp;&nbsp;&nbsp;&nbsp;/* your code here */<br/>
  {'}'}<br/>
    DcsBios::StringBuffer&lt;{output.length}&gt; {idCamelCase(control.name)}Buffer({hex(output.address)}, on{idCamelCase(control.name)}Change);</code>  
}

function IntegerBufferSnippet(props: { control: TIOElement, output: TOutputElement }) {
  let { control, output } = props;
  return <code>void on{idCamelCase(control.name)}Change(unsigned int newValue) {'{'}<br/>
&nbsp;&nbsp;&nbsp;&nbsp;/* your code here */<br/>
  {'}'}<br/>
    DcsBios::IntegerBuffer {idCamelCase(control.name)}Buffer({hex(output.address)}, on{idCamelCase(control.name)}Change);</code>  
}


export default ControlReference
