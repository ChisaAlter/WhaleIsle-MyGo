// Build-time generation of the actual registered schema and model-facing guide.
// This does not mount React, render a program, or contact a resource service.
import { writeFileSync } from "node:fs";
import { compatibleLibrary } from "../src/library";

writeFileSync("dist/library-spec.json", JSON.stringify(compatibleLibrary.toSpec(), null, 2) + "\n");
writeFileSync("dist/library-guide.txt", compatibleLibrary.prompt({
  preamble: "WhaleIsle-MyGo independently authored OpenUI components. Use only the registered component and function signatures. Root is Col. Components outside this list are unsupported. Resources use the application's typed resource bridge. CopyToClipboard, Code copy, Link and OpenUrl use owned Host actions only after actual user activation; they require Host capability and report actual results. Only absolute HTTP(S) URLs without credentials are supported. HtmlView runs local HTML/CSS/inline JavaScript in a separate opaque sandbox; it has no resource bridge, CDN, native actions, external navigation or media. YouTube reserves an inline native player after an actual user click, using a new temporary profile without existing cookies or native bindings. Player creation is not proof of playback or video availability. Business Query/Mutation providers are unavailable.",
  bindings: true,
  builtinFunctions: true,
  toolCalls: false,
  additionalRules: ["Preserve $binding names when continuing an existing interactive answer.", "Use local @Set and @Reset for local changes. @ToAssistant sends through the native host's existing confirmation and session flow.", "Image and Markdown/Text images use the fixed public Host image bridge for absolute HTTP(S) sources. Remote SVG is restricted to the Host-validated static subset; direct data SVG and credentialed/private-network images are unsupported. Resource failures must be shown; do not invent images, map tiles, route coordinates or live business data."],
}) + "\n");
