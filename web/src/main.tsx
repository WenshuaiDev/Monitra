import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { createStartupClient } from "./api/generated-client";
import { Bootstrap, type BootstrapDependencies } from "./bootstrap/bootstrap";
import { loadRuntimeConfig } from "./bootstrap/runtime-config";
import { BrowserHttpClient } from "./foundation/http-client";
import { createApplicationRouter } from "./router/application-router";
import "./styles.css";

const handshakeTimeoutMilliseconds = 5_000;
const browserFetch = window.fetch.bind(window);

const dependencies: BootstrapDependencies = {
  loadRuntimeConfig: () => loadRuntimeConfig(browserFetch),
  createHttpClient: () => new BrowserHttpClient(browserFetch, handshakeTimeoutMilliseconds),
  createStartupClient: (httpClient) => createStartupClient(window.location.origin, httpClient),
  createRouter: createApplicationRouter,
};

const rootElement = document.getElementById("root");
if (rootElement === null) {
  throw new Error("browser root element is missing");
}

createRoot(rootElement).render(
  <StrictMode>
    <Bootstrap dependencies={dependencies} />
  </StrictMode>,
);
