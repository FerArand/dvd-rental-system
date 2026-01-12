
const { contextBridge } = require('electron');
contextBridge.exposeInMainWorld('env', {
  API_BASE: process.env.API_BASE || 'http://localhost:8080'
});
