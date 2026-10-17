// Build-time generation of the actual registered schema and model-facing guide.
// This does not mount React, render a program, or contact a resource service.
import { writeFileSync } from "node:fs";
import { compatibleLibrary } from "../src/library";

writeFileSync("dist/library-spec.json", JSON.stringify(compatibleLibrary.toSpec(), null, 2) + "\n");
writeFileSync("dist/library-guide.txt", compatibleLibrary.prompt({
  preamble: "WhaleIsle-MyGo independently authored OpenUI components. Use only the registered component and function signatures. Root is Col. Components outside this list are unsupported. Network and native capabilities are available only through the application's typed resource bridge. Clipboard, external navigation, arbitrary HTML/media and business Query/Mutation providers are unavailable.",
  bindings: true,
  builtinFunctions: true,
  toolCalls: false,
  additionalRules: ["Preserve $binding names when continuing an existing interactive answer.", "Use local @Set and @Reset for local changes. @ToAssistant sends through the native host's existing confirmation and session flow.", "Resource failures must be shown; do not invent images, map tiles, route coordinates or live business data."],
}) + "\n");
