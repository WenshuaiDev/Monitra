import { useEffect, useRef, useState } from "react";
import { RouterProvider, type createBrowserRouter } from "react-router-dom";

import type { HttpClient } from "../foundation/http-client";
import type { RuntimeConfig } from "./runtime-config";

export interface StartupIdentity {
  releaseIdentity: string;
  apiMajor: number;
  requestID: string;
}

interface StartupHandshakeResponse {
  code: "FOUNDATION_STARTUP_READY";
  message: "startup handshake succeeded";
  data: {
    release_identity: string;
    api_major: number;
  };
  request_id: string;
}

export interface StartupHandshakeClient {
  getStartupHandshake(): Promise<{
    data?: unknown;
    error?: unknown;
  }>;
}

type ApplicationRouter = ReturnType<typeof createBrowserRouter>;

export interface BootstrapDependencies {
  loadRuntimeConfig(): Promise<RuntimeConfig>;
  createHttpClient(): HttpClient;
  createStartupClient(httpClient: HttpClient): StartupHandshakeClient;
  createRouter(identity: StartupIdentity): ApplicationRouter;
}

type BootstrapState =
  | { status: "loading" }
  | { status: "ready"; router: ApplicationRouter }
  | { status: "failure"; kind: "configuration" | "backend" }
  | { status: "failure"; kind: "api"; expected: number; actual: number }
  | { status: "failure"; kind: "release"; expected: string; actual: string };

export function Bootstrap({ dependencies }: { dependencies: BootstrapDependencies }) {
  const [state, setState] = useState<BootstrapState>({ status: "loading" });
  const started = useRef(false);

  useEffect(() => {
    if (started.current) {
      return;
    }
    started.current = true;

    void startApplication(dependencies).then(setState);
  }, [dependencies]);

  if (state.status === "ready") {
    return <RouterProvider router={state.router} />;
  }

  if (state.status === "failure") {
    if (state.kind === "release") {
      return (
        <main role="alert">
          <h1>Release Identity is incompatible</h1>
          <p>
            Expected Release Identity {state.expected} but received {state.actual}.
          </p>
        </main>
      );
    }
    if (state.kind === "api") {
      return (
        <main role="alert">
          <h1>API version is incompatible</h1>
          <p>
            Expected API major {state.expected} but received {state.actual}.
          </p>
        </main>
      );
    }
    if (state.kind === "backend") {
      return (
        <main role="alert">
          <h1>Backend is unavailable</h1>
          <p>The startup handshake could not be completed.</p>
        </main>
      );
    }
    return (
      <main role="alert">
        <h1>Runtime configuration is invalid</h1>
        <p>Check the deployment-time browser configuration.</p>
      </main>
    );
  }

  return <p role="status">Starting Monitra…</p>;
}

async function startApplication(dependencies: BootstrapDependencies): Promise<BootstrapState> {
  let config: RuntimeConfig;
  try {
    config = await dependencies.loadRuntimeConfig();
  } catch {
    return { status: "failure", kind: "configuration" };
  }
  let result: Awaited<ReturnType<StartupHandshakeClient["getStartupHandshake"]>>;
  try {
    const httpClient = dependencies.createHttpClient();
    const client = dependencies.createStartupClient(httpClient);
    result = await client.getStartupHandshake();
  } catch {
    return { status: "failure", kind: "backend" };
  }
  if (!isStartupHandshakeResponse(result.data)) {
    return { status: "failure", kind: "backend" };
  }
  if (result.data.data.api_major !== config.expectedApiMajor) {
    return {
      status: "failure",
      kind: "api",
      expected: config.expectedApiMajor,
      actual: result.data.data.api_major,
    };
  }
  if (result.data.data.release_identity !== config.expectedReleaseIdentity) {
    return {
      status: "failure",
      kind: "release",
      expected: config.expectedReleaseIdentity,
      actual: result.data.data.release_identity,
    };
  }

  return {
    status: "ready",
    router: dependencies.createRouter({
      releaseIdentity: result.data.data.release_identity,
      apiMajor: result.data.data.api_major,
      requestID: result.data.request_id,
    }),
  };
}

function isStartupHandshakeResponse(value: unknown): value is StartupHandshakeResponse {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const response = value as Record<string, unknown>;
  if (
    response.code !== "FOUNDATION_STARTUP_READY" ||
    response.message !== "startup handshake succeeded" ||
    typeof response.request_id !== "string" ||
    response.request_id.length === 0 ||
    typeof response.data !== "object" ||
    response.data === null
  ) {
    return false;
  }

  const data = response.data as Record<string, unknown>;
  return (
    typeof data.release_identity === "string" &&
    data.release_identity.length > 0 &&
    typeof data.api_major === "number" &&
    Number.isSafeInteger(data.api_major) &&
    data.api_major > 0
  );
}
