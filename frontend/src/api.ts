import type { ActorPage } from "./types/actor_page";

export async function fetchActorPage(world: string, actorId: string): Promise<ActorPage> {
  const res = await fetch(`/worlds/${world}/compendium/actors/${actorId}/page`);
  if (!res.ok) throw new Error(`actor page ${res.status}`);
  return (await res.json()) as ActorPage;
}
