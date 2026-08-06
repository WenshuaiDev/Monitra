import { describe, expect, test } from "vitest";

import type { HttpClient } from "../foundation/http-client";
import { createStartupClient } from "./generated-client";

class RecordingHttpClient implements HttpClient {
  request: Request | undefined;

  async execute(request: Request): Promise<Response> {
    this.request = request;
    return new Response(
      JSON.stringify({
        code: "FOUNDATION_STARTUP_READY",
        message: "startup handshake succeeded",
        data: {
          release_identity: "2026.08.06-test",
          api_major: 1,
        },
        request_id: "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
      }),
      {
        status: 200,
        headers: {
          "Content-Type": "application/json",
          "X-Request-ID": "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
        },
      },
    );
  }
}

describe("startup client", () => {
  test("executes the generated operation through Foundation HttpClient", async () => {
    const httpClient = new RecordingHttpClient();
    const client = createStartupClient("https://monitra.test", httpClient);

    const result = await client.getStartupHandshake();

    expect(httpClient.request?.method).toBe("GET");
    expect(httpClient.request?.url).toBe("https://monitra.test/api/v1/startup-handshake");
    expect(result.error).toBeUndefined();
    expect(result.data).toEqual({
      code: "FOUNDATION_STARTUP_READY",
      message: "startup handshake succeeded",
      data: {
        release_identity: "2026.08.06-test",
        api_major: 1,
      },
      request_id: "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
    });
  });
});
