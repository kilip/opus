import assert from "node:assert";
import { describe, it } from "node:test";
import { createElement } from "react";
import { renderToString } from "react-dom/server";

import { Dialog } from "./Dialog";
import { ErrorState } from "./ErrorState";

describe("Dialog & ErrorState primitives (AC7)", () => {
  it("renders native dialog with aria-labelledby and children when open", () => {
    const html = renderToString(
      createElement(
        Dialog,
        { open: true, onClose: () => {}, titleId: "dlg-title" },
        createElement("h2", { id: "dlg-title" }, "Modal Heading"),
        createElement("input", { type: "text", "data-autofocus": true }),
      ),
    );

    assert.match(html, /<dialog/);
    assert.match(html, /aria-labelledby="dlg-title"/);
    assert.match(html, /Modal Heading/);
  });

  it("omits children when Dialog is closed", () => {
    const html = renderToString(
      createElement(
        Dialog,
        { open: false, onClose: () => {}, titleId: "dlg-title" },
        createElement("p", null, "Hidden Body"),
      ),
    );

    assert.match(html, /<dialog/);
    assert.doesNotMatch(html, /Hidden Body/);
  });

  it("renders ErrorState with role=alert and Retry button", () => {
    const html = renderToString(
      createElement(ErrorState, {
        title: "Load failed",
        message: "Network timeout",
        onRetry: () => {},
      }),
    );

    assert.match(html, /role="alert"/);
    assert.match(html, /Load failed/);
    assert.match(html, /Network timeout/);
    assert.match(html, /Retry/);
  });
});
