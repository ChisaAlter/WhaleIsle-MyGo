import { createLibrary, defineAction } from "@openuidev/react-lang";
import { z } from "zod/v4";
import { layoutComponents } from "./components/layout";
import { contentComponents } from "./components/content";
import { controlComponents } from "./components/controls";
import { compatibleFunctions } from "./functions";
import { extraFunctions } from "./functions-extra";
import { imageComponents } from "./components/images";
import { interactiveComponents } from "./components/interactive";
import { chartComponents } from "./components/charts";
import { svgComponents } from "./components/svg";
import { mapComponents } from "./components/map";
import { extraControlComponents } from "./components/control-extras";
import { htmlComponents } from "./components/html-view";

// Native steps are intercepted by our gesture-bound controls, never forwarded
// automatically from an SDK event or generated page to a browser API.
const copyToClipboard = defineAction({ name: "CopyToClipboard", params: z.object({ text: z.string() }), description: "Copies text through the owned Host after a real user activation and reports its actual result." });

export const compatibleLibrary = createLibrary({
  id: "whaleisle-independent-col-v1",
  root: "Col",
  components: [...layoutComponents, ...contentComponents, ...controlComponents, ...imageComponents,
    ...interactiveComponents, ...chartComponents, ...svgComponents, ...mapComponents, ...extraControlComponents, ...htmlComponents],
  functions: [...compatibleFunctions, ...extraFunctions],
  actions: [copyToClipboard],
});
