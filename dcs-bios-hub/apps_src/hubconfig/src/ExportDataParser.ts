export default class ExportDataParser {
    private state: string = "WAIT_FOR_SYNC"
    private sync_byte_count: number = 0
    private address_buffer: ArrayBuffer
    private address_uint8: Uint8Array
    private address_uint16: Uint16Array
    private count_buffer: ArrayBuffer
    private count_uint8: Uint8Array
    private count_uint16: Uint16Array
    private data_buffer: ArrayBuffer
    private data_uint8: Uint8Array
    private data_uint16: Uint16Array
    
    private exportDataListeners: Array<{address: number, callback: (address: number, data: ArrayBuffer) => void}>
    private endOfUpdateListeners: Array<() => void>
  
    constructor() {
      this.address_buffer = new ArrayBuffer(2)
      this.address_uint8 = new Uint8Array(this.address_buffer)
      this.address_uint16 = new Uint16Array(this.address_buffer)
      this.count_buffer = new ArrayBuffer(2)
      this.count_uint8 = new Uint8Array(this.count_buffer)
      this.count_uint16 = new Uint16Array(this.count_buffer)
      this.data_buffer = new ArrayBuffer(2)
      this.data_uint8 = new Uint8Array(this.data_buffer)
      this.data_uint16 = new Uint16Array(this.data_buffer)
  
      this.exportDataListeners = new Array<{address: number, callback: (address: number, data: ArrayBuffer) => void}>();
      this.endOfUpdateListeners = new Array<() => void>();
    }
  
    public registerEndOfUpdateCallback(callback: () => void) {
      this.endOfUpdateListeners.push(callback)
    }
    public unregisterEndOfUpdateListener(callback: any) {
      this.endOfUpdateListeners = this.endOfUpdateListeners.filter(cb => cb !== callback)
    }
  
    public registerExportDataListener(address: number, callback: (address: number, data: ArrayBuffer) => void) {
      this.exportDataListeners.push({ address, callback })
    }
  
    public unregisterExportDataListener(callback: any) {
      this.exportDataListeners = this.exportDataListeners.filter(l => l.callback !== callback)
    }
  
    private notifyData(address: number, data: ArrayBuffer) {
      for (let l of this.exportDataListeners) {
        if (l.address === address) {
          l.callback(address, data)
        }
      }
    }
  
    private notifyEndOfUpdate() {
      for (let cb of this.endOfUpdateListeners) {
        cb()
      }
    }
  
    public processByte(c: number) {
      switch(this.state) {
              case "WAIT_FOR_SYNC":
              break;
              
              case "ADDRESS_LOW":
                  this.address_uint8[0] = c;
                  this.state = "ADDRESS_HIGH";
              break;
              
              case "ADDRESS_HIGH":
                  this.address_uint8[1] = c;
                  if (this.address_uint16[0] !== 0x5555) {
                      this.state = "COUNT_LOW";
                  } else {
                      this.state = "WAIT_FOR_SYNC";
                  }
              break;
              
              case "COUNT_LOW":
                  this.count_uint8[0] = c;
                  this.state = "COUNT_HIGH";
              break;
              
              case "COUNT_HIGH":
                  this.count_uint8[1] = c;
                  this.state = "DATA_LOW";
              break;
              
              case "DATA_LOW":
                  this.data_uint8[0] = c;
                  this.count_uint16[0]--;
                  this.state = "DATA_HIGH";
              break;
              
              case "DATA_HIGH":
                  this.data_uint8[1] = c;
                  this.count_uint16[0]--;
          this.notifyData(this.address_uint16[0], this.data_buffer)
                  this.address_uint16[0] += 2;
                  if (this.count_uint16[0] === 0) {
                      this.state = "ADDRESS_LOW";
                  } else {
                      this.state = "DATA_LOW";
                  }
              break;
                  
          }
          
          if (c === 0x55)
              this.sync_byte_count++;
          else
              this.sync_byte_count = 0;
              
          if (this.sync_byte_count === 4) {
              this.state = "ADDRESS_LOW";
              this.sync_byte_count = 0;
              this.notifyEndOfUpdate();
          }
    }
  }
  