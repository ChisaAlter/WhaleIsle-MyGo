// Trusted helper support is opt-in. This is not a component-controlled setting.
import { createContext } from "react";

export const HtmlViewSupportContext = createContext(false);
export const HostActionsContext = createContext(false);
// Trusted source/streaming revision; ordinary SDK state writes do not change it.
export const ProgramRevisionContext = createContext(0);
