// HttpClient is the single frontend application network execution seam.
export interface HttpClient {
  execute(request: Request): Promise<Response>;
}

export type BrowserFetch = (request: Request) => Promise<Response>;

export class BrowserHttpClient implements HttpClient {
  constructor(
    private readonly executeFetch: BrowserFetch,
    private readonly timeoutMilliseconds: number,
  ) {}

  execute(request: Request): Promise<Response> {
    const timeout = AbortSignal.timeout(this.timeoutMilliseconds);
    const signal = AbortSignal.any([request.signal, timeout]);
    return this.executeFetch(new Request(request, { signal }));
  }
}
