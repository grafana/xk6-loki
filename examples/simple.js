import sleep from 'k6';
import loki from 'k6/x/loki';

/**
 * Full URL of used for push requests
 * @constant {string}
 */
const PUSH_URL = `http://localhost:3100/loki/api/v1/push`;


export const options = {
  vus: 10,
  iterations: 10,
};

const conf = new loki.Config(PUSH_URL);
const client = new loki.Client(conf);

export default () => {
  client.push()
  sleep(1)
}
