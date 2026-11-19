import assert from "node:assert";
import { describe, it } from "node:test";
import { createElement } from "react";
import { renderToString } from "react-dom/server";

import { Chatter, type ChatterMessage } from "./Chatter";

describe("Chatter", () => {
  it("renders timeline messages in reverse chronological order and action buttons", () => {
    const messages: ChatterMessage[] = [
      {
        id: "m1",
        kind: "message",
        author: "Alice",
        body: "First message",
        at: "2026-01-01T10:00:00Z",
      },
      {
        id: "m2",
        kind: "tracking",
        body: "Moved to Ready",
        at: "2026-01-01T11:00:00Z",
      },
    ];

    const html = renderToString(
      createElement(Chatter, {
        messages,
        onPost: () => {},
      }),
    );

    assert.match(html, /Send message/);
    assert.match(html, /Log note/);
    assert.match(html, /Alice/);
    assert.match(html, /First message/);
    assert.match(html, /Moved to Ready/);
    assert.ok(html.indexOf("Moved to Ready") < html.indexOf("First message"));
  });
});
