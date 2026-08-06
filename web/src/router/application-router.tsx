import { createBrowserRouter } from "react-router-dom";

import type { StartupIdentity } from "../bootstrap/bootstrap";

export function createApplicationRouter(identity: StartupIdentity) {
  return createBrowserRouter([
    {
      path: "*",
      element: <StartupReadyPage identity={identity} />,
    },
  ]);
}

function StartupReadyPage({ identity }: { identity: StartupIdentity }) {
  return (
    <main className="startup-card" aria-labelledby="startup-ready-title">
      <p className="eyebrow">Foundation startup</p>
      <h1 id="startup-ready-title">Monitra is ready</h1>
      <p className="summary">The browser and backend passed the compatibility handshake.</p>
      <dl className="startup-details">
        <div>
          <dt>Release Identity</dt>
          <dd>{identity.releaseIdentity}</dd>
        </div>
        <div>
          <dt>API major</dt>
          <dd>{identity.apiMajor}</dd>
        </div>
        <div>
          <dt>Request ID</dt>
          <dd>{identity.requestID}</dd>
        </div>
      </dl>
    </main>
  );
}
