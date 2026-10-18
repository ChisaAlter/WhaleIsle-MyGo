// Build-time facade over the unmodified official MIT lang-core package.
// The bundler must map @whaleisle/official-lang-core directly to its installed
// dist/index.mjs, and map @openuidev/lang-core to this file to avoid recursion.
// Every official public export is retained; only store creation is adapted.
export * from "@whaleisle/official-lang-core";
export { createPersistentStore as createStore } from "./state-adapter";
