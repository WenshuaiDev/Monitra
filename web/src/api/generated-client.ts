/**
 * This file was auto-generated from api/openapi.yaml.
 * Do not make direct changes to the file.
 */

import createClient from "openapi-fetch";

import type { HttpClient } from "../foundation/http-client";
import type { paths } from "./generated";

const startupHandshakePath = "/api/v1/startup-handshake" as const;

// createStartupClient binds the generated transport contract to Foundation's
// single application network executor.
export function createStartupClient(baseURL: string, httpClient: HttpClient) {
  const client = createClient<paths>({
    baseUrl: baseURL,
    fetch: (request) => httpClient.execute(request),
  });

  return {
    getStartupHandshake: () => client.GET(startupHandshakePath),
  };
}
