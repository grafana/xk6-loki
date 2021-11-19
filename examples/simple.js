import sleep from 'k6';
import loki from 'k6/x/loki';

/**
 * URL used for push and query requests
 * Path is automatically appended by the client
 * @constant {string}
 */
const BASE_URL = `http://localhost:3100`;


export const options = {
  vus: 10,
  iterations: 10,
};

const conf = new loki.Config(BASE_URL);
const client = new loki.Client(conf);

export default () => {
  client.push()
  sleep(1)
}
