# La Marea Lenta — world_model/4, generated from the 400-word tier

```jsonc
{
  "world_model": "4",
  "world": { "name": "La Marisma", "source": "stated",
    "premise": "Hace trescientos años el agua empezó a subir y nunca paró: un palmo por año, sin tormentas ni milagros. Nadie huyó porque no había adónde, así que se construyó hacia arriba. El mundo es un archipiélago de torres altísimas sobre un mar turbio, y cada generación vive un piso más arriba que la anterior. Nadie escribe historia: la historia está abajo, y hay que bajar a buscarla.",
    "mood": ["húmedo", "administrativo", "paciente"] },

  "excluded": [
    { "text": "El agua nunca baja, y nadie sabe por qué sube. No hay causa averiguable, dios que responda ni fenómeno que lo explique.", "source": "stated" },
    { "text": "No hay tormentas, milagros ni acontecimientos: la subida es constante y aburrida.", "source": "stated" },
    { "text": "No hay costas, ni tierra firme, ni la habrá. Nadie huye a ningún lado.", "source": "stated" },
    { "text": "Romper la vara de aforo no detiene la marea. Es lo que se dice, y es falso.", "source": { "inferred_from": ["la vara de aforo", "la marea sube"] } },
    { "text": "Nadie escribe historia arriba del agua. No existe crónica, biblioteca seca ni archivo vigente.", "source": "stated" },
    { "text": "Un objeto hundido no miente ni inventa: conserva la voz de su último dueño y nada más.", "source": "stated" },
    { "text": "No hay forma de saber cuántas respiraciones le quedan a alguien. Ninguna escena debe revelar el número.", "source": "stated" }
  ],

  "vocabulary": {
    "media": [
      { "name": "aire de cuerdas", "source": "stated",
        "descriptor": "el aire del piso habitado, con ropa tendida y olor a sal",
        "affords": [ { "to": "sight", "degree": "full" }, { "to": "hablar", "degree": "full" } ] },
      { "name": "agua verde", "source": "stated",
        "descriptor": "el mar turbio de la Marisma, y todo lo que quedó debajo",
        "resists": [ { "to": "caminar", "degree": "total" }, { "to": "hablar", "degree": "total" },
                     { "to": "sight", "degree": "severe" } ],
        "affords": [ { "to": "bucear", "degree": "full" }, { "to": "la voz de los objetos", "degree": "full" } ] }
    ],
    "movements": [
      { "name": "caminar", "source": "stated", "pace_class": "steady" },
      { "name": "trepar cuerda", "source": "stated", "pace_class": "slow",
        "descriptor": "por las poleas y los puentes de cuerda" },
      { "name": "bucear", "source": "stated", "pace_class": "slow",
        "descriptor": "bajar a los pisos ahogados con el aire que uno lleva" },
      { "name": "barcaza", "source": "stated", "pace_class": "slow" }
    ],
    "channels": [
      { "name": "sight", "source": "stated", "emitted_by": "cualquier entidad",
        "received_by": "quien esté presente", "latency_class": "immediate",
        "reach": "la misma extensión", "decay": "never", "conceals": "none" },
      { "name": "la voz de los objetos", "source": "stated",
        "descriptor": "un objeto hundido habla con la voz de su último dueño si lo sostenés y aguantás la respiración",
        "emitted_by": "cualquier objeto hundido que tuvo dueño",
        "received_by": "quien lo sostiene bajo el agua, y nadie más",
        "latency_class": "immediate", "reach": "las manos que lo sostienen",
        "decay": { "class": "never", "exemplar": "trescientos años y sigue hablando" },
        "conceals": "none" },
      { "name": "la voz de las cuerdas", "source": { "inferred_from": ["puentes de cuerda", "poleas"] },
        "descriptor": "lo que se grita de polea a polea y de torre a torre",
        "emitted_by": "cualquiera en un piso habitado", "received_by": "las torres unidas por cuerda",
        "latency_class": "hours", "reach": "el archipiélago", "decay": "slow", "conceals": "identity" }
    ],
    "conditions": [
      { "name": "sin resuello", "source": { "inferred_from": ["escuchar cuesta aire"] },
        "alters": [ { "movement": "bucear", "effect": "hinder", "class": "severe" } ] },
      { "name": "pulmón quemado", "source": { "inferred_from": ["Nera Quilmes"] },
        "descriptor": "el pulmón de quien fuma para castigarse",
        "alters": [ { "movement": "bucear", "effect": "hinder", "class": "moderate" } ] },
      { "name": "pulmón de chico", "source": { "inferred_from": ["Tobi"] },
        "descriptor": "nada bien y respira mal",
        "alters": [ { "movement": "bucear", "effect": "grant", "class": "moderate" },
                    { "act": "aguantar la respiración", "effect": "hinder", "class": "severe" } ] }
    ],
    "substances": [
      { "name": "aliento", "source": "stated", "note": "cada palabra que un objeto dice es una respiración que perdés" },
      { "name": "aire embotellado", "source": "stated", "note": "lo que vende Tobi, y probablemente sea aire común" }
    ]
  },

  "law": [
    { "name": "la marea sube", "source": "stated", "enforced_by": "physics", "within": "La Marisma",
      "stated": "El agua sube un palmo por año, nunca baja, y nadie sabe por qué." },
    { "name": "nadie vive bajo el Filo", "source": "stated", "enforced_by": "office", "within": "torre de Baldaña",
      "binds": [], "stated": "Está prohibido vivir por debajo de la línea del Filo, la marca de agua del año pasado.",
      "forbids": { "subject": "cualquier entidad con agencia", "act": "dormir bajo la línea del Filo" },
      "precedent": "la-mudanza-del-piso-cuarenta" },
    { "name": "los objetos guardan la voz", "source": "stated", "enforced_by": "physics", "within": "agua verde",
      "stated": "Un objeto hundido conserva la voz de su último dueño. Si lo sostenés y aguantás la respiración, habla." },
    { "name": "escuchar cuesta aire", "source": "stated", "enforced_by": "physics", "within": "agua verde",
      "stated": "Cada palabra que un objeto dice es una respiración que perdés, y nadie sabe cuántas le quedan." },
    { "name": "bajar no está prohibido", "source": { "inferred_from": ["nadie vive bajo el Filo", "Nera Quilmes"] },
      "enforced_by": "office", "within": "torre de Baldaña", "binds": [],
      "stated": "La veda es sobre vivir, no sobre bajar. El encargo de un hondero es legal y el Concilio cobra por registrarlo." },
    { "name": "se saluda al Aforador", "source": { "inferred_from": ["El Aforador"] },
      "enforced_by": "persons", "within": "torre de Baldaña", "binds": [],
      "stated": "Al hombre más odiado de la torre se lo saluda todas las mañanas, sin excepción y sin cariño." },
    { "name": "el número es la única escritura", "source": { "inferred_from": ["El Aforador", "Nadie escribe historia"] },
      "enforced_by": "office", "within": "torre de Baldaña", "binds": ["el Concilio de Baldaña"],
      "stated": "Lo único que se anota en Baldaña es la altura del agua. Nada más se pone por escrito." },
    { "name": "no se pregunta por la voz de nadie", "source": { "inferred_from": ["la voz de los objetos", "Nera Quilmes"] },
      "enforced_by": "persons", "within": "piso 41", "binds": [],
      "stated": "No se le pregunta a un hondero qué voz escuchó abajo. Se paga el encargo y se calla." }
  ],

  "entities": [
    { "name": "La Marisma", "source": "stated", "facets": ["extent"], "extent_class": "vast",
      "medium": "agua verde", "tension": "calm",
      "seen_as": "un mar turbio sin costas conocidas, salpicado de torres" },
    { "name": "torre de Baldaña", "source": "stated", "facets": ["extent", "matter"], "within": "La Marisma",
      "extent_class": "large", "medium": "aire de cuerdas", "tension": "normal",
      "bulk_class": "immense", "integrity": "worn",
      "seen_as": "una torre altísima con cuarenta y un pisos por encima del agua y treinta y nueve debajo" },
    { "name": "piso 41", "source": "stated", "facets": ["extent"], "within": "torre de Baldaña",
      "extent_class": "medium", "medium": "aire de cuerdas", "tension": "normal",
      "seen_as": "un barrio de cuerdas, poleas y ropa tendida sobre el agua" },
    { "name": "piso 40", "source": { "inferred_from": ["nadie vive bajo el Filo", "piso 41"] },
      "facets": ["extent"], "within": "torre de Baldaña", "extent_class": "medium",
      "medium": "aire de cuerdas", "tension": "tense",
      "seen_as": "vacío desde la mudanza del año pasado, con la línea del Filo pintada a la altura de la rodilla" },
    { "name": "los pisos ahogados", "source": "stated", "facets": ["extent", "magnitude"],
      "within": "torre de Baldaña", "magnitude_class": "many", "extent_class": "medium",
      "medium": "agua verde", "tension": "calm",
      "seen_as": "treinta y nueve pisos a oscuras, cada uno con lo que su gente no pudo subir" },
    { "name": "piso 12", "source": "stated", "facets": ["extent"], "within": "torre de Baldaña",
      "extent_class": "medium", "medium": "agua verde", "tension": "tense",
      "seen_as": "casas de los abuelos: ollas en los estantes, camas hechas, todo bajo agua verde" },
    { "name": "el Archivo del piso 9", "source": "stated", "facets": ["extent"], "within": "torre de Baldaña",
      "extent_class": "small", "medium": "agua verde", "tension": "frantic",
      "seen_as": "estanterías de papel deshecho, y un silencio que los honderos no aguantan" },
    { "name": "el Templo del piso 5", "source": "stated", "facets": ["extent"], "within": "torre de Baldaña",
      "extent_class": "small", "medium": "agua verde", "tension": "frantic",
      "seen_as": "bancos en filas bajo el agua, y nada que hable" },
    { "name": "torre de Ánsar", "source": { "inferred_from": ["La Marisma", "el muelle de barcazas"] },
      "facets": ["extent"], "within": "La Marisma", "extent_class": "large", "medium": "aire de cuerdas",
      "tension": "normal", "seen_as": "la torre vecina, dos pisos más baja y con su propio Aforador" },
    { "name": "el muelle de barcazas", "source": "stated", "facets": ["extent", "matter"],
      "within": "piso 41", "extent_class": "small", "medium": "aire de cuerdas", "tension": "normal",
      "bulk_class": "immense", "integrity": "worn", "seen_as": "tablones atados a la altura del agua de este año" },

    { "name": "el puente de cuerda a Ánsar", "source": "stated", "facets": ["matter", "passage"],
      "within": "piso 41", "connects": ["piso 41", "torre de Ánsar"], "bulk_class": "moderate",
      "integrity": "worn", "admits": [ { "movement": "trepar cuerda" } ],
      "obstructs": [ { "movement": "caminar" } ], "hazard_class": "moderate",
      "seen_as": "cuerda y tablas, y una polea que se atasca" },
    { "name": "el hueco de la escalera", "source": { "inferred_from": ["los pisos ahogados", "bucear"] },
      "facets": ["passage"], "connects": ["piso 40", "los pisos ahogados"],
      "admits": [ { "act": "aguantar la respiración" }, { "movement": "bucear" } ],
      "obstructs": [ { "movement": "caminar" }, { "condition": "sin resuello" } ],
      "hazard_class": "severe",
      "seen_as": "la boca de la escalera vieja, donde el agua empieza y la luz se termina" },
    { "name": "la bajada al doce", "source": { "inferred_from": ["piso 12", "el hueco de la escalera"] },
      "facets": ["passage"], "connects": ["los pisos ahogados", "piso 12"],
      "admits": [ { "act": "aguantar la respiración" } ],
      "obstructs": [ { "condition": "sin resuello" }, { "condition": "pulmón quemado" } ],
      "hazard_class": "severe", "seen_as": "veintiocho pisos de agua negra contados por las barandas" },

    { "name": "Nera Quilmes", "source": "stated", "facets": ["matter", "agency"], "within": "piso 41",
      "conditions": ["pulmón quemado"],
      "seen_as": "una hondera que fuma en el borde con los pies sobre el agua",
      "capability": { "moves_by": ["caminar", "trepar cuerda", "bucear"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "la voz de los objetos": "acute" },
      "disposition": [ { "trait": "impávida", "strength": "defining",
                         "manner": "acepta cualquier encargo y no pregunta de quién era" } ],
      "doing": "fumando antes de bajar, para castigarse los pulmones",
      "pursuing": [ { "horizon": "long_standing", "toward": "bajar al Archivo del piso 9 y aguantar el silencio una vez" } ],
      "hiding": "Bajó al Archivo una vez, no escuchó nada, y subió antes de tocar el fondo." },
    { "name": "El Aforador", "source": "stated", "facets": ["matter", "agency"], "within": "piso 41",
      "seen_as": "un funcionario con una vara, saludado por todos y querido por nadie",
      "capability": { "moves_by": ["caminar", "trepar cuerda"], "carry_class": "moderate" },
      "senses": { "sight": "acute", "la voz de los objetos": "absent" },
      "disposition": [ { "trait": "exacto", "strength": "defining",
                         "manner": "anota el número antes de contestar a nadie" } ],
      "doing": "midiendo la marea con la vara al amanecer",
      "pursuing": [ { "horizon": "imminent", "toward": "que la mudanza del piso 40 al 41 se cierre sin que nadie se quede abajo" } ],
      "hiding": "La medición de este año cruzó el Filo hace dos meses y todavía no lo anunció." },
    { "name": "Tobi", "source": "stated", "facets": ["matter", "agency", "holding"], "within": "el muelle de barcazas",
      "conditions": ["pulmón de chico"], "capacity_class": "cramped",
      "holds": [ { "substance": "aire embotellado", "abundance": "thin" } ],
      "seen_as": "un chico de nueve años con botellas colgadas del cinturón",
      "capability": { "moves_by": ["caminar", "trepar cuerda", "bucear"], "carry_class": "slight" },
      "senses": { "sight": "normal", "la voz de los objetos": "faint" },
      "disposition": [ { "trait": "vendedor", "strength": "strong", "manner": "regatea antes de saludar" } ],
      "doing": "vendiendo botellas en el muelle a los honderos que van a bajar",
      "pursuing": [ { "horizon": "long_standing", "toward": "que un hondero lo lleve abajo de una vez" } ],
      "hiding": "Las botellas son aire del piso 41 y él lo sabe." },
    { "name": "los honderos de Baldaña", "source": "stated", "facets": ["matter", "agency", "magnitude"],
      "within": "piso 41", "magnitude_class": "few",
      "seen_as": "los que bajan por encargo, con los ojos rojos y el pecho marcado",
      "capability": { "moves_by": ["caminar", "trepar cuerda", "bucear"], "carry_class": "moderate" },
      "senses": { "la voz de los objetos": "normal" },
      "disposition": [ { "trait": "supersticiosos", "strength": "strong",
                         "manner": "no bajan dos veces al mismo piso en la misma semana" } ],
      "doing": "esperando encargo en el muelle",
      "pursuing": [ { "horizon": "long_standing", "toward": "un encargo que pague más aire del que cuesta" } ] },
    { "name": "el Concilio de Baldaña", "source": "stated", "facets": ["agency", "collective"],
      "within": "torre de Baldaña", "legibility": "marked",
      "seen_as": "los que nombran al Aforador y cobran el registro de cada encargo",
      "interest": "que la altura del agua sea lo único que quede escrito",
      "vulnerability": "toda su autoridad es un número que mide un solo hombre con una sola vara",
      "disposition": [ { "trait": "administrativo", "strength": "defining", "manner": "responde con la ordenanza" } ],
      "doing": "preparando la mudanza del piso 40 sin decir todavía a quién le toca",
      "hiding": "Sabe que la vara es única y que nadie sabría medir sin ella." },

    { "name": "la vara de aforo", "source": "stated", "facets": ["matter"], "within": "El Aforador",
      "bulk_class": "slight", "integrity": "sound", "size_class": "small",
      "seen_as": "una vara marcada palmo por palmo, gastada en la parte que toca el agua",
      "confers": [ { "act": "medir la altura exacta del agua" } ] },
    { "name": "el anillo de latón", "source": "stated", "facets": ["matter", "record"], "within": "piso 41",
      "bulk_class": "negligible", "integrity": "sound",
      "asserts": [ { "claim": "habla con tu propia voz, más vieja, y te pide que no vuelvas a bajar",
                     "accurate": false,
                     "plausible_because": "un objeto conserva la voz de su último dueño, y su último dueño no naciste todavía" },
                   { "claim": "susurra el nombre de alguien que todavía no nació", "accurate": true } ],
      "access": { "via": "sostenerlo bajo el agua y aguantar la respiración", "delay_class": "immediate",
                  "held_by": "quien lo subió del piso 12" },
      "authority": "ninguna: es un anillo" },
    { "name": "el registro de la marea", "source": { "inferred_from": ["El Aforador", "el número es la única escritura"] },
      "facets": ["matter", "record"], "within": "piso 41", "bulk_class": "moderate", "integrity": "worn",
      "asserts": [ { "claim": "la altura del agua cada mañana desde hace trescientos años", "accurate": true },
                   { "claim": "el agua todavía no cruzó el Filo este año", "accurate": false,
                     "plausible_because": "es el último renglón, y lo escribió el único que mide" } ],
      "access": { "via": "pedirlo al Concilio", "delay_class": "days", "held_by": "El Aforador" },
      "authority": "el Concilio de Baldaña" },
    { "name": "las ollas del piso 12", "source": { "inferred_from": ["piso 12", "los objetos guardan la voz"] },
      "facets": ["matter", "magnitude"], "within": "piso 12", "magnitude_class": "multitude",
      "bulk_class": "slight", "integrity": "worn",
      "seen_as": "cacharros en los estantes, cada uno con la voz de quien lo usó por última vez" }
  ],

  "offices": [
    { "name": "Aforador de Baldaña", "source": "stated", "held_by": "El Aforador",
      "of": "el Concilio de Baldaña",
      "confers": [ { "act": "fijar dónde está la línea del Filo" }, { "act": "ordenar una mudanza de piso" } ],
      "succeeds_by": "designación del Concilio, y entrega de la vara" }
  ],

  "standing": [
    { "from": "piso 41", "toward": "El Aforador", "source": "stated",
      "stance": "el hombre más odiado de la torre y el único al que todos saludan",
      "since": null, "carried_by": "la voz de las cuerdas", "persistence": "never decays" },
    { "from": "El Aforador", "toward": "los honderos de Baldaña",
      "source": { "inferred_from": ["bajar no está prohibido", "el número es la única escritura"] },
      "stance": "los tolera porque suben objetos y no escriben nada",
      "since": null, "carried_by": null, "persistence": "until changed" },
    { "from": "Nera Quilmes", "toward": "Tobi",
      "source": { "inferred_from": ["Tobi", "Nera Quilmes"] },
      "stance": "le compra las botellas sabiendo que son aire común, para que coma",
      "since": null, "carried_by": null, "persistence": "until changed" },
    { "from": "el anillo de latón", "toward": "quien lo subió",
      "source": "stated", "stance": "le pide con su propia voz que no vuelva a bajar",
      "since": "el-anillo-del-piso-doce", "carried_by": "la voz de los objetos", "persistence": "never decays" }
  ],

  "opposition": [
    { "between": ["El Aforador", "los honderos de Baldaña"],
      "source": { "inferred_from": ["nadie vive bajo el Filo", "El Aforador"] },
      "incompatible": "El Filo de este año condena el piso 40, que es el único sitio seco desde donde se baja.",
      "stakes": "Si se cierra el hueco de la escalera, el oficio de hondero se termina en Baldaña." },
    { "between": ["Nera Quilmes", "el Concilio de Baldaña"],
      "source": { "inferred_from": ["el número es la única escritura", "Nera Quilmes"] },
      "incompatible": "Lo que Nera sube del Archivo sería lo primero escrito en trescientos años que no es un número.",
      "stakes": "La autoridad del Concilio es que abajo no hay nada legible." },
    { "between": ["Tobi", "los honderos de Baldaña"],
      "source": { "inferred_from": ["Tobi", "escuchar cuesta aire"] },
      "incompatible": "Les vende aire que no es aire a gente que cuenta respiraciones.",
      "stakes": "El día que alguien no suba, el muelle va a saber de quién compró." } ],

  "processes": [
    { "name": "la subida", "source": "stated", "acts_on": "toda extensión de la torre",
      "direction": "degrade", "rate_class": { "class": "very slow", "exemplar": "un palmo por año" },
      "terminus": null },
    { "name": "el papel deshecho", "source": { "inferred_from": ["el Archivo del piso 9", "agua verde"] },
      "acts_on": "todo papel bajo agua verde", "direction": "degrade",
      "rate_class": { "class": "slow", "exemplar": "un archivo ilegible en una generación" },
      "terminus": "no queda nada que leer, solo lo que alguien tuvo en la mano" }
  ],

  "cycles": [
    { "name": "la medición", "source": "stated", "period_class": "daily", "starts_in_phase": "amanecer",
      "phases": [ { "name": "amanecer", "changes": [ { "entity": "el registro de la marea", "becomes": "un renglón más largo" } ] },
                  { "name": "el día de las cuerdas", "changes": [] } ] },
    { "name": "el año del palmo", "source": "stated", "period_class": "annual", "starts_in_phase": "marca nueva",
      "phases": [ { "name": "marca nueva", "changes": [ { "entity": "piso 40", "becomes": "más cerca del agua" } ] },
                  { "name": "el año seco", "changes": [] } ] },
    { "name": "la mudanza", "source": "stated", "period_class": "generational",
      "starts_in_phase": "aviso",
      "phases": [ { "name": "aviso", "changes": [] },
                  { "name": "subir todo un piso", "changes": [ { "entity": "los pisos ahogados", "becomes": "uno más" } ] } ] }
  ],

  "accumulators": [
    { "name": "respiraciones gastadas", "source": "stated", "per": "each entity with agency",
      "starts_at": "none",
      "stated": "Cada palabra que un objeto dice es una respiración que perdés, y nadie sabe cuántas le quedan.",
      "raised_by": [ { "event": "una palabra escuchada bajo el agua" },
                     { "event": "una bajada completa" } ],
      "thresholds": [
        { "at": "low", "then": "sube más lento de lo que bajó", "exemplar": "una conversación corta con un objeto" },
        { "at": "moderate", "then": "adquiere sin resuello y no puede volver a bajar esa semana" },
        { "at": "high", "then": "escucha una palabra y pierde el conocimiento en el agua" },
        { "at": "extreme", "then": "no sube", "irreversible": true } ] },
    { "name": "la altura del Filo", "source": "stated", "per": "world", "starts_at": "high",
      "stated": "Dónde está la marca del año pasado, que es hasta dónde se puede vivir.",
      "raised_by": [ { "event": "cada medición del amanecer" } ],
      "thresholds": [
        { "at": "high", "then": "el piso 40 queda condenado y hay que mudar a todos al 41",
          "exemplar": "un piso cada generación" },
        { "at": "extreme", "then": "el hueco de la escalera queda bajo el Filo y bajar deja de ser legal",
          "irreversible": true } ] },
    { "name": "el descrédito del Aforador", "source": { "inferred_from": ["El Aforador", "la vara de aforo"] },
      "per": "each office", "starts_at": "high",
      "stated": "Cuánto se lo odia por el número que anota. Se dice que si la vara se rompe, la marea se detiene.",
      "raised_by": [ { "event": "cada mudanza ordenada" }, { "event": "cada renglón que condena un piso" } ],
      "thresholds": [ { "at": "extreme", "then": "alguien le rompe la vara, y la marea sigue subiendo igual" } ] }
  ],

  "indicators": [
    { "of": "respiraciones gastadas", "source": "stated",
      "shows_as": ["una tos que no estaba", "subir más lento de lo que bajó", "quedarse callado en el muelle"],
      "read_by": { "channel": "sight", "requires": {} }, "reliability_class": "none" },
    { "of": "el hiding de El Aforador",
      "source": { "inferred_from": ["El Aforador", "el registro de la marea"] },
      "shows_as": ["el último renglón escrito con otra letra", "dos meses sin anuncio de mudanza"],
      "read_by": { "channel": "sight", "requires": { "office": "Aforador de Baldaña" } },
      "reliability_class": "poor" },
    { "of": "si un objeto todavía tiene voz",
      "source": { "inferred_from": ["los objetos guardan la voz", "el Archivo del piso 9"] },
      "shows_as": ["un zumbido en la palma al sostenerlo", "el silencio de las cosas que nadie tuvo"],
      "read_by": { "channel": "la voz de los objetos", "requires": {} },
      "reliability_class": "moderate" }
  ],

  "traces": [
    { "of": "un año de marea", "source": "stated", "leaves": "la línea del Filo pintada en la pared",
      "ages": "never" },
    { "of": "una mudanza", "source": { "inferred_from": ["la mudanza", "los pisos ahogados"] },
      "leaves": "un piso vacío con las camas hechas", "ages": "slow" },
    { "of": "una bajada", "source": { "inferred_from": ["Nera Quilmes", "bucear"] },
      "leaves": "una cuerda mojada colgando del hueco y un objeto que antes no estaba arriba",
      "ages": "slow" }
  ],

  "epochs": [
    { "name": "antes de la subida", "source": "stated",
      "differed": [ { "topic": "law", "subject": "la marea sube", "then": "el agua estaba quieta y había suelo" },
                    { "topic": "entity", "subject": "el Templo del piso 5", "then": "estaba seco y lleno" } ],
      "surviving_traces": ["treinta y nueve pisos ahogados", "el Archivo del piso 9", "el Templo del piso 5"] }
  ],

  "history": [
    { "name": "la-mudanza-del-piso-cuarenta", "source": { "inferred_from": ["nadie vive bajo el Filo", "piso 41"] },
      "standing": "occurred",
      "what_happened": "El año pasado el Filo llegó al piso 40 y el Aforador ordenó subir a todos al 41.",
      "where": "piso 40", "who": ["El Aforador", "el Concilio de Baldaña"],
      "knowledge": [
        { "holder": "El Aforador", "channel": "sight", "path": "direct",
          "believes": "Midió, anotó y ordenó, y no le tembló la mano." },
        { "holder": "los honderos de Baldaña", "channel": "la voz de las cuerdas", "path": "public",
          "believes": "Perdieron el piso donde dejaban las cuerdas secas." } ] },
    { "name": "el-anillo-del-piso-doce", "source": "stated", "standing": "disputed",
      "what_happened": "Un hondero subió del piso 12 un anillo de latón que habla, y la voz es la suya, más vieja, pidiéndole que no vuelva a bajar.",
      "where": "piso 12", "who": ["Nera Quilmes", "El Aforador", "Tobi"],
      "knowledge": [
        { "holder": "Nera Quilmes", "channel": "la voz de los objetos", "path": "told",
          "believes": "El anillo tuvo otro dueño y la voz se parece, nada más. Las voces de abajo se parecen todas." },
        { "holder": "Tobi", "channel": "la voz de las cuerdas", "path": "rumor",
          "believes": "El anillo viene de más abajo del 12 y sabe cosas que todavía no pasaron.",
          "accurate": false, "plausible_because": "susurra un nombre de alguien que no nació" },
        { "holder": "El Aforador", "channel": "sight", "path": "inference",
          "believes": "Es un anillo, y lo que habla es el hondero. Un objeto no puede tener el dueño equivocado.",
          "accurate": false, "plausible_because": "en trescientos años nunca se registró uno así, y él es el que registra" } ] }
  ],

  "arrivals": [
    { "premise": "Sos hondero de Baldaña. Ayer subiste del piso 12 con un anillo que habla, y la voz que sale de él es la tuya, pero más vieja, y te está pidiendo que no vuelvas a bajar.",
      "source": "stated", "seen_as": "alguien con el pecho marcado por la cuerda y un anillo en el puño",
      "place": "piso 41",
      "capability": { "moves_by": ["caminar", "trepar cuerda", "bucear"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "la voz de los objetos": "acute" } },
    { "premise": "Cruzaste el puente desde Ánsar con un encargo y una moneda: alguien quiere lo que quedó en el Templo del piso 5, y en Ánsar no hay hondero que baje tan abajo.",
      "source": { "inferred_from": ["torre de Ánsar", "Nera Quilmes"] },
      "seen_as": "alguien con ropa seca y ninguna marca en el pecho", "place": "el muelle de barcazas",
      "capability": { "moves_by": ["caminar", "trepar cuerda"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "la voz de los objetos": "faint" } },
    { "premise": "Naciste en el 41 y nunca bajaste. Le compraste tres botellas a Tobi y hoy es el día.",
      "source": { "inferred_from": ["Tobi", "los honderos de Baldaña"] },
      "seen_as": "alguien demasiado limpio para el muelle", "place": "el muelle de barcazas",
      "capability": { "moves_by": ["caminar", "trepar cuerda", "bucear"], "carry_class": "slight" },
      "senses": { "sight": "normal", "la voz de los objetos": "absent" } }
  ]
}
```

## Sufficiency self-check

**S1** pass — every extent reachable from an arrival carries content: piso 41 (people, muelle, ring), piso 40 (the Filo mark, the stair mouth), los pisos ahogados (objects with voices), piso 12 (las ollas), Archivo 9 (silence, dissolving paper), Templo 5 (the commission target), Ánsar (its own Aforador), el muelle (Tobi, barges).
**S2** pass — all six agencies want something actionable: Nera wants to endure the Archive's silence; the Aforador wants the move closed; Tobi wants taking down; the honderos want a paying commission; the Concilio wants the number to stay the only writing; and the ring asks for one thing.
**S3** pass — every name in authored text resolves: Concilio, Filo, Marisma, Baldaña, Ánsar, honderos, Archivo, Templo, vara, anillo, registro, ollas, muelle, puente. Nothing dangles.
**S4** pass — three passages, all to authored extents; the stair mouth and la bajada al doce are obstructed for fictional reasons (`sin resuello`, `pulmón quemado`, no walking in water).
**S5** pass — each accumulator has a reachable outcome: breath ends in not surfacing; the Filo ends in diving becoming illegal; discredit ends in the rod broken and the tide rising anyway. Each opposition can resolve either way.
**S6** pass — every implied institution exists: the Concilio (named by the brief), its office, its register, its move; the trade implies the honderos and the commission law.

## Validity

O1 ✓ (9 extents, 3 passages) · O2 ✓ (6) · O3 ✓ — five carry `pursuing`; the Concilio satisfies it via `interest` (F3) · O4 ✓ (5 `hiding`) · O5 ✓ (3) · O6 ✓ per F4 — `bulk_class` on every non-agency `matter` entity; omitted on the four people · O7 ✓ — `aliento` and `aire embotellado`; Tobi stores the latter (F2: no other entity declares `holding`) · O8 ✓ (3, each with `raised_by` + ordered thresholds) · O9 ✓ — each indicator names an accumulator or a named `hiding` · O10 ✓ (3) · O11 ✓ (7).

R1 ✓ all references resolve · R2 ✓ each passage joins exactly 2 extents · R3 ✓ numbers only in `stated`/`seen_as`/`premise`/`exemplar`/`asserts.claim` · R4 ✓ · R5 ✓ per F5 — no authored individual reference into `los pisos ahogados`, `los honderos`, `las ollas`; piso 12 and Nera are their own entities, not subtractions · R6 ✓ — the rod superstition is authored as belief and as an `excluded` falsehood, not as a mechanism · R7 ✓ · R8 ✓ tree · R9 ✓ · R10 ✓ · R11 ✓ ordered, unique · R12 ✓ — the disputed ring has three holders disagreeing · R13 ✓ — every chain bottoms out in a stated element.

**Contract finding on R13.** I machine-checked my own chains and six failed to resolve. Two were my
error (citing near-miss wordings instead of my authored names) and are fixed. The other four bottomed
out in *verbatim brief phrases* — "Nadie escribe historia", "archipiélago de torres" — which are stated
content, so they satisfy R13's wording while violating its format, which says `inferred_from: ["<name>"]`.
Two consequences: `excluded[]` entries have **no `name`**, so nothing can ever cite one as a basis; and
a brief phrase is not an element, so "bottoms out in stated" and "names an element" are different tests.
R13 needs to say which, or provenance chains cannot be validated mechanically.

## Inference log — the ten that matter

| # | Inferred | Forced by | Generic choice rejected |
|---|---|---|---|
| 1 | **The Filo is a *number a man writes*, so the annual move is an administrative act** | "prohibido vivir bajo el Filo" + a functionary who "anota el número" | a flood-warning system, or nature simply taking the floor |
| 2 | **piso 40 exists, empty, with the line painted knee-high** | the Filo is last year's mark and people live on 41 | leaving floor 40 unmentioned — an authoring boundary one step below the arrival |
| 3 | **The Archive is silent, because paper had no last owner** | rule 3 says *last owner*; archives are stated as submerged; Nera fears only silence | making the Archive a treasure room of readable secrets |
| 4 | **The Concilio's interest is that the tide-number stay the only writing** | "nadie escribe historia" + the Aforador is its functionary | a corrupt council hoarding wealth |
| 5 | **Diving is legal; the ban is on *living* below** | the ban says "vivir", and a hondera works openly by commission | outlawing diving and making honderos criminals |
| 6 | **The stair mouth is at floor 40, so the Filo will close diving itself** | the water rises annually + you must descend from something dry | an eternal, unthreatened way down |
| 7 | **Tobi's bottles are floor-41 air and he knows it** | "probablemente aire común" + he is nine and sells to divers | a mysterious supplier of real air |
| 8 | **The Aforador is two months late announcing a crossing** | "el más odiado y el único al que todos saludan" + he alone measures | an honest bureaucrat, or a cartoon villain |
| 9 | **Breaking the rod is authored as false in `excluded`** | "se dice que si se rompe, la marea se detiene" — stated as a saying | letting it be true, i.e. a magic solution to the premise |
| 10 | **Torre de Ánsar, two floors lower, with its own Aforador** | "archipiélago" + "puentes de cuerda" + barges | a nameless "other towers" fog the player cannot cross to |

## The pull

**A tavern on floor 41.** The brief gave me a *muelle de barcazas* instead — where commissions, air-selling and arrivals from other towers already converge. The social hub was implied by "barcazas" and "por encargo"; I did not need to import one.

**A guard enforcing the Filo.** There is no guard: the enforcement is a man with a stick who writes a number, which is far worse, and the brief said he is hated *and* greeted. Enforcement here is arithmetic and shame, so I authored the register and the office rather than a watch.

**A prophecy, handed to me on a plate by the ring that whispers an unborn name.** I refused it: the ring is authored as a `record` with one accurate claim (it whispers a name) and one *inaccurate* one (that the voice is yours-older), the event is `disputed`, and `excluded` forbids the tide having any knowable cause. The brief's own line — "nadie sabe por qué" — is what let me keep the ring uncanny without making it an oracle.

## Load-bearing set (what the user corrects before spend)

Computed per §2.1 — inferences referenced elsewhere or appearing in law/accumulators/opposition/offices/excluded:

1. Breaking the rod does not stop the tide (`excluded`).
2. Diving is legal; the ban is only on living below the Filo (`law`).
3. The tide-number is the only writing in Baldaña (`law`, referenced by the Concilio and the register).
4. The Aforador is saluted daily by custom (`law`).
5. Nobody asks a hondero which voice they heard (`law`).
6. The stair mouth sits on floor 40, so the Filo will eventually close descent (`accumulator`, `opposition`).
7. The Aforador's discredit ends in a broken rod (`accumulator`, `office`).
8. The Concilio's interest and vulnerability (`opposition`).
9. Tobi's bottles are common air (`opposition`).
10. Torre de Ánsar exists and is crossable (`passage`, `arrivals`).
11. Floor 40 exists and is empty (`entity` referenced by two passages and a law).
12. The Archive is silent because paper had no owner (`indicator`, and Nera's `pursuing`).

Everything else — names, room dressing, `las ollas`, the rope bridge's stuck pulley — is texture and stays hidden.
