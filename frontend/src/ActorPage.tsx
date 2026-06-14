import { useEffect, useState } from "react";
import type { ActorPage as ActorPageT } from "./types/actor_page";
import { fetchActorPage } from "./api";

export function ActorPage({ world, actorId }: { world: string; actorId: string }) {
  const [page, setPage] = useState<ActorPageT | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    fetchActorPage(world, actorId).then(setPage).catch((e) => setErr(String(e)));
  }, [world, actorId]);

  if (err) return <main>Could not load this page.</main>;
  if (!page) return <main>Loading…</main>;

  const a = page.actor;
  const groups = a.collected_knowledge_groups ?? [];
  const hasKnowledge = groups.some((g) => (g.items ?? []).length > 0);

  return (
    <main style={{ maxWidth: 680, margin: "2rem auto", fontFamily: "system-ui" }}>
      <h1>{a.perceived_name ?? "Unknown"}</h1>
      {a.perceived_role && <p style={{ color: "#666" }}>{a.perceived_role}</p>}

      <section>
        <h2>Synthesis</h2>
        <p>{a.current_synthesis ?? <em>Nothing synthesized yet.</em>}</p>
      </section>

      {a.last_known_status && (
        <section><h2>Last known</h2><p>{a.last_known_status}</p></section>
      )}

      <section>
        <h2>Collected Knowledge</h2>
        {!hasKnowledge && <p><em>You know nothing about them yet.</em></p>}
        {groups.map((g) => (
          <div key={g.group_key}>
            {g.group_label && <h3>{g.group_label}</h3>}
            <ul>
              {(g.items ?? []).map((it) => (
                <li key={it.perception_id} style={{ marginBottom: "0.5rem" }}>
                  <div>{it.content}</div>
                  <small style={{ color: "#888" }}>
                    {it.epistemic_type}
                    {it.display_label ? ` · ${it.display_label}` : ""}
                    {it.decay && (it.decay as { stale?: boolean }).stale ? " · last known…" : ""}
                  </small>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </section>
    </main>
  );
}
