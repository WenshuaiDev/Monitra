import { expect, test, type Page, type TestInfo } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { rename, writeFile } from "node:fs/promises";

const releaseIdentity = process.env.MONITRA_E2E_RELEASE_IDENTITY ?? "ticket06-test";

test.beforeEach(async () => {
  await setRuntimeConfig({
    expected_api_major: 1,
    expected_release_identity: releaseIdentity,
  });
  runCompose(["up", "--detach", "--wait", "--wait-timeout", "90", "core"], "inherit");
});

test.afterEach(async ({}, testInfo) => {
  await testInfo.attach("compose-state", {
    body: Buffer.from(runCompose(["ps", "--all"], "capture")),
    contentType: "text/plain",
  });
  await testInfo.attach("compose-logs", {
    body: Buffer.from(
      runCompose(
        ["logs", "--timestamps", "--tail", "100", "runtime-config", "core", "caddy"],
        "capture",
      ),
    ),
    contentType: "text/plain",
  });
});

test("production browser shows the compatible startup identity", async ({ page }, testInfo) => {
  await page.goto("/");

  const shell = page.locator("main.startup-card");
  await expect(shell.getByRole("heading", { name: "Monitra is ready" })).toBeVisible();
  await expect(shell.getByText(releaseIdentity, { exact: true })).toBeVisible();
  await expect(
    shell.locator(".startup-details div").filter({ hasText: "API major" }).locator("dd"),
  ).toHaveText("1");
  await expect(
    shell.locator(".startup-details div").filter({ hasText: "Request ID" }).locator("dd"),
  ).not.toHaveText("");

  await attachBrowserState(page, testInfo);
});

test("invalid runtime config shows configuration failure without the normal shell", async ({
  page,
}, testInfo) => {
  await setRuntimeConfigDocument("not valid JSON\n");
  await page.goto("/");

  await assertStartupFailure(
    page,
    "Runtime configuration is invalid",
    "Check the deployment-time browser configuration.",
  );
  await attachBrowserState(page, testInfo);
});

test("unavailable core shows backend failure without the normal shell", async ({ page }, testInfo) => {
  await setRuntimeConfig({
    expected_api_major: 1,
    expected_release_identity: releaseIdentity,
  });
  runCompose(["stop", "core"], "inherit");
  await page.goto("/");

  await assertStartupFailure(
    page,
    "Backend is unavailable",
    "The startup handshake could not be completed.",
  );
  await attachBrowserState(page, testInfo);
});

test("incompatible API major shows the exact compatibility failure", async ({ page }, testInfo) => {
  await setRuntimeConfig({
    expected_api_major: 2,
    expected_release_identity: releaseIdentity,
  });
  await page.goto("/");

  await assertStartupFailure(
    page,
    "API version is incompatible",
    "Expected API major 2 but received 1.",
  );
  await attachBrowserState(page, testInfo);
});

test("incompatible release identity shows the exact compatibility failure", async ({
  page,
}, testInfo) => {
  await setRuntimeConfig({
    expected_api_major: 1,
    expected_release_identity: "incompatible-release",
  });
  await page.goto("/");

  await assertStartupFailure(
    page,
    "Release Identity is incompatible",
    `Expected Release Identity incompatible-release but received ${releaseIdentity}.`,
  );
  await attachBrowserState(page, testInfo);
});

async function assertStartupFailure(page: Page, heading: string, message: string) {
  const failure = page.locator('main[role="alert"]');
  await expect(failure.getByRole("heading", { name: heading, exact: true })).toBeVisible();
  await expect(failure.getByText(message, { exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Monitra is ready", exact: true })).toHaveCount(0);
  await expect(page.locator("main.startup-card")).toHaveCount(0);
}

async function attachBrowserState(page: Page, testInfo: TestInfo) {
  await testInfo.attach("browser-state", {
    body: await page.screenshot({ fullPage: true }),
    contentType: "image/png",
  });
}

async function setRuntimeConfigDocument(document: string) {
  const runtimeConfigFile = process.env.MONITRA_E2E_RUNTIME_CONFIG_FILE;
  if (runtimeConfigFile === undefined || runtimeConfigFile.length === 0) {
    throw new Error("MONITRA_E2E_RUNTIME_CONFIG_FILE is required");
  }
  const temporaryFile = `${runtimeConfigFile}.${process.pid}.tmp`;
  await writeFile(temporaryFile, document, { encoding: "utf8", mode: 0o600 });
  await rename(temporaryFile, runtimeConfigFile);
}

async function setRuntimeConfig(configuration: {
  expected_api_major: number;
  expected_release_identity: string;
}) {
  await setRuntimeConfigDocument(`${JSON.stringify(configuration)}\n`);
}

function runCompose(arguments_: string[], output: "inherit"): void;
function runCompose(arguments_: string[], output: "capture"): string;
function runCompose(arguments_: string[], output: "inherit" | "capture") {
  if (output === "inherit") {
    execFileSync("docker", ["compose", ...arguments_], { stdio: "inherit" });
    return;
  }
  return execFileSync("docker", ["compose", ...arguments_], { encoding: "utf8" });
}
