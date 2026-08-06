import { describe, expect, test, vi } from "vitest";

import { BrowserHttpClient } from "./http-client";

describe("Foundation HttpClient", () => {
  test("executes an application request through the injected browser transport", async () => {
    const expectedResponse = new Response("{}", { status: 200 });
    const executeFetch = vi.fn(async (_request: Request) => expectedResponse);
    const client = new BrowserHttpClient(executeFetch, 5_000);
    const request = new Request("https://monitra.test/api/v1/startup-handshake");

    await expect(client.execute(request)).resolves.toBe(expectedResponse);
    expect(executeFetch).toHaveBeenCalledOnce();
    expect(executeFetch.mock.calls[0]?.[0]).toBeInstanceOf(Request);
    expect(executeFetch.mock.calls[0]?.[0].url).toBe(request.url);
  });
});
