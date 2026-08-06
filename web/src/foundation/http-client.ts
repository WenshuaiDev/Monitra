// HttpClient is the single frontend application network execution seam.
export interface HttpClient {
  execute(request: Request): Promise<Response>;
}
