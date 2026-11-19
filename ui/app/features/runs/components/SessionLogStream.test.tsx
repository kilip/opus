import assert from "node:assert";
import { describe, it } from "node:test";
import { createElement } from "react";
import { renderToString } from "react-dom/server";

import {
  appendLogEntries,
  buildSessionEventsUrl,
  DEFAULT_MAX_LOG_ENTRIES,
  nextReconnectDelayMs,
  SessionLogStream,
} from "./SessionLogStream";

describe("SessionLogStream (AC9)", () => {
  it("renders role=log and aria-live=polite", () => {
    const html = renderToString(
      createElement(SessionLogStream, {
        sessionId: "sess-1",
        initialEntries: [{ seq: 1, kind: "stdout", payload: "hello" }],
      }),
    ).replaceAll("<!-- -->", "");

    assert.match(html, /role="log"/);
    assert.match(html, /aria-live="polite"/);
    assert.match(html, /Session sess-1/);
    assert.match(html, /hello/);
  });

  it("caps entries at maxEntries and renders older entries hidden notice", () => {
    const initialEntries = [
      { seq: 1, kind: "stdout", payload: "first" },
      { seq: 2, kind: "stdout", payload: "second" },
      { seq: 3, kind: "stdout", payload: "third" },
    ];
    const html = renderToString(
      createElement(SessionLogStream, {
        sessionId: "sess-1",
        initialEntries,
        maxEntries: 2,
      }),
    ).replaceAll("<!-- -->", "");

    assert.match(html, /1 older entries hidden/);
    assert.doesNotMatch(html, />first</);
    assert.match(html, /second/);
    assert.match(html, /third/);
  });

  it("computes exponential reconnect backoff capped at 30s and after_seq URL", () => {
    assert.strictEqual(DEFAULT_MAX_LOG_ENTRIES, 2000);
    assert.strictEqual(nextReconnectDelayMs(0), 1000);
    assert.strictEqual(nextReconnectDelayMs(1), 2000);
    assert.strictEqual(nextReconnectDelayMs(4), 16000);
    assert.strictEqual(nextReconnectDelayMs(5), 30000);
    assert.strictEqual(nextReconnectDelayMs(10), 30000);

    assert.strictEqual(
      buildSessionEventsUrl("s-1"),
      "/api/sessions/s-1/events",
    );
    assert.strictEqual(
      buildSessionEventsUrl("s-1", 42),
      "/api/sessions/s-1/events?after_seq=42",
    );

    const capped = appendLogEntries([{ seq: 1 }, { seq: 2 }], [{ seq: 3 }], 2);
    assert.strictEqual(capped.dropped, 1);
    assert.deepStrictEqual(capped.entries, [{ seq: 2 }, { seq: 3 }]);
  });
});
