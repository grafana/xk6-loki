import { check, fail } from 'k6';
import loki from 'k6/x/loki';

/*
 * Host name with port
 * @constant {string}
 */
const HOST = "xxx.grafana.net:3100";

/**
 * Name of the Loki tenant
 * @constant {string}
 */
const TENANT_ID = __ENV.LOKI_TENANT_ID || fail("provide LOKI_TENANT_ID when starting k6");

/**
 * Access token of the Loki tenant with logs:write and logs:read permissions
 * @constant {string}
 */
const ACCESS_TOKEN = __ENV.LOKI_ACCESS_TOKEN || fail("provide LOKI_ACCESS_TOKEN when starting k6");

/**
 * URL used for push and query requests
 * Path is automatically appended by the client
 * @constant {string}
 */
const BASE_URL = `https://${TENANT_ID}:${ACCESS_TOKEN}@${HOST}`;

/**
 * Minimum amount of virtual users (VUs)
 * @constant {number}
 */
const MIN_VUS = 10

/**
 * Maximum amount of virtual users (VUs)
 * @constant {number}
 */
const MAX_VUS = 100;

const KB = 1024;
const MB = KB * KB;

/**
 * Definition of test scenario
 */
export const options = {
  thresholds: {
    'http_req_failed': [{ threshold: 'rate==0.00', abortOnFail: true }],
  },
  scenarios: {
    loki_write: {
      executor: 'ramping-vus',
      exec: 'write',
      startVUs: MIN_VUS,
      stages: [
        { duration: '60s', target: MAX_VUS },
        { duration: '60s', target: MAX_VUS },
        { duration: '60s', target: MIN_VUS },
      ],
      gracefulRampDown: '10s',
    },
    loki_read: {
      executor: 'constant-vus',
      exec: 'read',
      duration: '180s',
      vus: MAX_VUS,
    },
  },
};

const labelCardinality = {
  "app": 5,
  "namespace": 10,
  "pod": 50,
};
const conf = new loki.Config(BASE_URL, 10000, 0.9, labelCardinality);
const client = new loki.Client(conf);

/**
 * Entrypoint for write scenario
 */
export function write() {
  let streams = randomInt(4, 8);
  let res = client.pushParametrized(streams, 800 * KB, 1 * MB);
  check(res,
    {
      'successful write': (res) => {
        let success = res.status == 204;
        if (!success) console.log(res.status, res.body);
        return success;
      },
    }
  );
};

/**
 * Entrypoint for read scenario
 */
export function read() {
  let limit = 1000;
  let duration = randomChoice(['1m', '5m', '10m', '15m']);

  let labelNames = readLabels(duration);
  if (labelNames == null || labelNames.length == 0) return;

  let labelValues = readLabelValues(labelNames, duration);
  if (labelValues == null || labelValues.length == 0) return;

  let app = randomChoice(labelValues.app);
  let namespace = randomChoice(labelValues.namespace);
  let format = randomChoice(labelValues.format);
  let pod = randomChoice(labelValues.pod);
  let instance = randomChoice(labelValues.instance);

  let queries = [
    `{app="${app}"} |= "GET" != "GET"`,
    `{namespace="${namespace}"} |~ "GET|POST"`,
    `{format="${format}"} | json | method = "GET"`,
    `{format="${format}"} | json | bytes > 10000`,
  ];
  readRange(queries, duration, limit);
  readInstant(queries, limit);

  let seriesSelector = [
    `{app="${app}"}`,
    `{namespace="${namespace}"}`,
    `{format="${format}"}`,
    `{pod="${pod}"}`,
    `{instance="${instance}"}`,
  ];
  readSeries(seriesSelector, duration);
};

/**
 * Execute labels query with given client
 */
function readLabels(range) {
  let res = client.labelsQuery(range);
  check(res, { 'successful labels query': (res) => res.status == 200 });
  if (res.status !== 200) return null;

  try {
    let data = JSON.parse(res.body);
    return data.data;
  } catch (e) {
    console.error(e);
  }
  return null
};

/**
 * Execute label values query with given client
 */
function readLabelValues(labels, range) {
  let labelValues = {};

  labels.forEach((label) => {
    if (label == '__name__') return;
    let res = client.labelValuesQuery(label, range);
    check(res, { 'successful label values query': (res) => res.status == 200 });
    if (res.status !== 200) return null;

    try {
      let data = JSON.parse(res.body);
      labelValues[label] = data.data;
    } catch (e) {
      console.error(e);
      return null;
    }
  });

  return labelValues;
};

/**
 * Execute instant query with given client
 */
function readInstant(queries, limit) {
  queries.forEach((q) => {
    let res = client.instantQuery(q, limit);
    check(res,
      {
        'successful instant query': (res) => {
          let success = res.status == 200;
          if (!success) console.log(res.status, res.body);
          return success;
        },
      }
    );
  });
};

/**
 * Execute range query with given client
 */
function readRange(queries, duration, limit) {
  queries.forEach((q) => {
    let res = client.rangeQuery(q, duration, limit);
    check(res,
      {
        'successful range query': (res) => {
          let success = res.status == 200;
          if (!success) console.log(res.status, res.body);
          return success;
        },
      }
    );
  });
};

/**
 * Execute range query with given client
 */
function readSeries(queries, duration) {
  queries.forEach((q) => {
    let res = client.seriesQuery(q, duration);
    check(res,
      {
        'successful range query': (res) => {
          let success = res.status == 200;
          if (!success) console.log(res.status, res.body);
          return success;
        },
      }
    );
  });
};

/**
 * Return an item of random choice of a list
 */
function randomChoice(items) {
  return items[Math.floor(Math.random() * items.length)];
};

/**
 * Return a random integer between min and max including min and max
 */
function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1) + min);
};

/**
 * Return a random float between min and max including min and max
 */
function randomFloat(min, max) {
  return Math.random() * (max - min + 1) + min;
};
