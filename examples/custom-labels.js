import { check, sleep } from 'k6';
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

const KB = 1024;
const MB = KB * KB;

export const options = {
  vus: 1,
  iterations: 100,
};

const labels = loki.Labels({
  "format": ["logfmt"], // must contain at least one of the supported log formats
  "os": ["linux"],
  "cluster": ["k3d", "minikube"],
  "namespace": ["loki-prod", "loki-dev"],
  "container": ["distributor", "ingester", "querier", "query-frontend", "query-scheduler", "index-gateway", "compactor"],
  "instance": ["localhost"], // overrides the `instance` label which is otherwise derived from the hostname and VU
});

const conf = new loki.Config(BASE_URL, 10000, 1.0, {}, labels);
const client = new loki.Client(conf);

export default () => {
  let res = client.pushParameterized(10, 1 * MB, 2 * MB);
  check(res,
    {
      'successful write': (res) => {
        let success = res.status == 204;
        if (!success) console.log("write", res.status, res.body);
        return success;
      },
    }
  );
  let resp = client.labelsQuery("1m").body;
  let labels = JSON.parse(resp).data;
  labels.forEach((label) => {
    res = client.labelValuesQuery(label, "1m")
    check(res,
      {
        'successful read': (res) => {
          let success = res.status == 200;
          if (!success) console.log("read", label, res.status, res.body);
          return success;
        },
      }
    );
  })
  sleep(1);
};
