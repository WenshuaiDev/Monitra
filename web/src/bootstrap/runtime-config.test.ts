import { describe, expect, test, vi } from "vitest";

import { loadRuntimeConfig, RuntimeConfigError } from "./runtime-config";

describe("runtime config loader", () => {
  test("loads the non-cacheable deployment config into the browser contract", async () => {
    const fetchConfig = vi.fn(async () =>
      new Response(
        JSON.stringify({
          expected_api_major: 1,
          expected_release_identity: "2026.08.06-test",
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );

    await expect(loadRuntimeConfig(fetchConfig)).resolves.toEqual({
      expectedApiMajor: 1,
      expectedReleaseIdentity: "2026.08.06-test",
    });
    expect(fetchConfig).toHaveBeenCalledWith("/runtime-config.json", {
      cache: "no-store",
      headers: { Accept: "application/json" },
    });
  });

  test.each([
    ["a missing field", { expected_api_major: 1 }],
    [
      "an unknown field",
      {
        expected_api_major: 1,
        expected_release_identity: "2026.08.06-test",
        api_base_url: "https://unexpected.test",
      },
    ],
    ["a non-positive API major", { expected_api_major: 0, expected_release_identity: "2026.08.06-test" }],
    ["a fractional API major", { expected_api_major: 1.5, expected_release_identity: "2026.08.06-test" }],
    ["an empty Release Identity", { expected_api_major: 1, expected_release_identity: "" }],
    ["an array document", []],
  ])("rejects %s", async (_name, document) => {
    const fetchConfig = vi.fn(async () =>
      new Response(JSON.stringify(document), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    await expect(loadRuntimeConfig(fetchConfig)).rejects.toBeInstanceOf(RuntimeConfigError);
  });
});
