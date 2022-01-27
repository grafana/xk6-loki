import {sleep, check} from 'k6';
import loki from 'k6/x/loki';

/**
 * URL used for push and query requests
 * Path is automatically appended by the client
 * @constant {string}
 */
const BASE_URL = `http://localhost:3100`;

/**
 * Helper constant for byte values
 * @constant {number}
 */
const KB = 1024;

/**
 * Helper constant for byte values
 * @constant {number}
 */
const MB = KB * KB;

/**
 * Instantiate config and Loki client
 */
const conf = new loki.Config(BASE_URL);
const client = new loki.Client(conf);

/**
 * Define test scenario
 */
export const options = {
  vus: 10,
  iterations: 10,
};

export default () => {
  var res = client.pushParameterized(2, 500 * KB, 1 * MB);
  check(res, { 'successful write': (res) => res.status == 204 });
  sleep(1);
}
