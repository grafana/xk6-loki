import {sleep, check} from 'k6';
import loki from 'k6/x/loki';

/*
 * Host name with port
 * @constant {string}
 */
const HOST = __ENV.LOKI_ADDR || fail("provide LOKI_ADDR when starting k6");

/**
 * Optional name of the Loki tenant
 * @constant {string}
 */
const TENANT_ID = __ENV.LOKI_TENANT_ID || '';

/**
 * Optional access token of the Loki tenant with logs:write and logs:read permissions
 * @constant {string}
 */
const ACCESS_TOKEN = __ENV.LOKI_ACCESS_TOKEN || '';

/**
 * Configures the protocol scheme used for requests.
 * @constant {string}
 */
const SCHEME = __ENV.K6_SCHEME || 'http';

/**
 * URL used for push and query requests
 * Path is automatically appended by the client
 * @constant {string}
 */
const BASE_URL = `${SCHEME}://${TENANT_ID}:${ACCESS_TOKEN}@${HOST}`;

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

/**
 * "main" function for each VU iteration
 */
export default () => {
  // Push request with 10 streams and uncompressed logs between 800KB and 2MB
  var res = client.pushParameterized(10, 800 * KB, 2 * MB);
  // Check for successful write
  check(res, { 'successful write': (res) => res.status == 204 });

  // Pick a random log format from label pool
  let format = randomChoice(conf.labels["format"]);

  // Execute instant query with limit 1
  res = client.instantQuery(`count_over_time({format="${format}"}[1m])`, 1)
  // Check for successful read
  check(res, { 'successful instant query': (res) => res.status == 200 });

  // Execute range query over last 5m and limit 1000
  res = client.rangeQuery(`{format="${format}"}`, "5m", 1000)
  // Check for successful read
  check(res, { 'successful range query': (res) => res.status == 200 });

  // Wait before next iteration
  sleep(1);
}

/**
 * Helper function to get random item from array
 */
function randomChoice(items) {
  return items[Math.floor(Math.random() * items.length)];
}
