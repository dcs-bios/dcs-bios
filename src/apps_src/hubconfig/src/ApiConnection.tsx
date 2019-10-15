import websocket, { w3cwebsocket } from "websocket";

function getApiConnection(): websocket.w3cwebsocket {
  if (window.location.port === "3000") {
    // assume we are served by the webpack dev server and the API is found on a different port
    return new w3cwebsocket('ws://'+window.location.hostname+':5010/api/websocket')
  } else {
    // otherwise connect to the same host:port that the site is being served from
    return new w3cwebsocket('ws://'+window.location.host+'/api/websocket')
  }
}

export default getApiConnection
