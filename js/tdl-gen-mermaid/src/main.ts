#!/usr/bin/env node
import { run } from "@unmango/tdl-plugin";

import { mermaid } from "./index.js";

await run(mermaid);
