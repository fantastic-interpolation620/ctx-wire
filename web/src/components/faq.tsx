import { RiArrowDownSLine } from "@remixicon/react";
import { motion, useReducedMotion } from "motion/react";
import { fadeUp, staggerContainer } from "../lib/variants";

// Engineering-honest answers to the questions a skeptical dev actually asks.
// Claims here must stay true to the implementation: pass-through by default,
// fail-closed scrubbing, local-only data, conformance-tested filters.
const ITEMS = [
  {
    q: "What if a filter doesn't recognize the output?",
    a: "It passes through untouched, up to a generous ceiling. A filter only compresses output it positively recognizes, and unknown output reaches the agent unmodified unless a single dump runs past roughly 64 KB; then the head and tail are kept, the omitted middle is marked explicitly, and the full output stays on disk. The ceiling scales with the truncate dial in config, and setting it to none disables even that.",
  },
  {
    q: "Can it corrupt something my agent parses?",
    a: "That risk is engineered against. Command substitutions like $(cat config.json) are never rewritten, streaming and interactive commands are auto-detected and bypassed, failing commands keep their output, and 330+ conformance tests pin every filter's behavior on each release.",
  },
  {
    q: "What about secrets?",
    a: "Scrubbing is fail-closed: tokens, keys, and credentials are redacted from both the command line and the output before anything reaches the agent or disk. When the scrubber can't be sure, it withholds rather than leaks.",
  },
  {
    q: "What's the overhead?",
    a: "A single local Go binary on the hot path: no daemon, no network call, filtering runs in-process in milliseconds. The command's own runtime dominates. Dev servers, watchers, and interactive commands are bypassed entirely.",
  },
  {
    q: "Does anything leave my machine?",
    a: "Command output, logs, and gain history stay local. Optional telemetry sends aggregate counters only (command counts, bytes saved), never commands or output, and a single flag turns it off.",
  },
  {
    q: "Which agents does it cover?",
    a: "Hooks or plugins for Claude Code, Codex, Cursor, Gemini, Copilot, OpenCode, Pi, and Hermes. Rules plus PATH shims for steering-only agents like Cline and Windsurf. An MCP server for VS Code and Visual Studio, and mcp-wrap for any MCP server's tool output.",
  },
];

const eyebrow =
  "m-0 inline-flex items-center gap-2.5 font-mono text-xs font-medium uppercase tracking-eyebrow text-green";

export function Faq() {
  const reduce = useReducedMotion();
  const v = (variant: typeof fadeUp) => (reduce ? undefined : variant);

  return (
    <motion.section
      aria-label="Frequently asked questions"
      variants={v(staggerContainer)}
      initial={reduce ? undefined : "hidden"}
      whileInView="visible"
      viewport={{ once: true, amount: 0.15 }}
      className="globe-card-bg w-full max-w-stage rounded-section p-cardpad"
    >
      <motion.p variants={v(fadeUp)} className={eyebrow}>
        <span className="size-1.5 rounded-full bg-green shadow-dot" />
        faq
      </motion.p>

      <motion.div variants={v(staggerContainer)} className="mt-4">
        {ITEMS.map((item) => (
          <motion.div
            key={item.q}
            variants={v(fadeUp)}
            className="border-t border-line-soft first:border-t-0"
          >
            <details className="group">
              <summary className="flex cursor-pointer list-none items-center gap-3 py-4 font-mono text-cap font-bold text-head [&::-webkit-details-marker]:hidden">
                <RiArrowDownSLine
                  size={16}
                  className="shrink-0 text-green transition-transform duration-150 ease-out group-open:rotate-180"
                />
                {item.q}
              </summary>
              <p className="m-0 pb-4 pl-7 font-mono text-2xs leading-relaxed text-label">
                {item.a}
              </p>
            </details>
          </motion.div>
        ))}
      </motion.div>
    </motion.section>
  );
}
