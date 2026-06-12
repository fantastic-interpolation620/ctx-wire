import { motion, useReducedMotion } from "motion/react";
import { formatInt, formatTokens } from "../format";
import { fadeUp, staggerContainer } from "../lib/variants";
import type { AgentStats, ImpactStats } from "../types";
import { AgentLogo, agentLabel } from "./agent-logos";

const tokensOf = (a: AgentStats) => Number(a.tokens_saved || 0);
const EASE_OUT = [0.23, 1, 0.32, 1] as const;

function Row({
  agent,
  pct,
  rank,
  lead,
  reduce,
}: {
  agent: AgentStats;
  pct: number; // share of the leader, drives the row fill
  rank: number;
  lead: boolean;
  reduce: boolean;
}) {
  return (
    <motion.div
      variants={reduce ? undefined : fadeUp}
      className="relative overflow-hidden rounded-xl bg-white/[0.02] ring-1 ring-inset ring-line-soft"
    >
      {/* share fill , the bar IS the row */}
      <motion.div
        aria-hidden="true"
        className={`absolute inset-y-0 left-0 ${
          lead ? "bg-green/[0.14]" : "bg-cyan/[0.08]"
        }`}
        style={{ width: `${pct}%`, transformOrigin: "left" }}
        initial={reduce ? false : { scaleX: 0 }}
        animate={{ scaleX: 1 }}
        transition={
          reduce ? { duration: 0 } : { duration: 0.85, ease: EASE_OUT }
        }
      />
      <div className="relative flex items-center gap-3 px-4 py-3 sm:gap-4">
        <span className="w-5 shrink-0 text-right font-mono text-xs font-bold tabular-nums text-label">
          {rank}
        </span>
        <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-white ring-1 ring-inset ring-black/10">
          <AgentLogo name={agent.agent} size={28} />
        </span>
        <div className="min-w-0">
          <div className="truncate font-mono text-cap font-bold text-head">
            {agentLabel(agent.agent)}
          </div>
          <div className="truncate font-mono text-2xs text-label">
            {formatInt(agent.runs)} commands
          </div>
        </div>
        <div className="ml-auto shrink-0 text-right">
          <span className="font-mono text-base font-bold tabular-nums text-head sm:text-lg">
            {formatTokens(tokensOf(agent))}
          </span>
          <span className="ml-1 font-mono text-2xs font-medium text-label">
            tokens
          </span>
        </div>
      </div>
    </motion.div>
  );
}

export function SavedByAgent({ stats }: { stats: ImpactStats }) {
  const reduce = useReducedMotion();
  const agents = (stats.agents ?? [])
    .filter((a) => tokensOf(a) > 0)
    .sort((a, b) => tokensOf(b) - tokensOf(a));

  if (agents.length === 0) {
    return null;
  }

  const max = tokensOf(agents[0]) || 1;
  const total = agents.reduce((sum, a) => sum + tokensOf(a), 0);

  return (
    <motion.section
      aria-label="Token savings by agent"
      variants={reduce ? undefined : staggerContainer}
      initial={reduce ? undefined : "hidden"}
      whileInView="visible"
      viewport={{ once: true, amount: 0.2 }}
      className="globe-card-bg w-full max-w-stage rounded-section p-cardpad"
    >
      <motion.p
        variants={reduce ? undefined : fadeUp}
        className="m-0 inline-flex items-center gap-2.5 font-mono text-xs font-medium uppercase tracking-eyebrow text-green"
      >
        <span className="size-1.5 rounded-full bg-green shadow-dot" />
        saved by agent
      </motion.p>
      <motion.p
        variants={reduce ? undefined : fadeUp}
        className="m-0 mt-3 mb-5 font-mono text-cap text-label"
      >
        Token savings attributed to each coding agent, live from telemetry.{" "}
        <span className="text-dim">
          {formatInt(agents.length)} agents · {formatTokens(total)} saved.
        </span>
      </motion.p>

      <motion.div
        variants={reduce ? undefined : staggerContainer}
        className="flex flex-col gap-2"
      >
        {agents.map((agent, index) => (
          <Row
            key={agent.agent}
            agent={agent}
            rank={index + 1}
            pct={(tokensOf(agent) / max) * 100}
            lead={index === 0}
            reduce={Boolean(reduce)}
          />
        ))}
      </motion.div>
    </motion.section>
  );
}
