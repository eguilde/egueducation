import { contractServerBaseUrl } from './client';

describe('contractServerBaseUrl', () => {
  it.each([
    ['/api', 'https://app.example'],
    ['/gateway/api/', 'https://app.example/gateway'],
    ['https://school.example/api', 'https://school.example'],
  ])('maps %s to the OpenAPI server base %s', (input, expected) => {
    expect(contractServerBaseUrl(input, 'https://app.example')).toBe(expected);
  });
});
