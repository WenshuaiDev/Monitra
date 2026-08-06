#!/usr/bin/env node

import { mkdir, readFile, writeFile } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import openapiTS, { astToString } from "openapi-typescript";
import { parse } from "yaml";

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const schemaPath = join(repositoryRoot, "api", "openapi.yaml");
const outputDirectory = resolve(repositoryRoot, process.argv[2] ?? "web/src/api");
const schemaSource = await readFile(schemaPath, "utf8");
const document = parse(schemaSource);
const handshakePaths = Object.entries(document.paths ?? {}).filter(
  ([, pathItem]) => pathItem?.get?.operationId === "getStartupHandshake",
);

if (handshakePaths.length !== 1) {
  throw new Error(`expected one getStartupHandshake operation, found ${handshakePaths.length}`);
}

const [handshakePath] = handshakePaths[0];
const typeNodes = await openapiTS(pathToFileURL(schemaPath));
const generatedClient = `/**
 * This file was auto-generated from api/openapi.yaml.
 * Do not make direct changes to the file.
 */

import createClient from "openapi-fetch";

import type { HttpClient } from "../foundation/http-client";
import type { paths } from "./generated";

const startupHandshakePath = ${JSON.stringify(handshakePath)} as const;

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
`;

await mkdir(outputDirectory, { recursive: true });
await Promise.all([
  writeFile(join(outputDirectory, "generated.ts"), astToString(typeNodes), "utf8"),
  writeFile(join(outputDirectory, "generated-client.ts"), generatedClient, "utf8"),
]);
