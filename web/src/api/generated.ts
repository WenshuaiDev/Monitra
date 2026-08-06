export interface paths {
    "/api/v1/startup-handshake": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Read the running release and API compatibility identity */
        get: operations["getStartupHandshake"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
}
export type webhooks = Record<string, never>;
export interface components {
    schemas: {
        RequestID: string;
        StartupHandshakeResponse: {
            /** @enum {string} */
            code: "FOUNDATION_STARTUP_READY";
            /** @enum {string} */
            message: "startup handshake succeeded";
            data: components["schemas"]["StartupHandshakeData"];
            request_id: components["schemas"]["RequestID"];
        };
        StartupHandshakeData: {
            release_identity: string;
            /**
             * Format: int32
             * @enum {integer}
             */
            api_major: 1;
        };
        ErrorResponse: {
            /** @enum {string} */
            code: "FOUNDATION_NOT_FOUND" | "FOUNDATION_METHOD_NOT_ALLOWED";
            /** @enum {string} */
            message: "resource not found" | "method not allowed";
            data: null;
            request_id: components["schemas"]["RequestID"];
        };
    };
    responses: {
        /** @description The requested application resource does not exist. */
        NotFound: {
            headers: {
                "X-Request-ID": components["headers"]["RequestID"];
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["ErrorResponse"];
            };
        };
        /** @description The requested method is not supported by this route. */
        MethodNotAllowed: {
            headers: {
                "X-Request-ID": components["headers"]["RequestID"];
                /** @description Methods supported by this route. */
                Allow: "GET";
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["ErrorResponse"];
            };
        };
    };
    parameters: never;
    requestBodies: never;
    headers: {
        /** @description Cryptographically safe server-generated correlation identity. */
        RequestID: components["schemas"]["RequestID"];
    };
    pathItems: never;
}
export type $defs = Record<string, never>;
export interface operations {
    getStartupHandshake: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The core process is available for a compatible browser bootstrap. */
            200: {
                headers: {
                    "X-Request-ID": components["headers"]["RequestID"];
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["StartupHandshakeResponse"];
                };
            };
            404: components["responses"]["NotFound"];
            405: components["responses"]["MethodNotAllowed"];
        };
    };
}
