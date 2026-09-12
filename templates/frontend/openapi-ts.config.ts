import { defineConfig } from "@hey-api/openapi-ts";

// Generates lib/api/ from the backend's committed contract. lib/api/ is
// gitignored and rebuilt on every install (see "postinstall"), so the only
// hand-maintained piece of the client is lib/api-client.ts.
export default defineConfig({
  input: "../docs/openapi.yaml",
  output: "lib/api",
  plugins: [
    { name: "@hey-api/client-fetch", runtimeConfigPath: "./lib/api-client" },
    {
      name: "@hey-api/sdk",
      operations: {
        strategy: "single",
        containerName: { name: "api", casing: "preserve" },
        methods: "static",
        // Group by the operation's OpenAPI tag (the httpx.Group tag) so an
        // operation id like "get-current-user" under "Users" becomes
        // api.users.getCurrentUser().
        nesting: (op) => [
          op.tags?.[0]?.toLowerCase() ?? "default",
          op.operationId ?? op.id,
        ],
      },
    },
  ],
});
