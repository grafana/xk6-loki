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
 * Amount of virtual users (VUs)
 * @constant {number}
 */
const VUS = 100;

/**
 * Definition of test scenario
 */
export const options = {
  thresholds: {
    'http_req_failed': [{ threshold: 'rate<=0.01', abortOnFail: true }],
  },
  scenarios: {
    read: {
      executor: 'constant-vus',
      exec: 'read',
      duration: '180s',
      vus: VUS,
    },
  },
};

const labelCardinality = {
  "app": 5,
  "namespace": 1,
  "pod": 10,
};
const conf = new loki.Config(BASE_URL, 10000, 0.9, labelCardinality);
const client = new loki.Client(conf);

const createSelectorByRatio = (ratioConfig) => {
  let ratioSum = 0;
  const executorsIntervals = [];
  for (let i = 0; i < ratioConfig.length; i++) {
    executorsIntervals.push({
      start: ratioSum,
      end: ratioSum + ratioConfig[i].ratio,
      item: ratioConfig[i].item,
    })
    ratioSum += ratioConfig[i].ratio
  }
  return (random) => {
    if (random >= 1 || random < 0) {
      fail(`random value must be within range [0-1)`)
    }
    const value = random * ratioSum;
    for (let i = 0; i < executorsIntervals.length; i++) {
      let currentInterval = executorsIntervals[i];
      if (value < currentInterval.end && value >= currentInterval.start) {
        return currentInterval.item
      }
    }
  }
}

const queryTypeRatioConfig = [
  {
    ratio: 0.1,
    item: readLabels
  },
  {
    ratio: 0.1,
    item: readLabelValues
  },
  {
    ratio: 0.1,
    item: readSeries
  },
  {
    ratio: 0.5,
    item: readRange
  },
  {
    ratio: 0.2,
    item: readInstant
  },
];

const selectQueryTypeByRatio = createSelectorByRatio(queryTypeRatioConfig);

const rangesRatioConfig = [
  {
    ratio: 0.2,
    item: '15m'
  },
  {
    ratio: 0.2,
    item: '30m'
  },
  {
    ratio: 0.3,
    item: '1h'
  },
  {
    ratio: 0.2,
    item: '3h'
  },
  {
    ratio: 0.1,
    item: '12h'
  },
];

const selectRangeByRatio = createSelectorByRatio(rangesRatioConfig);

/**
 * Entrypoint for read scenario
 */
export function read() {
  selectQueryTypeByRatio(Math.random())();
}

/**
 * Execute labels query with given client
 */
function readLabels() {
  const range = selectRangeByRatio(Math.random())
  let res = client.labelsQuery(range);
  check(res, {'successful labels query': (res) => res.status === 200});
}

/**
 * Execute label values query with given client
 */
function readLabelValues() {
  const label = randomChoice(Object.keys(conf.labels))
  const range = selectRangeByRatio(Math.random())
  let res = client.labelValuesQuery(label, range);
  check(res, {'successful label values query': (res) => res.status === 200});
}

const limit = 1000;

const instantQuerySuppliers = [
  () => `rate({app="${randomChoice(conf.labels.app)}"}[5m])`,
  () => `sum by (namespace) (rate({app="${randomChoice(conf.labels.app)}"} [5m]))`,
  () => `sum by (namespace) (rate({app="${randomChoice(conf.labels.app)}"} |~ ".*a" [5m]))`,
  () => `sum by (namespace) (rate({app="${randomChoice(conf.labels.app)}"} |= "USB" [5m]))`,
  () => `sum by (status) (rate({app="${randomChoice(conf.labels.app)}"} | json | __error__ = "" [5m]))`,
  () => `sum by (_client) (rate({app="${randomChoice(conf.labels.app)}"} | logfmt | __error__=""  | _client="" [5m]))`,
  () => `sum by (namespace) (sum_over_time({app="${randomChoice(conf.labels.app)}"} | json | __error__ = "" | unwrap bytes [5m]))`,
  () => `quantile_over_time(0.99, {app="${randomChoice(conf.labels.app)}"} | json | __error__ = "" | unwrap bytes [5m]) by (namespace)`,
];

/**
 * Execute instant query with given client
 */
function readInstant() {
  const query = randomChoice(rangeQuerySuppliers)()
  let res = client.instantQuery(query, limit);
  check(res,
    {
      'successful instant query': (res) => {
        let success = res.status === 200;
        if (!success) console.log(res.status, res.body);
        return success;
      },
    }
  );
}

const rangeQuerySuppliers = [
  ...instantQuerySuppliers,
  () => `{app="${randomChoice(conf.labels.app)}"}`,
  () => `{app="${randomChoice(conf.labels.app)}"} |= "USB" != "USB"`,
  () => `{app="${randomChoice(conf.labels.app)}"} |~ "US.*(a|o)"`,
  () => `{app="${randomChoice(conf.labels.app)}", format="json"} | json | status < 300`,
]

/**
 * Execute range query with given client
 */
function readRange() {
  const query = randomChoice(instantQuerySuppliers)()
  let range = selectRangeByRatio(Math.random());
  let res = client.rangeQuery(query, range, limit);
  check(res,
    {
      'successful range query': (res) => {
        let success = res.status === 200;
        if (!success) console.log(res.status, res.body);
        return success;
      },
    }
  );
}

let seriesSelectorSuppliers = [
  () => `{app="${randomChoice(conf.labels.app)}"}`,
  () => `{namespace="${randomChoice(conf.labels.namespace)}"}`,
  () => `{format="${randomChoice(conf.labels.format)}"}`,
  () => `{pod="${randomChoice(conf.labels.pod)}"}`,
];

/**
 * Execute range query with given client
 */
function readSeries() {
  let range = selectRangeByRatio(Math.random());
  let selector = randomChoice(seriesSelectorSuppliers)();
  let res = client.seriesQuery(selector, range);
  check(res,
    {
      'successful series query': (res) => {
        let success = res.status === 200;
        if (!success) console.log(res.status, res.body);
        return success;
      },
    }
  );
}

/**
 * Return an item of random choice of a list
 */
function randomChoice(items) {
  return items[Math.floor(Math.random() * items.length)];
}
