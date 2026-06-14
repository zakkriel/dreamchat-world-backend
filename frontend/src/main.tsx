import React from "react";
import { createRoot } from "react-dom/client";
import { ActorPage } from "./ActorPage";

const WORLD = "11111111-1111-1111-1111-111111111111";
const MARA = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb";

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <ActorPage world={WORLD} actorId={MARA} />
  </React.StrictMode>
);
