// Local acceptance server: only deterministic echo; no filesystem or network writes.
const readline = require('node:readline');
const path = require('node:path');
const input = readline.createInterface({input: process.stdin});
input.on('line', line => {
  const request = JSON.parse(line);
  if (request.id === undefined) return;
  let result;
  switch (request.method) {
    case 'initialize':
      result = {protocolVersion: request.params.protocolVersion, capabilities: {tools: {}}, serverInfo: {name: 'mygo-fixture', version: '1.0.0'}};
      break;
    case 'ping': result = {}; break;
    case 'tools/list':
      result = {tools: [{name: 'echo', description: 'Owned acceptance fixture', inputSchema: {type: 'object', properties: {message: {type: 'string'}}, required: ['message']}}]};
      break;
    case 'tools/call':
      result = {content: [{type: 'text', text: `fixture:${request.params.arguments.message}:${path.basename(process.cwd())}:${process.env.TEST_TOKEN === 'fixture-secret'}`}]};
      break;
    default:
      process.stdout.write(JSON.stringify({jsonrpc: '2.0', id: request.id, error: {code: -32601, message: 'unknown fixture method'}}) + '\n');
      return;
  }
  process.stdout.write(JSON.stringify({jsonrpc: '2.0', id: request.id, result}) + '\n');
});
