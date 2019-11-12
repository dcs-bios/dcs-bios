import React, { useState, useEffect, ReactElement, useRef, useContext } from 'react';
import { HashLink } from 'react-router-hash-link';

import {
  Route,
  Link,
  useParams,
  useRouteMatch
} from "react-router-dom";

import './ControlReference.css';

import ExportDataParser from './ExportDataParser'

import { apiPost, getApiConnection } from './ApiConnection';

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
  max_length: number
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
  suggested_step: number
}

// A TLiveDataContext is provided by the top level ControlReference component and provides connectivity to the
// DCS-BIOS Hub to listen to export data and send commands to DCS. This enables the interactive features of the
// control reference documentation.
type TLiveDataContext = {
  subscribeExportCallback: (address: number, callback: (address: number, data: ArrayBuffer) => void) => void
  unsubscribeExportCallback: (callback: (address: number, data: ArrayBuffer) => void) => void
  subscribeEndOfUpdateCallback: (callback: () => void) => void
  unsubscribeEndOfUpdateCallback: (callback: () => void) => void
  sendInputData: (msg: string) => void
}
const LiveDataContext = React.createContext<TLiveDataContext>({
  subscribeExportCallback: console.log,
  unsubscribeExportCallback: console.log,
  subscribeEndOfUpdateCallback: console.log,
  unsubscribeEndOfUpdateCallback: console.log,
  sendInputData: console.log
});


function ControlReference() {
  let match = useRouteMatch() as any;


  let [moduleToCategory, setModuleToCategory] = useState<any>({});
  let exportDataParser = useState<ExportDataParser>(() => new ExportDataParser())[0]
  const dummySend = (msg:any ) => console.log
  let [ sendWebsocketMsg, setSendWebsocketMsg ] = useState<(msg: any)=>void>(dummySend)

  const liveDataCallbacks: TLiveDataContext = {
    subscribeExportCallback: exportDataParser.registerExportDataListener.bind(exportDataParser),
    unsubscribeExportCallback: exportDataParser.unregisterExportDataListener.bind(exportDataParser),
    subscribeEndOfUpdateCallback: exportDataParser.registerEndOfUpdateCallback.bind(exportDataParser),
    unsubscribeEndOfUpdateCallback: exportDataParser.unregisterEndOfUpdateListener.bind(exportDataParser),
    sendInputData: (msg) => {
      sendWebsocketMsg(JSON.stringify({
        "datatype": "input_command",
        "data": msg
      }))
    }
  }

  useEffect(() => {
    const liveDataWebsocket = getApiConnection()
    liveDataWebsocket.onopen = () => {
      const foo = (msg: any) => liveDataWebsocket.send(msg)
      setSendWebsocketMsg(() => foo)
      liveDataWebsocket.binaryType = "arraybuffer"
      liveDataWebsocket.send(JSON.stringify({
        datatype: "live_data",
        data: {}
      }))
    }
    liveDataWebsocket.onmessage = async (response) => {
      let data = response.data
      //console.log("wsresponse", data)
      let a = new Uint8Array(data)
      for (let i = 0; i < a.length; i++) {
        exportDataParser.processByte(a[i])
      } 
    }
    return () => {
      if (liveDataWebsocket) liveDataWebsocket.close();
    }
  }, [exportDataParser])


  useEffect(() => {
    apiPost({
      datatype: "control_reference_get_modules",
      data: {}
    }).then((msg: any) => {
      setModuleToCategory(msg.data)
    })
  }, [])

  return (
    <LiveDataContext.Provider value={liveDataCallbacks}>
      <div>
        <Route exact path={`${match.path}`} component={ControlReferenceIndex} />
        <Route exact path={`${match.path}/:moduleName`} render={() => <ControlReferenceForModule parentUrl={match.url} moduleNameToCategoryList={moduleToCategory} />} />
        <Route exact path={`${match.path}/:moduleName/:categoryName`} render={() => <ControlReferenceCategory />} />
      </div>
    </LiveDataContext.Provider>
  )
}

function ControlReferenceIndex() {
  const [moduleNames, setModuleNames] = React.useState<string[]>([])
  const [modules, setModules] = React.useState<any>({})

  useEffect(() => {
    apiPost({
      datatype: "control_reference_get_modules",
      data: {}
    }).then((msg: any) => {
      let names = Object.keys(msg.data)
      names.sort()
      setModuleNames(names)
      setModules(msg.data)
    })
  }, [])
  
  let allModulesElement = (
      <div>
        <h2>Control Reference</h2>
        {
          moduleNames.map(name => <IndexCard key={name} moduleName={name} categories={modules[name]} />)
        }
      </div>);
  
  return (
    <div>
      {allModulesElement}
      <div style={{ clear: "both" }}></div>
    </div>
  )
}


function IndexCard(props: { moduleName: string, categories: string[] }) {
  return (
    <div className="controlreference-index-module">
      <Link to={'/controlreference/' + encodeURIComponent(props.moduleName)}><b>{props.moduleName}</b></Link>
    </div>
  )
}

function ControlReferenceSearchResults(props: { searchTerm: string, moduleName: string }) {
  const [searchResults, setSearchResults] = useState<TIOElement[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAll, setShowAll] = useState(false);
  const match = useRouteMatch()

  useEffect(() => {
    if (props.searchTerm === "") return;
    let ignore = false;
    apiPost({
      datatype: "control_reference_query_ioelements",
      data: {
        module: props.moduleName,
        category: "",
        searchTerm: props.searchTerm
      }
    }).then(data => {
      if (!ignore) {
        setShowAll(false)
        setSearchResults((data as any).data)
        setLoading(false)
      }
    })
    return () => { ignore = true; }
  }, [props.searchTerm, props.moduleName])


  if (props.searchTerm === "") return null;
  if (loading) return <div>searching {props.moduleName} for "{props.searchTerm}"...</div>;

  let searchResultsByCategory: Map<string, Array<TIOElement>> = new Map()
  for (let elem of searchResults) {
    if (!searchResultsByCategory.has(elem.category)) {
      searchResultsByCategory.set(elem.category, [])
    }
    (searchResultsByCategory.get(elem.category) as TIOElement[]).push(elem)
  }

  let sortedSearchResultCategories = new Array<string>()
  searchResultsByCategory.forEach((_, k) => sortedSearchResultCategories.push(k))
  sortedSearchResultCategories.sort()

  searchResultsByCategory.forEach(resultList => {
    resultList.sort((a, b) => {
      if (a.description < b.description) return 1;
      if (a.description > b.description) return -1;
      return 0;
    })
  });

  const makeResultUl = () => {
    const defaultNumberOfResults = 10; // if more than 10 results found, show "show more" button
    let count = 0
    let showMoreButton: ReactElement | null = null
    let categoryListItems: ReactElement[] = []
    for (let categoryName of sortedSearchResultCategories) {
      if (count === -1) break;

      let listItemsInCategory: ReactElement[] = []
      let searchResultsInCategory = searchResultsByCategory.get(categoryName) as TIOElement[]

      // create a list of <li> elements in searchResultsInCategory
      for (let elem of searchResultsInCategory) {
        if (count === -1) break;
        listItemsInCategory.push(<li key={elem.name}>
          <HashLink key={elem.name} to={(match ? match.url : "") + '/' + encodeURIComponent(categoryName) + '#' + elem.name}>
            <b>{props.moduleName}/{elem.name}:</b> {elem.description}
          </HashLink>
        </li>)

        count++
        if (count === defaultNumberOfResults && searchResults.length > defaultNumberOfResults && !showAll) {
          showMoreButton = <button onClick={() => setShowAll(true)}>Show {searchResults.length - defaultNumberOfResults} more results</button>
          count = -1
          break
        }
      }
      // make a <ul> for the category
      categoryListItems.push(<li key={categoryName}>
        <b>{categoryName}</b>
        <ul>{listItemsInCategory}</ul>
      </li>)
    }

    return <ul>{categoryListItems}<br />{showMoreButton}</ul>
  }

  return (
    <div>{searchResults.length.toString()} results for {props.searchTerm}:
    {makeResultUl()}
    </div>
  )
}

function ControlReferenceForModule(props: { moduleNameToCategoryList: any, parentUrl: string }) {

  let params = useParams<{ moduleName: string }>();
  let match: any = useRouteMatch() || {}
  let categoryNames: string[] = props.moduleNameToCategoryList[params.moduleName] || []
  const [categoryListFilter, setCategoryListFilter] = useState("");
  const [searchTerm, setSearchTerm] = useState("");

  let filteredCategories = categoryNames.filter((catName) => catName.toLowerCase().indexOf(categoryListFilter.toLowerCase()) >= 0)

  let moduleName = decodeURIComponent(params.moduleName)

  return (
    <div><h3><Link to={`${props.parentUrl}`}>Control Reference:</Link> {moduleName}</h3>
      Search for a specific control inside the {moduleName} module:<br />
      <input type="text" value={searchTerm} onChange={(e) => setSearchTerm(e.target.value)} /><br />
      <ControlReferenceSearchResults searchTerm={searchTerm} moduleName={moduleName} />
      <hr />

      Browse a category:<br /><input type="text" value={categoryListFilter} placeholder="Filter categories" onChange={(e) => setCategoryListFilter(e.target.value)} />
      <ul>
        {filteredCategories.map(catName =>
          <li key={catName}><Link to={match.url + '/' + encodeURIComponent(catName)}>{catName}</Link></li>
        )}
      </ul>

    </div>
  )
}

function ControlReferenceCategory() {
  let params = useParams<{ moduleName: string, categoryName: string }>()
  let [ioElements, setIOElements] = useState<TIOElement[]>([]);
  let [controlFilterText, setControlFilterText] = useState("");

  let [moduleName, categoryName] = [params.moduleName, params.categoryName].map(decodeURIComponent)

  // load list of IOElements when the component is loaded
  useEffect(() => {
    apiPost({
      datatype: "control_reference_query_ioelements",
      data: {
        module: moduleName,
        category: decodeURIComponent(categoryName)
      }
    }).then((msg: any) => {
      const compareByKey = (a: TIOElement, b: TIOElement) => {
        if (a.description < b.description)
          return -1;
        else if (a.description > b.description)
          return 1;
        else
          return 0;
      }
      msg.data.sort(compareByKey);
      setIOElements(msg.data as TIOElement[]);
    })
  }, [moduleName, categoryName])

  let filteredIOElements = ioElements.filter((element) => {
    let fstr = controlFilterText.toLowerCase()
    return (element.description.toLowerCase().indexOf(fstr) >= 0) || (element.name.toLowerCase().indexOf(fstr) >= 0)
  })

  return (
    <div>
      <h3><Link to='/controlreference'>Control Reference:</Link> <Link to={'/controlreference/' + encodeURIComponent(params.moduleName)}>{moduleName}</Link>: {categoryName}</h3>
      <input type="text" placeholder="Filter" value={controlFilterText} onChange={(e) => setControlFilterText(e.target.value)} />
      <span> </span>({filteredIOElements.length.toString()}/{ioElements.length.toString()} displayed)
      {filteredIOElements.map((elem: any) => <IOElementDocumentation key={elem.name} item={elem} />)}

    </div>
  )
}


type SnippetDescriptionPair = { snippet: ReactElement, description: string }


function IOElementDocumentation(props: { item: TIOElement }) {
  const inputSnippetPrecedence = [
    "Potentiometer",
    "Switch2Pos",
    "Switch3Pos",
    "SwitchMultiPos",
    "RotaryEncoderVariableStep",
    "RotaryEncoderFixedStep",
    "ActionButton",
    "LED",
    "StringBuffer",
    "IntegerBuffer",
    "ServoOutput",
  ];
  // take a list of inputs and transform it into a list of { CodeSnippet, Description } pairs
  let inputSnippets: Array<SnippetDescriptionPair> = props.item.inputs.flatMap(input => getInputCodeSnippets(props.item, input).map(snippet => ({ snippet, description: input.description })));
  const compareByCodeSnippetPrecedence = (a: SnippetDescriptionPair, b: SnippetDescriptionPair) => {
    let aIdx = inputSnippetPrecedence.indexOf(a.snippet.key as string);
    let bIdx = inputSnippetPrecedence.indexOf(b.snippet.key as string);
    if (aIdx === -1) console.log("missing code snippet sort key:", a.snippet.key);
    if (bIdx === -1) console.log("missing code snippet sort key:", b.snippet.key)
    return aIdx - bIdx;
  }
  inputSnippets.sort(compareByCodeSnippetPrecedence);

  let integerOutputs = props.item.outputs.filter(o => o.type === "integer");
  let stringOutputs = props.item.outputs.filter(o => o.type === "string");

  let integerOutputSnippets: Array<SnippetDescriptionPair> = integerOutputs.flatMap(output => getOutputCodeSnippets(props.item, output).map(snippet => ({ snippet, description: output.description })));
  let stringOutputSnippets: Array<SnippetDescriptionPair> = stringOutputs.flatMap(output => getOutputCodeSnippets(props.item, output).map(snippet => ({ snippet, description: output.description })));

  let outputElements: ReactElement[] = []
  if (integerOutputSnippets.length > 0) {
    outputElements.push(<div key="intOutput" className="outputWrapper">
      <CodeSnippetSelector descriptionPrefix={<b>Integer Output: </b>} snippetDescriptionPairs={integerOutputSnippets} />
      <LiveOutputData output={props.item.outputs.find(out => out.type === "integer") as TOutputElement} />
    </div>);
  }
  if (stringOutputSnippets.length > 0) {
    outputElements.push(<div key="strOutput" className="outputWrapper">
      <CodeSnippetSelector descriptionPrefix={<b>String Output: </b>} snippetDescriptionPairs={stringOutputSnippets} />
      <LiveOutputData output={props.item.outputs.find(out => out.type === "string") as TOutputElement} />
    </div>);
  }

  return (
    <div className="control">
      <div className="controlheader">
        <span id={props.item.name} />
        <b>{props.item.description}</b>
        <span className="controlidentifier">{props.item.module}/{props.item.name}</span>
      </div>
      <div className="controlbody">
        <div className="inputWrapper">
          <CodeSnippetSelector descriptionPrefix={<b>Input: </b>} snippetDescriptionPairs={inputSnippets} />
          <LiveInputControls control={props.item} />
        </div>
        {outputElements}
      </div>
    </div>
  )
}

function getInputCodeSnippets(control: TIOElement, input: TInputElement) {
  let props = { control, input };
  let codeSnippets: ReactElement[] = [];

  if (input.interface === "set_state" && input.max_value === 1) {
    codeSnippets.push(<Switch2PosSnippet key="Switch2Pos" {...props} />)
  }
  if (input.interface === "set_state" && input.max_value === 2) {
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

  return codeSnippets;
}



function getOutputCodeSnippets(control: TIOElement, output: TOutputElement) {
  let props = { control, output };
  let codeSnippets: ReactElement[] = [];

  if (output.type === "integer" && output.max_value === 1) {
    codeSnippets.push(<LEDSnippet key="LED" {...props} />);
  }
  if (output.type === "integer") {
    codeSnippets.push(<IntegerBufferSnippet key="IntegerBuffer" {...props} />);
  }
  if (output.type === "integer" && output.max_value === 65535) {
    codeSnippets.push(<ServoOutputSnippet key="ServoOutput" {...props} />);
  }
  if (output.type === "string") {
    codeSnippets.push(<StringBufferSnippet key="StringBuffer" {...props} />);
  }

  return codeSnippets;
}



// Displays one or more code snippets in tabs.
// Tabs are hidden if only one code snippet is available.
// Includes a "copy to clipboard" button for the currently visible snippet.
function CodeSnippetSelector(props: { snippetDescriptionPairs: Array<SnippetDescriptionPair>, descriptionPrefix: any }) {
  let snippetDescriptionPairs = props.snippetDescriptionPairs;
  let snippetRef = useRef<HTMLDivElement>(null);
  let initialSelectedTab = (snippetDescriptionPairs.length > 0) ? snippetDescriptionPairs[0].snippet.key : "";
  let [activeTabKey, setActiveTab] = useState(initialSelectedTab)
  if (snippetDescriptionPairs.length === 0) {
    return null;
  }

  const copyToClipboard = () => {
    let element = snippetRef.current;

    let currentSelection = document.getSelection();
    let selectionType = currentSelection && currentSelection.type;
    if (selectionType === "Range") return; // do not mess with the user's own selection

    // http://stackoverflow.com/questions/11128130/select-text-in-javascript
    var doc = document;
    if ((doc.body as any).createTextRange) { // ms
      var range = (doc.body as any).createTextRange();
      range.moveToElementText(element);
      if (!range.execCommand('copy')) {
        range.select();
      }
    } else if (window.getSelection) { // moz, opera, webkit
      let selection = window.getSelection();
      let range = doc.createRange() as any;
      range.selectNodeContents(element);
      if (selection) {
        selection.removeAllRanges();
        selection.addRange(range);
        if (document.execCommand('copy')) {
          selection.removeAllRanges();
        }
      }
    }
    (snippetRef.current as HTMLDivElement).classList.add("copied");
    setTimeout(() => { (snippetRef.current as HTMLDivElement).classList.remove("copied"); }, 150)
  }

  let tabSelectors: ReactElement[] = [];
  for (let st of snippetDescriptionPairs) {
    let snippet = st.snippet
    let isSelected = snippet.key === activeTabKey;
    let style: React.CSSProperties = {
      cursor: "hand"
    }
    let activeClass = isSelected ? " active-tab-handle" : "";
    tabSelectors.push(<button className={"snippet-tab-handle" + activeClass} key={snippet.key as string} style={style} onClick={() => setActiveTab(snippet.key)}>{snippet.key}</button>)
  }

  if (tabSelectors.length === 1) tabSelectors = [];

  let activeSnippetTuple = snippetDescriptionPairs.find(x => x.snippet.key === activeTabKey) as SnippetDescriptionPair; // type assertion to guarantee that this will not be null

  return (
    <div className="snippetSelector">
      <div>{tabSelectors} <span className="io-description">{props.descriptionPrefix}{activeSnippetTuple.description}</span></div>
      <div className="current-snippet" ref={snippetRef} onClick={copyToClipboard}>{activeSnippetTuple.snippet}</div>
    </div>
  )
}

// idCamelCase converts a control identifier of the form "UFC_BTN_CLEAR"
// to camel case (ufcBtnClear) for use as an identifier in C++ code.
const idCamelCase = function (input: string) {
  var ret = "";
  var capitalize = false;
  for (var i = 0; i < input.length; i++) {
    if (input[i] === '_') {
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
  let { control } = props;
  return <code>DcsBios::Switch2Pos {idCamelCase(control.name)}("{control.name}", <b className="pinNo">PIN</b>);</code>;
}

function Switch3PosSnippet(props: { control: TIOElement, input: TInputElement }) {
  let { control } = props;
  return <code>DcsBios::Switch3Pos {idCamelCase(control.name)}("{control.name}", <b className="pinNo">PIN_A</b>, <b className="pinNo">PIN_B</b>);</code>;
}

function SwitchMultiPosSnippet(props: { control: TIOElement, input: TInputElement }) {
  let { control, input } = props;
  let pins: ReactElement[] = [];
  for (let i = 0; i <= input.max_value; i++) {
    let pinText = "PIN_" + i.toString();
    pins.push(<b className="pinNo" key={pinText}>{pinText}</b>);
    if (i < input.max_value) {
      pins.push(<span key={"comma" + pinText}>, </span>)
    }
  }
  return <code>const byte {idCamelCase(control.name)}Pins[{pins.length.toString()}] = {'{'}{pins.map(x => x)}{'}'}<br />DcsBios::SwitchMultiPos {idCamelCase(control.name)}("{control.name}", {idCamelCase(control.name)}Pins, {pins.length.toString()});</code>;
}

function RotaryEncoderVariableStepSnippet(props: { control: TIOElement, input: TInputElement }) {
  let { control } = props;
  return <code>DcsBios::RotaryEncoder {idCamelCase(control.name)}("{control.name}", "-3200", "+3200", <b className="pinNo">PIN_A</b>, <b className="pinNo">PIN_B</b>);</code>;
}

function RotaryEncoderFixedStepSnippet(props: { control: TIOElement, input: TInputElement }) {
  let { control } = props;
  return <code>DcsBios::RotaryEncoder {idCamelCase(control.name)}("{control.name}", "DEC", "INC", <b className="pinNo">PIN_A</b>, <b className="pinNo">PIN_B</b>);</code>;
}

function PotentiometerSnippet(props: { control: TIOElement, input: TInputElement }) {
  let { control } = props;
  return <code>DcsBios::Potentiometer {idCamelCase(control.name)}("{control.name}", <b className="pinNo">PIN</b>);</code>;
}


// output code snippets:

// hex() converts a number into a four-digit
// lower case hexadecimal representation, prefixed
// by "0x".
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
  return <code>void {idCamelCase("ON_"+control.name)}Change(char* newValue) {'{'}<br />
    &nbsp;&nbsp;&nbsp;&nbsp;/* your code here */<br />
    {'}'}<br />
    DcsBios::StringBuffer&lt;{output.max_length}&gt; {idCamelCase(control.name)}Buffer({hex(output.address)}, {idCamelCase("ON_"+control.name)}Change);</code>
}

function IntegerBufferSnippet(props: { control: TIOElement, output: TOutputElement }) {
  let { control, output } = props;
  return <code>void {idCamelCase("ON_"+control.name)}Change(unsigned int newValue) {'{'}<br />
    &nbsp;&nbsp;&nbsp;&nbsp;/* your code here */<br />
    {'}'}<br />
    DcsBios::IntegerBuffer {idCamelCase(control.name)}Buffer({hex(output.address)}, {hex(output.mask)}, {output.shift_by.toString()}, {idCamelCase("ON_"+control.name)}Change);</code>
}



// Live Data
function LiveOutputData(props: { output: TOutputElement }) {
  if (props.output.type === "integer") {
    return LiveIntegerData(props)
  } else if (props.output.type === "string") {
    return LiveStringData(props)
  } else {
    return null;
  }
}

function LiveIntegerData(props: { output: TOutputElement }) {
  const output = props.output;
  const [hasValue, setHasValue] = useState(false)
  const [value, setValue] = useState(0)

  const liveDataCtx = useContext(LiveDataContext)

  useEffect(() => {
    const callback = (_: number, data: ArrayBuffer) => {
      setValue((new Uint16Array(data))[0])
      setHasValue(true)
    }
    liveDataCtx.subscribeExportCallback(output.address, callback);
    return (() => { liveDataCtx.unsubscribeExportCallback(callback); })
  }, [output.address, liveDataCtx]);

  var displayValue = value;
  displayValue &= output.mask;
  displayValue >>= output.shift_by;

  return (
    <div className="live-output">
      {hasValue ? displayValue : "no data yet"}
    </div>
  )
}

function LiveStringData(props: { output: TOutputElement }) {
  const output = props.output;
  const [hasValue, setHasValue] = useState(false)
  const [value, setValue] = useState(() => new Uint8Array(0))

  const liveDataCtx = useContext(LiveDataContext)

  useEffect(() => {
    let changed = false;
    const buffer = new ArrayBuffer(output.max_length)
    const bufferArray = new Uint8Array(buffer)
    const dataCallback = (address: number, data: ArrayBuffer) => {
      const offset = address - output.address;
      const newDataArray = new Uint8Array(data)
      bufferArray[offset] = newDataArray[0];
      if (offset < output.max_length - 1) {
        bufferArray[offset + 1] = newDataArray[1];
      }
      changed = true;
    }
    const endOfUpdateCallback = () => {
      if (!changed) return;
      setValue(new Uint8Array(bufferArray));
      setHasValue(true)
    }

    for (let i = 0; i <= output.address + output.max_length; i += 2) {
      liveDataCtx.subscribeExportCallback(output.address + i, dataCallback)
    }
    liveDataCtx.subscribeEndOfUpdateCallback(endOfUpdateCallback)

    return (() => {
      liveDataCtx.unsubscribeExportCallback(dataCallback)
      liveDataCtx.unsubscribeEndOfUpdateCallback(endOfUpdateCallback)
    })
  }, [output.max_length, output.address, liveDataCtx])

  let displayValue = "";
  if (hasValue) {
    var str = "";
    for (var i = 0; i < value.length; i++) {
      if (value[i] === 0) break;
      str = str + String.fromCharCode(value[i]);
    }
    displayValue = str
  }

  return (<div className="live-output">{hasValue ? displayValue.toString() : "no data yet"}</div>);
}

function LiveInputControls(props: { control: TIOElement }) {
  let controls: Array<ReactElement> = []

  for (let input of props.control.inputs) {
    if (input.interface === "action") {
      controls.push(<LiveActionInputControls key={input.interface} control={props.control} input={input} />)
    } else if (input.interface === "fixed_step") {
      controls.push(<LiveFixedStepInputControls key={input.interface} control={props.control} input={input} />)
    } else if (input.interface === "variable_step") {
      controls.push(<LiveVariableStepInputControls key={input.interface} control={props.control} input={input} />)
    } else if (input.interface === "set_state") {
      controls.push(<LiveSetStateInputControls key={input.interface} control={props.control} input={input} />)
    }
  }
  controls.sort((a, b) => {
    const order = ["action", "fixed_step", "variable_step", "set_state"]
    let aIdx = order.indexOf(a.key as string)
    let bIdx = order.indexOf(b.key as string)
    return aIdx - bIdx
  })

  return (
    <div className="live-controls">
      Commands:<br />
      {controls}
    </div>
  )
}


function LiveSetStateInputControls(props: { control: TIOElement, input: TInputElement }) {
  let { control, input } = props;
  const liveDataCtx = useContext(LiveDataContext)
  const [targetValue, setTargetValue] = useState(0);
  return (
    <div className="fixed-step-controls">
      <input type="range" value={targetValue} max={input.max_value} onChange={(e) => setTargetValue(parseInt(e.target.value))} /><br />
      <button onClick={() => liveDataCtx.sendInputData(control.name + " " + targetValue.toString())}>Set to {targetValue.toString()}</button>
    </div>
  )
}

function LiveActionInputControls(props: { control: TIOElement, input: TInputElement }) {
  const liveDataCtx = useContext(LiveDataContext)
  return (
    <div className="action-controls">
      <button onClick={() => liveDataCtx.sendInputData(props.control.name + " " + props.input.argument)}>{props.input.argument}</button>
    </div>
  );
}

function LiveFixedStepInputControls(props: { control: TIOElement, input: TInputElement }) {
  const liveDataCtx = useContext(LiveDataContext)
  return (
    <div className="fixed-step-controls">
      <button onClick={() => liveDataCtx.sendInputData(props.control.name + " DEC")}>DEC</button>
      <button onClick={() => liveDataCtx.sendInputData(props.control.name + " INC")}>INC</button>
    </div>
  )
}
function LiveVariableStepInputControls(props: { control: TIOElement, input: TInputElement }) {
  const liveDataCtx = useContext(LiveDataContext)
  const [delta, setDelta] = useState(props.input.suggested_step || 3200);
  return (
    <div className="variable-step-controls">
      <input type="range" min="0" max={props.input.max_value.toString()} value={delta.toString()} onChange={(e) => setDelta(parseInt(e.target.value))} /><br />
      <button onClick={() => liveDataCtx.sendInputData(props.control.name + " -" + delta.toString())}>-{delta}</button>
      <button onClick={() => liveDataCtx.sendInputData(props.control.name + " +" + delta.toString())}>+{delta}</button>
    </div>
  )
}

export { ControlReference }
