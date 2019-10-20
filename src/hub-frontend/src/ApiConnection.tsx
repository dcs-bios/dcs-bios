import websocket, { w3cwebsocket } from "websocket";

function getApiHostPart() {
  if (window.location.port === "3000") {
    // assume we are served by the webpack dev server and the API is found on a different port
    return window.location.hostname+':5010';
  } else {
    // otherwise connect to the same host:port that the site is being served from
    return window.location.host;
  }
}

export function getApiConnection(): websocket.w3cwebsocket {
    return new w3cwebsocket('ws://'+getApiHostPart()+'/api/websocket')
}

type ApiJsonMessage = {
  datatype: string
  data: any
}

export function apiPost(message: ApiJsonMessage): Promise<any> {
  var request = {
    method: 'POST',
    body: JSON.stringify(message) as unknown as ReadableStream<Uint8Array>
  } as Request
  return fetch('http://'+getApiHostPart()+'/api/postjson', request).then((resp, ) => {
    if (!resp.ok) {
      console.log("/api/postjson: server responded with error", resp)
      throw resp
    }
    
    return resp.json()
  })
}
