import assert from "node:assert";
import { describe, it } from "node:test";
import { createElement } from "react";
import { renderToString } from "react-dom/server";

import { Icon, isIconName } from "./Icon";

describe("Icon component (AC3)", () => {
  it("recognizes github, google, and facebook icons", () => {
    assert.strictEqual(isIconName("github"), true);
    assert.strictEqual(isIconName("google"), true);
    assert.strictEqual(isIconName("facebook"), true);
    assert.strictEqual(isIconName("unknown-provider"), false);
  });

  it("renders github, google, and facebook SVGs without error", () => {
    const githubHtml = renderToString(createElement(Icon, { name: "github" }));
    assert.match(githubHtml, /<svg/);
    assert.match(githubHtml, /<path d="M15 22/);

    const googleHtml = renderToString(createElement(Icon, { name: "google" }));
    assert.match(googleHtml, /<svg/);
    assert.match(googleHtml, /<path d="M17\.788/);

    const facebookHtml = renderToString(
      createElement(Icon, { name: "facebook" }),
    );
    assert.match(facebookHtml, /<svg/);
    assert.match(facebookHtml, /<path d="M18 2/);
  });
});
