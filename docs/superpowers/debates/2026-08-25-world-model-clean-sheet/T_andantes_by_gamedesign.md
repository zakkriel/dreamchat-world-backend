# Stress test — Los Andantes vs world_model/1 (systems/simulation hat)

## 1. The encoding (detailed tier, year 641)

```jsonc
{
  "world_model": "1",
  "world": { "name": "Los Andantes",
    "premise": "No hay continentes. Nueve criaturas vivas de 44-90 km caminan un océano somero en rutas fijas, y las ciudades están construidas sobre sus lomos. Un Andante enfermo es una ciudad con fecha de vencimiento.",
    "mood": ["clínico", "administrativo", "catástrofe lenta"] },

  "vocabulary": {
    "media": [
      { "name": "aire de dorso", "descriptor": "aire abierto a 60-200 m sobre la línea de agua",
        "affords": [ { "to": "sight", "degree": "full" }, { "to": "sound", "degree": "full" } ] },
      { "name": "mar somero", "descriptor": "30-60 m de agua sobre fondo firme, sin costas",
        "resists": [ { "to": "a pie", "degree": "total" }, { "to": "habitación", "degree": "total" } ],
        "affords": [ { "to": "en barcaza", "degree": "partial" } ] }
    ],
    "movements": [
      { "name": "a pie", "pace_class": "steady" },
      { "name": "en barcaza", "pace_class": "slow",
        "note": "único tránsito fuera de Convergencia; supervivencia inferior al 30%" }
    ],
    "channels": [
      { "name": "sight" }, { "name": "sound" },
      { "name": "el pulso profundo", "descriptor": "latido audible con trompetilla apoyada en el dorso",
        "note": "// BREAK: no es un canal de sucesos, es un INDICADOR de un estado oculto continuo" },
      { "name": "la Campana", "descriptor": "cifra de marcha diaria, pública, suena al alba" }
    ],
    "conditions": [],
    "substances": [ { "name": "carne de Andante", "note": "prohibición no escrita, cumplimiento universal" } ]
  },

  "law": [
    { "name": "indirigibilidad", "stated": "No existe método para alterar la ruta ni la velocidad de un Andante.", "governs": "movement" },
    { "name": "hundimiento", "stated": "Un Andante muerto se hunde en 5-7 días. No hay excepciones registradas.", "governs": "integrity" },
    { "name": "irreversibilidad", "stated": "Ningún tratamiento revierte la enfermedad de un Andante.", "governs": "integrity" },
    { "name": "ausencia de tierra", "stated": "No hay superficie habitable fuera de un Andante.", "governs": "medium" }
  ],

  "places": [
    // BREAK 1 — Tercera Hembra no es un contenedor: camina 2,1 km/día en una ruta cerrada de 19 años,
    // tiene límite de carga, enferma y puede morir. Aquí sólo puede ser un `parent` inerte.
    { "name": "Tercera Hembra", "parent": null, "extent_class": "vast", "sort": "Andante",
      "medium": "aire de dorso", "tension": "tense" },
    { "name": "Primera", "parent": null, "extent_class": "vast", "sort": "Andante", "medium": "aire de dorso", "tension": "calm" },
    { "name": "Quinta", "parent": null, "extent_class": "vast", "sort": "Andante", "medium": "aire de dorso", "tension": "normal" },
    { "name": "Octavo", "parent": null, "extent_class": "vast", "sort": "Andante muerto y hundido en 611",
      "medium": "mar somero", "tension": "none" },   // BREAK 2 — un lugar que dejó de existir
    { "name": "Ossa", "parent": "Tercera Hembra", "extent_class": "large", "sort": "ciudad", "medium": "aire de dorso", "tension": "tense" },
    { "name": "Belna", "parent": "Tercera Hembra", "extent_class": "medium", "sort": "ciudad", "medium": "aire de dorso", "tension": "normal" },
    { "name": "Alto Omóplato", "parent": "Ossa", "extent_class": "medium", "sort": "distrito sobre la escápula izquierda", "medium": "aire de dorso", "tension": "normal" },
    { "name": "Cuenca", "parent": "Ossa", "extent_class": "medium", "sort": "distrito en la depresión lumbar",
      "medium": "aire de dorso", "tension": "tense" },
    { "name": "Los Espiráculos", "parent": "Ossa", "extent_class": "small", "sort": "veda de 400 m",
      "medium": "aire de dorso", "tension": "frantic",
      "ambient_demand": [ { "requires": "licencia que nadie da", "absent_effect": "prohibición", "onset": "inmediato" } ] },
    { "name": "Cola Baja", "parent": "Ossa", "extent_class": "medium", "sort": "distrito del tercio posterior",
      "medium": "aire de dorso", "tension": "normal" }
  ],

  "ways": [
    { "name": "camino dorsal", "connects": ["Ossa", "Belna"], "state": "open", "affords": ["a pie"], "obstructs": [] },
    // BREAK 3 — la Convergencia 645 existe 31 días y es calculable con siglos de anticipación.
    // `state` es estático; no hay forma de decir "existe sólo durante".
    { "name": "Convergencia 645", "connects": ["Tercera Hembra", "Primera"], "state": "shut",
      "affords": ["a pie"], "obstructs": [] },
    // BREAK 4 — 70% de bajas no es `obstructs`: no impide, mata.
    { "name": "travesía de barcaza", "connects": ["Tercera Hembra", "Primera"], "state": "open",
      "affords": ["en barcaza"], "obstructs": ["a pie"] }
  ],

  "things": [
    { "name": "trompetilla de auscultación", "bulk_class": "slight", "integrity": "sound",
      "where": { "in_place": "Alto Omóplato" } },   // BREAK 6 — único instrumento diagnóstico: confiere capacidad, y eso no se puede decir
    { "name": "Registro de Peso", "bulk_class": "moderate", "integrity": "sound", "where": { "in_place": "Alto Omóplato" } },
    { "name": "parte mensual", "bulk_class": "negligible", "integrity": "sound", "where": { "in_place": "Alto Omóplato" } },
    { "name": "archivo del Colegio", "bulk_class": "moderate", "integrity": "sound", "where": { "in_place": "Alto Omóplato" } },
    { "name": "expediente Octavo", "bulk_class": "slight", "integrity": "worn", "where": { "in_place": "Alto Omóplato" } },
    { "name": "Tablas de Convergencia", "bulk_class": "slight", "integrity": "sound", "where": { "in_place": "Alto Omóplato" } },
    { "name": "registro paralelo de Illa", "bulk_class": "negligible", "integrity": "sound", "where": { "carried_by": "Illa" } },
    { "name": "Campana de Marcha", "bulk_class": "immense", "integrity": "sound", "where": { "in_place": "Alto Omóplato" } },
    { "name": "barcazas", "bulk_class": "immense", "integrity": "worn", "where": { "in_place": "Cola Baja" } }
  ],
  // BREAK 5 — seis de estos nueve objetos son REGISTROS: su contenido es una afirmación que puede
  // ser falsa, se consulta por un procedimiento (tres días de demora; bajo llave; clasificado) y
  // su autoridad viene de una firma. Aquí son bultos con integridad.

  "stocks": [
    { "name": "capacidad de acogida de Primera", "held_in": "Primera", "abundance": "thin",
      "drawn_by": "cada derecho de paso vendido", "replenished_by": null },
    { "name": "barcazas operativas", "held_in": "Cola Baja", "abundance": "thin",
      "drawn_by": "cada travesía", "replenished_by": null }
  ],

  "processes": [
    // BREAK 2b — `acts_on` apunta a un lugar que no tiene integridad. El proceso no puede morder.
    { "name": "enfermedad sistémica", "acts_on": "Tercera Hembra", "direction": "degrade",
      "rate_class": "slow", "terminus": "muerte y hundimiento en 5-7 días" }
  ],

  "cycles": [
    { "name": "marcha diaria", "period_class": "short", "starts_in_phase": "alba",
      "phases": [ { "name": "alba", "changes": [] }, { "name": "jornada", "changes": [] } ] },
    // BREAK 3b — la ruta migratoria es un ciclo de 19 años sin fases significativas, y la
    // Convergencia es la COINCIDENCIA de dos ciclos. No hay forma de expresar una coincidencia.
    { "name": "ruta de Tercera Hembra", "period_class": "generational", "starts_in_phase": "tránsito",
      "phases": [ { "name": "tránsito", "changes": [] } ] }
  ],

  "accumulators": [
    // BREAK 7 — un solo umbral. La regla dura 2 tiene CUATRO peldaños ordenados:
    // velocidad ↓ → cojera → lesión articular → enfermedad sistémica.
    // Y `raised_by` sólo acepta sucesos: la carga real es la SUMA de lo construido por sector.
    { "name": "carga sobre Tercera Hembra", "stated": "Cuánto peso lleva encima el Andante.",
      "starts_at": "high", "raised_by": ["cada obra licenciada en Cuenca", "cada obra sin licencia"],
      "threshold": { "class": "high", "then": "El Andante enferma." } },
    { "name": "discordancia entre partes", "stated": "Cuánto se separa el parte de Belna del de Ossa.",
      "starts_at": "moderate", "raised_by": ["cada parte mensual firmado por Renn"],
      "threshold": { "class": "high", "then": "El Colegio de Ossa pierde autoridad diagnóstica." } }
  ],

  "traces": [
    { "of": "sobrecarga", "leaves": "una cifra de Campana más baja cada mañana", "ages": "no se borra" },
    { "of": "una obra nueva", "leaves": "un asiento en el Registro de Peso con firma", "ages": "no se borra" }
  ],

  "propagation": [
    { "of": "un parte mensual publicado", "spreads": "por toda la ciudad en un día" },
    { "of": "una lectura de auscultación", "spreads": "no sale del Colegio salvo que alguien la lleve" },
    { "of": "la cifra de la Campana", "spreads": "a todo el que esté despierto al alba" }
  ],

  "collectives": [
    { "name": "Colegio de Auscultadores", "legibility": "marked", "descriptor": "gente con trompetilla de bronce",
      "interest": "que la autoridad sobre el estado del Andante no sea verificable desde afuera", "speaks_through": "Del Vas" },
    { "name": "Gremio de Peso", "legibility": "marked", "descriptor": "registradores con libro",
      "interest": "recaudar por licencia y que no se audite lo licenciado", "speaks_through": "Registrador Onn" },
    { "name": "los Convergentes", "legibility": "marked", "descriptor": "gente que sólo trabaja un año de cada diecinueve",
      "interest": "que la Convergencia 645 se cumpla y sea suya", "speaks_through": "Bara Quel" },
    { "name": "la Flota Muerta", "legibility": "marked", "descriptor": "bargueros con casco doble",
      "interest": "conservar el monopolio del tránsito de emergencia", "speaks_through": "Sento" },
    { "name": "Consejo de Ossa", "legibility": "marked", "descriptor": "gobierno civil",
      "interest": "evitar el pánico", "speaks_through": "Uma Ret" }
  ],

  "people": [
    { "name": "Del Vas", "seen_as": "una mujer de 58 años con la trompetilla colgada al cuello",
      "role": "Auscultadora Mayor de Ossa: firma el parte mensual",   // BREAK 8 — el cargo, no la persona, es lo que da autoridad; y se retira en 643
      "belongs_to": ["Colegio de Auscultadores"], "starts_in": "Alto Omóplato",
      "capability": { "moves_by": ["a pie"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "el pulso profundo": "acute" },
      "disposition": [ { "trait": "administrativa", "strength": "defining", "manner": "responde con el procedimiento, no con el dato" } ],
      "doing": "cerrando el archivo con llave antes de la ronda de la mañana",
      "pursuing": [ { "horizon": "long_standing", "toward": "retirarse en 643 sin un colapso en su hoja", "progress": "late", "step": "nueve partes consecutivos sin hallazgos" } ],
      "obligation": [ { "owed_to": "Colegio de Auscultadores", "stated": "no publica una lectura antes que el Colegio" } ],
      "regard": [ { "toward": "Illa", "stance": "la vigila desde que repitió una auscultación sin pedir permiso", "since": "la-arritmia-del-dia-4" } ],
      "hiding": "Detectó la arritmia hace nueve meses y no la consignó." },

    { "name": "Illa", "seen_as": "una aprendiza de 19 años que anota más de lo que le piden",
      "role": "aprendiza de primer año", "belongs_to": ["Colegio de Auscultadores"], "starts_in": "Alto Omóplato",
      "capability": { "moves_by": ["a pie"], "carry_class": "slight" },
      "senses": { "sight": "normal", "el pulso profundo": "normal" },
      "disposition": [ { "trait": "escrupulosa", "strength": "strong", "manner": "repite la medición en vez de discutirla" } ],
      "doing": "copiando la lectura del día 4 en su cuaderno personal",
      "pursuing": [ { "horizon": "long_standing", "toward": "titularse", "progress": "early", "step": "no dar motivo para que la expulsen" } ],
      "hiding": "Confirmó tu lectura, no la declaró, y lleva un registro paralelo." },

    { "name": "Registrador Onn", "seen_as": "un hombre de 44 años con las manos manchadas de tinta",
      "role": "Gremio de Peso: licencia toda obra", "belongs_to": ["Gremio de Peso"], "starts_in": "Alto Omóplato",
      "capability": { "moves_by": ["a pie"], "carry_class": "moderate" },
      "senses": { "sight": "normal" },
      "disposition": [ { "trait": "servicial", "strength": "strong", "manner": "encuentra la manera de firmar" } ],
      "doing": "adelantando el traslado a Quinta antes de que se revise Cuenca",
      "hiding": "Licenció 14 obras sobre el límite en Cuenca entre 637 y 640." },

    { "name": "Bara Quel", "seen_as": "un hombre de 36 años con ropa de otro Andante",
      "role": "Convergente", "belongs_to": ["los Convergentes"], "starts_in": "Alto Omóplato",
      "capability": { "moves_by": ["a pie"], "carry_class": "moderate" }, "senses": { "sight": "normal" },
      "disposition": [ { "trait": "expansivo", "strength": "strong", "manner": "promete el año 645 como si fuera suyo" } ],
      "doing": "vendiendo un derecho de paso más",
      "hiding": "Vendió 3.000 derechos y Primera admite 12.000 en total, incluida Belna." },

    { "name": "Sento", "seen_as": "un barguero de 40 años que no habla de las travesías",
      "role": "Flota Muerta", "belongs_to": ["la Flota Muerta"], "starts_in": "Cola Baja",
      "capability": { "moves_by": ["a pie", "en barcaza"], "carry_class": "strong" }, "senses": { "sight": "normal" },
      "disposition": [ { "trait": "fatalista", "strength": "defining", "manner": "da la cifra de supervivencia sin suavizarla" } ],
      "doing": "revisando un casco que ya cruzó dos veces",
      "hiding": "Fue el único superviviente de su barcaza en dos de tres cruces." },

    { "name": "Vieja Marda", "seen_as": "una mujer de 81 años sentada donde da el sol",
      "role": "superviviente de Ruma", "belongs_to": [], "starts_in": "Cola Baja",
      "capability": { "moves_by": ["a pie"], "carry_class": "slight" }, "senses": { "sight": "faint", "sound": "normal" },
      "disposition": [ { "trait": "callada", "strength": "defining", "manner": "contesta una pregunta de cada cinco" } ],
      "doing": "mirando el agua por el flanco",
      "hiding": "Tenía 51 años en Ruma y sabe qué pasó los seis días." },

    { "name": "Consejera Uma Ret", "seen_as": "una mujer de 61 años con una carpeta que no abre",
      "role": "Consejo de Ossa", "belongs_to": ["Consejo de Ossa"], "starts_in": "Alto Omóplato",
      "capability": { "moves_by": ["a pie"], "carry_class": "moderate" }, "senses": { "sight": "normal" },
      "disposition": [ { "trait": "prudente", "strength": "strong", "manner": "pide una recomendación técnica antes de decidir" } ],
      "doing": "esperando que el Colegio le dé algo firmado",
      "hiding": "Tiene redactado un plan de triaje que no ha presentado a nadie." },

    { "name": "Auscultador Mayor Renn", "seen_as": "un hombre de 49 años que llega de Belna a pie",
      "role": "Colegio, Belna", "belongs_to": ["Colegio de Auscultadores"], "starts_in": "Belna",
      "capability": { "moves_by": ["a pie"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "el pulso profundo": "acute" },
      "disposition": [ { "trait": "independiente", "strength": "strong", "manner": "publica su propia lectura y no la explica" } ],
      "doing": "firmando un parte que no coincide con el de Ossa por cuarto mes",
      "hiding": "No sabe si su lectura es mejor o si su tercio del Andante está distinto." }
  ],

  "opposition": [
    { "between": ["Del Vas", "Illa"], "incompatible": "El archivo no puede estar cerrado y la lectura del día 4 no puede estar en dos registros a la vez.",
      "stakes": "Quien publique primero decide el triaje de 48.000 personas." }
    // BREAK 9 — la oposición central del mundo no es entre dos personas:
    // 48.000 habitantes contra 12.000 plazas, y 14-30 meses contra 4 años hasta la ventana.
  ],

  "norms": [
    { "name": "licencia previa", "stated": "Toda obra requiere licencia del Gremio de Peso antes de empezar.", "binds": [], "precedent": "el-caso-octavo" },
    { "name": "parte del día 1", "stated": "El Colegio publica el estado del Andante el día 1 de cada mes.", "binds": ["Colegio de Auscultadores"], "precedent": "el-caso-octavo" },
    { "name": "no se come carne de Andante", "stated": "Nadie come carne de Andante. No está escrito y se cumple igual.", "binds": [], "precedent": null },
    { "name": "no delante de menores", "stated": "No se discute la edad de un Andante delante de un niño.", "binds": [], "precedent": null },
    { "name": "el auscultador no se sienta en el Consejo", "stated": "Los auscultadores no participan en el Consejo. En la práctica lo dirigen.", "binds": ["Colegio de Auscultadores"], "precedent": null }
  ],

  "epochs": [
    { "name": "antes del Octavo",
      "differed": [ { "topic": "collective", "subject": "la Flota Muerta", "then": "no existía" },
                    { "topic": "norm", "subject": "parte del día 1", "then": "no existía" } ],
      "surviving_traces": ["el expediente clasificado", "Vieja Marda", "la Flota Muerta misma"] }
  ],

  "history": [
    { "name": "el-caso-octavo",
      "what_happened": "El Octavo murió por causa no establecida y se hundió en seis días. Ruma tenía 60.000 habitantes; se evacuaron 3.100.",
      "where": "Octavo", "who": ["Vieja Marda"],
      "knowledge": [
        { "holder": "Vieja Marda", "channel": "sight", "path": "direct", "believes": "Vio los seis días y sabe qué falló en la evacuación." },
        { "holder": "Del Vas", "channel": "sight", "path": "told", "believes": "El expediente está clasificado para prevenir el alarmismo.", "accurate": false, "plausible_because": "Es el motivo consignado en la carátula." } ] },

    { "name": "la-arritmia-del-dia-4",
      "what_happened": "En la auscultación de rutina del día 4 el pulso profundo de Tercera Hembra salió arrítmico sostenido, con temperatura dorsal +4° en dos puntos y marcha caída de 3,4 a 2,1 km/día.",
      "where": "Alto Omóplato", "who": ["Illa"],
      "knowledge": [
        { "holder": "Illa", "channel": "el pulso profundo", "path": "direct", "believes": "La lectura es correcta: el Andante está en estado crítico." },
        { "holder": "Del Vas", "channel": "el pulso profundo", "path": "direct", "believes": "Lo mismo, desde hace nueve meses, y que decirlo sin salida disponible sólo produce pánico." },
        { "holder": "Registrador Onn", "channel": "sound", "path": "overheard", "believes": "Si hay arritmia es por causa natural y no por las obras que él firmó.", "accurate": false, "plausible_because": "Nadie ha cruzado todavía el Registro de Peso con la fecha." },
        { "holder": "Uma Ret", "channel": "sight", "path": "told", "believes": "El Andante está en observación sin hallazgos.", "accurate": false, "plausible_because": "Es lo que dice el parte firmado del día 1." } ] }
  ],

  "arrivals": [
    { "premise": "Sos aprendiz de cuarto año. Registraste la arritmia el día 4, Illa la confirmó, y el parte firmado del día 1 dice que no hay hallazgos.",
      "seen_as": "alguien con delantal de aprendiz y una lectura anotada en la mano", "place": "Alto Omóplato",
      "capability": { "moves_by": ["a pie"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "el pulso profundo": "acute", "la Campana": "normal" } },
    { "premise": "Llegaste de Belna con el parte de Renn en el bolsillo y nadie en Ossa quiere leerlo delante de otro.",
      "seen_as": "alguien con polvo de cuarenta kilómetros de camino dorsal", "place": "Alto Omóplato",
      "capability": { "moves_by": ["a pie"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "el pulso profundo": "faint", "la Campana": "normal" } }
  ]
}
```

## 2. Breaks

**B1 — A place that walks. (i) inexpressible.** *"Cada Andante recorre una ruta migratoria cerrada, fija, no alterable"*, at 2–5 km/día, cycle 11–40 years (3-detallado 1.4). Should be held by `places[]`. A place has extent, medium and tension — no position, no trajectory. Tercera Hembra becomes an inert `parent` string, and the world's premise ("*la geografía de este mundo es un calendario*", 1.6) vanishes entirely.

**B2 — A place that sickens and dies, taking everything on it. (ii) inert prose.** *"Muerto, año 611… hundimiento en 6 días… 60.000 habitantes; evacuados 3.100"* (2.1). `things[]` have `integrity`; `places[]` do not. A `process` pointed at a place grips nothing, so the death is a sentence in `terminus` and the cascade — every contained district and person — has no expression.

**B3 — Reachability as a function of date. (i) inexpressible.** *"El paso entre Andantes solo es seguro durante Convergencia"*, and the 645 window is 31 days, *"predecible con siglos de anticipación"* (1.5, 8). `ways[].state` is a static open/shut. Worse, a Convergence is the **coincidence of two trajectories** — a derived event, not an authored one. `cycles[]` gives one place a period; nothing composes two.

**B4 — Passage that kills rather than blocks. (iii) wrong-shaped.** *"supervivencia inferior al 30%"* (3.3). `ways[]` offers `obstructs`/`affords` — a binary gate. A crossing that admits everyone and returns 30% of them is not obstruction; encoding it as `affords:["en barcaza"]` says the route is *fine*.

**B5 — Records. (ii) inert prose.** Six of nine `things` are documents whose content is a **claim that can be false**, consultable by procedure (*"tres días de demora"*, 10.3), locked (*"bajo llave del Auscultador Mayor"*) or classified (2.2). The plot *is* a record disagreeing with reality; `things[]` gives them bulk and integrity, and the claim lives nowhere.

**B6 — Conferral. (i) inexpressible.** *"Trompetilla… único instrumento diagnóstico. Se entrega al titularse"* (LIBRO VII). Reading the pulse requires the instrument **and** the grade. `capability` is intrinsic to a body; nothing says a held thing or a rank grants an act.

**B7 — Staged escalation and aggregate load. (iii) wrong-shaped.** *"reducción de velocidad → cojera → lesión articular → enfermedad sistémica"* (3.2) is four ordered rungs; `accumulators[].threshold` is one. And the quantity is the **sum of what is built** (Cuenca 41.300 t, *"sobre el límite del sector desde 639"*, 10.2), while `raised_by` accepts only event names — the schema cannot say "this counter *is* the total of a property over everything contained".

**B8 — Office vs holder. (ii) inert prose.** *"Parte mensual… firma del Auscultador Mayor"* (LIBRO VII) has authority because of the post; Del Vas *"retirarse en el 643"*. Encoded as a `role` string, so when she leaves, the authority leaves with her.

**B9 — The world's real opposition is not interpersonal. (iii) wrong-shaped.** *"Ninguna opción salva a 48.000 personas"* (14.3): 48.000 people against 12.000 places, 14–30 months against 4 years. `opposition[].between` demands exactly two named people, so the central conflict must be forged into a personal quarrel.

**B10 — Hidden state read through unreliable signs. (i) inexpressible, and the largest.** LIBRO III is six indicators × four bands plus a state→plazo mapping *"derivada de un único caso documentado… la base estadística es insuficiente y el Colegio lo sabe"* (5.4), which LIBRO X forbids treating as certain. The whole game is measuring a hidden variable badly. `channels[]` covers who perceives **events** — not a continuous hidden state, its signs, the instrument, or the error. I encoded "el pulso profundo" as a channel and it is a lie.

**B11 — Negative canon. (i) inexpressible.** LIBRO X is eight prohibitions (*"Ninguna trama debe introducir"* a cure; no tenth Andante; sinking is firm). Nothing in the schema bounds what may not exist — and here it is not decoration, it is what stops the model inventing a rescue.

## 3. Minimal fixes — the general primitive, then two unrelated worlds

**F1 · An entity may be carried by another that moves.** Not "Andantes": the primitive is **borne position**. `places[]` gains `borne_by` and `trajectory: {period_class, phase_at_start}`. *Andantes:* Ossa `borne_by` Tercera Hembra; the Andante's trajectory is `generational`. *Second world:* a monastery `borne_by` a glacier with a `geological` trajectory — the same two keys, no shared vocabulary.

**F2 · A way may exist only under a condition, including the coincidence of two trajectories.** `ways[]` gains `exists_when`, taking either a cycle phase or `{coincidence_of: [A, B]}`; the engine derives the calendar and refuses passage outside it. *Andantes:* the Convergence way exists on the coincidence of two routes. *Second world:* a causeway that exists only in the *dry* phase of a lake's cycle.

**F3 · Places are entities: they take `integrity`, `capacity_class`, and may be the target of a process with a terminus, and their destruction cascades to what they contain.** *Andantes:* Tercera Hembra `integrity: failing`, terminus sinks Ossa, Belna and everyone borne. *Second world:* a rotting world-tree whose limb-villages fall with the limb.

**F4 · A threshold ladder, and counters that measure an aggregate.** `accumulators[].thresholds[]` becomes an ordered list; `raised_by` may name `{aggregate_of: "bulk", over: "everything borne by X"}`. *Andantes:* load → four rungs. *Second world:* a granary's stores against a siege, with rationing rungs.

**F5 · Records: a thing whose content is a claim.** `things[]` gains `asserts[]` (statements that may be `accurate: false`), `access {via, delay_class, held_by}`, and `authority`. *Andantes:* the parte asserts "sin hallazgos", inaccurate, authority = an office. *Second world:* a ship's log asserting a course never sailed.

**F6 · Capability may be conferred by a held thing or a standing, not only by a body.** `things[].confers` and `norms[]`/office `confers`. *Andantes:* trompetilla + grade confer reading the pulse. *Second world:* a lamp confers sight in a lightless medium.

**F7 · Offices.** `offices[]`: `{name, held_by, confers, succeeds_by}` — authority that outlives its holder. *Andantes:* Auscultador Mayor. *Second world:* a lighthouse keepership passed by seniority.

**F8 · Observation: hidden state, signs, instrument, reliability.** New top-level `indicators[]`: `{of: <hidden state>, shows_as: [ordered bands], read_by: {channel, requires}, reliability_class}`. The engine derives what an observer reads and never leaks the hidden value. *Andantes:* six indicators over the Andante's condition, `reliability_class: "poor"` (one documented case). *Second world:* an orchard's blight read from leaf colour, unreliable until it is too late.

**F9 · `excluded[]`.** Flat list of statements about what does not exist or cannot happen, binding on every authoring seat. *Andantes:* no cure, no tenth Andante, no route change. *Second world:* "nobody here can be lied to" — the same section, no shared vocabulary.

**F10 · Hazard on passage.** `ways[].hazard_class` — a route that admits and harms, distinct from one that blocks. *Andantes:* barge crossing, `severe`. *Second world:* a scree slope that takes an ankle from most who cross.

## 4. Tier fidelity

**The schema does not degrade gracefully, because the breaks are premise-driven, not detail-driven.** Tier 1 already states moving places (§1), the weight limit (3.1.2), Convergence-only passage (3.1.3), sinking in six days (3.1.4) and a dead Andante (§2). Every one of B1–B4 fires on the *basic* brief. The sparse tier does not author less; it authors the same impossible things with less around them.

What the extra detail has nowhere to go is specific: the indicator table and plazo mapping (LIBRO III), the evacuation arithmetic (§9), the *"punto débil"* column that makes each institution corruptible (§7), the training ladder (§6), the per-district tonnages (10.2), and LIBRO X entire. It all lands as prose inside `role`, `hiding` and `interest` — the detailed tier makes the document longer without making the world more mechanical, the precise failure this schema exists to prevent.

Two smaller notes. `senses{}` and `conditions[]` are mandatory-shaped and this world has **no** bodily variation at any tier — I wrote "normal" nine times. And the class rule correctly refuses the brief's numbers, but the *published* figures (the Campana's daily count, the Tables' dates) are quoted aloud by characters; without F5 and F8 a derived, publicly-cited quantity has nowhere to live.

## 5. The one shape change

**Stop separating `places` from `things`.** They are the same object at different scales, and this world proves it: Tercera Hembra is a *thing* (integrity, bulk, capacity, it can die) that is also a *place* (it contains districts and people), and it is *carried* nothing but itself walking. The schema's split forced me to encode the single most important entity in the world twice — once as an inert `parent` string and once as the target of a process that cannot grip it — and to lose the relation between the two halves.

Collapse them into one recursive `entities[]` where *being a place* is simply having interior extent, *being carried* is a relation to another entity, and integrity, capacity, trajectory and medium are properties any entity may have. `things[].supports` and `places[].parent` become the same edge. That single change absorbs F1, F3 and most of F4 for free, and it is the difference between a schema that can hold a world whose ground is alive and one that assumes the ground is scenery.
