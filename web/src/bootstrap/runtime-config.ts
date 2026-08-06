export interface RuntimeConfig {
  expectedApiMajor: number;
  expectedReleaseIdentity: string;
}

export class RuntimeConfigError extends Error {
  override readonly name = "RuntimeConfigError";
}

export type ConfigFetcher = (input: string, init: RequestInit) => Promise<Response>;

export async function loadRuntimeConfig(
  fetchConfig: ConfigFetcher = globalThis.fetch,
): Promise<RuntimeConfig> {
  let response: Response;
  try {
    response = await fetchConfig("/runtime-config.json", {
      cache: "no-store",
      headers: { Accept: "application/json" },
    });
  } catch (error) {
    throw new RuntimeConfigError("runtime config could not be loaded", { cause: error });
  }

  if (!response.ok) {
    throw new RuntimeConfigError(`runtime config returned HTTP ${response.status}`);
  }

  let document: unknown;
  try {
    document = await response.json();
  } catch (error) {
    throw new RuntimeConfigError("runtime config is not valid JSON", { cause: error });
  }

  if (
    typeof document !== "object" ||
    document === null ||
    Array.isArray(document) ||
    Object.keys(document).sort().join(",") !==
      "expected_api_major,expected_release_identity" ||
    !("expected_api_major" in document) ||
    typeof document.expected_api_major !== "number" ||
    !Number.isSafeInteger(document.expected_api_major) ||
    document.expected_api_major < 1 ||
    !("expected_release_identity" in document) ||
    typeof document.expected_release_identity !== "string" ||
    document.expected_release_identity.length === 0 ||
    document.expected_release_identity.trim() !== document.expected_release_identity
  ) {
    throw new RuntimeConfigError("runtime config does not match the browser contract");
  }

  return {
    expectedApiMajor: document.expected_api_major,
    expectedReleaseIdentity: document.expected_release_identity,
  };
}
