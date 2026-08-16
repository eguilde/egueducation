const spec = require('../../openapi/openapi.json');
const seen = new Set(); let failures = [];
function walk(schema, where) {
  if (!schema || typeof schema !== 'object') return;
  if (Object.keys(schema).length === 0) failures.push(`${where} (empty schema)`);
  if (schema.$ref) { const name=schema.$ref.split('/').pop(); if(seen.has(`${where}:${name}`))return; seen.add(`${where}:${name}`); return walk(spec.components.schemas[name], where); }
  if (schema.type === 'object' && schema.additionalProperties === true && !schema['x-free-form-property']) failures.push(where);
  if (schema.type === 'object' && !schema['x-free-form-property'] && !schema.properties && schema.additionalProperties !== false) failures.push(`${where} (open object)`);
  for (const part of [...(schema.allOf||[]), ...(schema.oneOf||[]), ...(schema.anyOf||[])]) walk(part, where);
  if (schema.items) walk(schema.items, where);
  for (const value of Object.values(schema.properties||{})) walk(value, where);
}
for(const [path,item] of Object.entries(spec.paths)) for(const [method,op] of Object.entries(item)) for(const [status,response] of Object.entries(op.responses||{})) if(/^2\d\d$/.test(status)) for(const media of Object.values(response.content||{})) walk(media.schema, `${method.toUpperCase()} ${path} ${status}`);
if(failures.length){console.error([...new Set(failures)].join('\n'));process.exit(1)}
console.log('closed-success-response-objects 0 unrestricted');
