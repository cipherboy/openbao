## Module Report
### Unknown Global

**Global**: `Ember.testing`

**Location**: `app/components/auth-form.js` at line 257

```js

  delayAuthMessageReminder: task(function* () {
    if (Ember.testing) {
      yield timeout(0);
    } else {
```

### Unknown Global

**Global**: `Ember.testing`

**Location**: `app/components/oidc-consent-block.js` at line 52

```js
    const { redirect, ...params } = this.args;
    const redirectUrl = this.buildUrl(redirect, params);
    if (Ember.testing) {
      this.args.testRedirect(redirectUrl.toString());
    } else {
```

### Unknown Global

**Global**: `Ember.testing`

**Location**: `app/helpers/-date-base.js` at line 14

```js

  compute(value, { interval }) {
    if (Ember.testing) {
      // issues with flaky test, suspect it has to the do with the run loop not being cleared as intended farther down.
      return;
```

### Unknown Global

**Global**: `Ember.testing`

**Location**: `app/routes/vault.js` at line 12

```js
import Ember from 'ember';
/* eslint-disable ember/no-ember-testing-in-module-scope */
const SPLASH_DELAY = Ember.testing ? 0 : 300;

export default Route.extend({
```

### Unknown Global

**Global**: `Ember.testing`

**Location**: `app/routes/vault/cluster/logout.js` at line 40

```js
      queryParams.namespace = ns;
    }
    if (Ember.testing) {
      // Don't redirect on the test
      this.replaceWith('vault.cluster.auth', { queryParams });
```

### Unknown Global

**Global**: `Ember.onerror`

**Location**: `tests/helpers/wait-for-error.js` at line 10

```js

export default function waitForError(opts) {
  const orig = Ember.onerror;

  let error = null;
```

### Unknown Global

**Global**: `Ember.onerror`

**Location**: `tests/helpers/wait-for-error.js` at line 10

```js

export default function waitForError(opts) {
  const orig = Ember.onerror;

  let error = null;
```

### Unknown Global

**Global**: `Ember.onerror`

**Location**: `tests/helpers/wait-for-error.js` at line 13

```js

  let error = null;
  Ember.onerror = (err) => {
    error = err;
  };
```

### Unknown Global

**Global**: `Ember.onerror`

**Location**: `tests/helpers/wait-for-error.js` at line 18

```js

  return waitUntil(() => error, opts).finally(() => {
    Ember.onerror = orig;
  });
}
```

### Unknown Global

**Global**: `Ember.Test`

**Location**: `tests/acceptance/not-found-test.js` at line 19

```js

  hooks.beforeEach(function () {
    adapterException = Ember.Test.adapter.exception;
    Ember.Test.adapter.exception = () => {};
    return authPage.login();
```

### Unknown Global

**Global**: `Ember.Test`

**Location**: `tests/acceptance/not-found-test.js` at line 20

```js
  hooks.beforeEach(function () {
    adapterException = Ember.Test.adapter.exception;
    Ember.Test.adapter.exception = () => {};
    return authPage.login();
  });
```

### Unknown Global

**Global**: `Ember.Test`

**Location**: `tests/acceptance/not-found-test.js` at line 25

```js

  hooks.afterEach(function () {
    Ember.Test.adapter.exception = adapterException;
    return logout.visit();
  });
```
