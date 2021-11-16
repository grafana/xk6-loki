# xk6-loki

**A k6 extension for pushing logs to Loki.**


## Getting started

1. Install `xk6`

   ```bash
   go install go.k6.io/xk6/cmd/xk6@latest
   ```

1. Checkout `grafana/xk6-loki`

   ```bash
   git clone github.com/grafana/xk6-loki
   cd xk6-loki
   ```

1. Build the extension

   ```bash
   make k6
   ```

## Example

```javascript
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

```

```bash
./k6 run examples/simple.js
```
