// @vitest-environment jsdom

import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen } from "@testing-library/react";
import { createMemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, test, vi } from "vitest";

import { BrowserHttpClient, type HttpClient } from "../foundation/http-client";
import { createApplicationRouter } from "../router/application-router";
import {
  Bootstrap,
  type BootstrapDependencies,
  type StartupHandshakeClient,
} from "./bootstrap";
import { RuntimeConfigError } from "./runtime-config";

const compatibleHandshake = {
  code: "FOUNDATION_STARTUP_READY" as const,
  message: "startup handshake succeeded" as const,
  data: {
    release_identity: "2026.08.06-test",
    api_major: 1,
  },
  request_id: "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
};

afterEach(cleanup);

function createDependencies(
  overrides: Partial<BootstrapDependencies> = {},
): BootstrapDependencies {
  return {
    loadRuntimeConfig: async () => ({
      expectedApiMajor: 1,
      expectedReleaseIdentity: "2026.08.06-test",
    }),
    createHttpClient: () => ({ execute: vi.fn() }),
    createStartupClient: () => ({
      getStartupHandshake: async () => ({ data: compatibleHandshake }),
    }),
    createRouter: () => createMemoryRouter([{ path: "*", element: null }]),
    ...overrides,
  };
}

describe("browser bootstrap", () => {
  test("loads config before networking and creates one Router only after a compatible handshake", async () => {
    const events: string[] = [];
    const httpClient: HttpClient = {
      execute: vi.fn(),
    };
    const startupClient: StartupHandshakeClient = {
      getStartupHandshake: async () => {
        events.push("handshake");
        return { data: compatibleHandshake };
      },
    };
    const createRouter = vi.fn((identity) => {
      events.push("router");
      return createApplicationRouter(identity);
    });
    const dependencies = createDependencies({
      loadRuntimeConfig: async () => {
        events.push("config");
        return {
          expectedApiMajor: 1,
          expectedReleaseIdentity: "2026.08.06-test",
        };
      },
      createHttpClient: () => {
        events.push("http-client");
        return httpClient;
      },
      createStartupClient: (createdHttpClient) => {
        expect(createdHttpClient).toBe(httpClient);
        events.push("startup-client");
        return startupClient;
      },
      createRouter,
    });

    render(<Bootstrap dependencies={dependencies} />);

    expect(await screen.findByRole("heading", { name: "Monitra is ready" })).toBeVisible();
    expect(screen.getByText("2026.08.06-test")).toBeVisible();
    expect(screen.getByText("1")).toBeVisible();
    expect(screen.getByText("ABCDEFGHIJKLMNOPQRSTUVWXYZ")).toBeVisible();
    expect(events).toEqual(["config", "http-client", "startup-client", "handshake", "router"]);
    expect(createRouter).toHaveBeenCalledOnce();
  });

  test("shows a configuration failure without creating HttpClient or Router", async () => {
    const createHttpClient = vi.fn();
    const createRouter = vi.fn(() => createMemoryRouter([]));
    const dependencies = createDependencies({
      loadRuntimeConfig: async () => {
        throw new RuntimeConfigError("runtime config contains unknown fields");
      },
      createHttpClient,
      createStartupClient: vi.fn(),
      createRouter,
    });

    render(<Bootstrap dependencies={dependencies} />);

    expect(await screen.findByRole("heading", { name: "Runtime configuration is invalid" })).toBeVisible();
    expect(screen.getByRole("alert")).toHaveTextContent("Check the deployment-time browser configuration.");
    expect(screen.queryByRole("heading", { name: "Monitra is ready" })).not.toBeInTheDocument();
    expect(createHttpClient).not.toHaveBeenCalled();
    expect(createRouter).not.toHaveBeenCalled();
  });

  test("shows a backend failure when the handshake times out", async () => {
    const createRouter = vi.fn(() => createMemoryRouter([]));
    const executeFetch = vi.fn(
      (request: Request) =>
        new Promise<Response>((_resolve, reject) => {
          request.signal.addEventListener("abort", () => reject(request.signal.reason), {
            once: true,
          });
        }),
    );
    const dependencies = createDependencies({
      createHttpClient: () => new BrowserHttpClient(executeFetch, 5),
      createStartupClient: (httpClient) => ({
        getStartupHandshake: async () => {
          await httpClient.execute(new Request("https://monitra.test/api/v1/startup-handshake"));
          return { data: compatibleHandshake };
        },
      }),
      createRouter,
    });

    render(<Bootstrap dependencies={dependencies} />);

    expect(await screen.findByRole("heading", { name: "Backend is unavailable" })).toBeVisible();
    expect(screen.getByRole("alert")).toHaveTextContent("The startup handshake could not be completed.");
    expect(screen.queryByRole("heading", { name: "Monitra is ready" })).not.toBeInTheDocument();
    expect(executeFetch.mock.calls[0]?.[0].signal.aborted).toBe(true);
    expect(createRouter).not.toHaveBeenCalled();
  });

  test("shows a backend failure for an unusable handshake response", async () => {
    const createRouter = vi.fn(() => createMemoryRouter([]));
    const dependencies = createDependencies({
      createStartupClient: () => ({
        getStartupHandshake: async () => ({
          data: { ...compatibleHandshake, request_id: "" },
        }),
      }),
      createRouter,
    });

    render(<Bootstrap dependencies={dependencies} />);

    expect(await screen.findByRole("heading", { name: "Backend is unavailable" })).toBeVisible();
    expect(createRouter).not.toHaveBeenCalled();
  });

  test("rejects an incompatible API major before creating the Router", async () => {
    const createRouter = vi.fn(() => createMemoryRouter([]));
    const dependencies = createDependencies({
      createStartupClient: () => ({
        getStartupHandshake: async () => ({
          data: {
            ...compatibleHandshake,
            data: { ...compatibleHandshake.data, api_major: 2 },
          },
        }),
      }),
      createRouter,
    });

    render(<Bootstrap dependencies={dependencies} />);

    expect(await screen.findByRole("heading", { name: "API version is incompatible" })).toBeVisible();
    expect(screen.getByRole("alert")).toHaveTextContent("Expected API major 1 but received 2.");
    expect(screen.queryByRole("heading", { name: "Monitra is ready" })).not.toBeInTheDocument();
    expect(createRouter).not.toHaveBeenCalled();
  });

  test("rejects an incompatible Release Identity before creating the Router", async () => {
    const createRouter = vi.fn(() => createMemoryRouter([]));
    const dependencies = createDependencies({
      loadRuntimeConfig: async () => ({
        expectedApiMajor: 1,
        expectedReleaseIdentity: "2026.08.06-web",
      }),
      createRouter,
    });

    render(<Bootstrap dependencies={dependencies} />);

    expect(await screen.findByRole("heading", { name: "Release Identity is incompatible" })).toBeVisible();
    expect(screen.getByRole("alert")).toHaveTextContent(
      "Expected Release Identity 2026.08.06-web but received 2026.08.06-test.",
    );
    expect(screen.queryByRole("heading", { name: "Monitra is ready" })).not.toBeInTheDocument();
    expect(createRouter).not.toHaveBeenCalled();
  });
});
