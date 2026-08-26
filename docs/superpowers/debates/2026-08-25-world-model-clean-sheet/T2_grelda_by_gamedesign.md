# Las Casas de Grelda — world_model/2 encoding (detailed tier)

## 1. Encoding

```jsonc
{
  "world_model": "2",
  "world": { "name": "Grelda",
    "premise": "Una ciudad de veinte mil personas y tres mil casas, y las casas están vivas: comen, crecen, se encogen y deciden quién duerme adentro. Ochocientas personas que ninguna casa aceptó nunca duermen bajo lona en el centro de una ciudad llena de cuartos vacíos.",
    "mood": ["doméstico", "cálido", "humor de vecindario"] },

  "excluded": [
    "Las casas no hablan con palabras. No tienen diálogo, voz interior ni intención explicada.",
    "No hay magia, hechizos ni magos. Hay oficio, oído, costumbre y biología rara.",
    "No hay forma de forzar un trato. Todo intento fracasa y el fracaso es la escena.",
    "No hay casas malignas ni casas con plan. El daño que causan es indiferencia, no crueldad.",
    "Una casa cerrada no se reabre nunca, por ningún medio. Que Tomás saliera es la única anomalía y debe seguir siendo única.",
    "No hay solución técnica al problema de los Sin Trato. La única vía es convencer casa por casa."
  ],

  "layers": [ { "name": "Grelda", "default": true } ],

  "vocabulary": {
    "media": [ { "name": "aire de valle templado", "affords": [ { "to": "sight", "degree": "full" } ] } ],
    "movements": [ { "name": "caminar", "pace_class": "steady" },
                   { "name": "subir la cuesta", "pace_class": "slow" } ],
    "channels": [
      { "name": "el suelo", "descriptor": "golpes y respuesta transmitidos por el subsuelo",
        "emitted_by": "cualquier casa, y cualquiera con una vara de tratante",
        "received_by": "toda casa del mismo subsuelo",
        "latency_class": "seasonal", "reach": "la ciudad entera, decayendo por barrio",
        "decay": "no se borra", "conceals": "identity",
        "note": "dos a seis semanas para cruzar un barrio, meses de extremo a extremo — la ventana en la que todavía nadie sabe lo que hiciste" },
      { "name": "sight" }, { "name": "sound" },
      { "name": "la plazoleta", "descriptor": "el mercado de grano tres veces por semana",
        "emitted_by": "cualquiera hablando en el mercado", "received_by": "el barrio",
        "latency_class": "days", "reach": "Cuesta Menor", "decay": "slow", "conceals": "none" }
    ],
    "conditions": [
      { "name": "sin oído de suelo",
        "alters": [ { "channel": "el suelo", "effect": "hinder", "class": "total" } ] },
      { "name": "sin trato",
        "alters": [ { "act": "dormir bajo techo", "effect": "hinder", "class": "total" } ] }
    ],
    "substances": [
      { "name": "grano", "note": "lo que reparte la Junta y lo único que se mide" },
      { "name": "calor de fuego", "note": "gratis, por eso no se discute" },
      { "name": "ruido de gente", "note": "lo que más les gusta; no se raciona ni se finge" }
    ]
  },

  "law": [
    { "name": "una casa no se fuerza", "enforced_by": "physics",
      "stated": "No hay dinero, ley, herramienta ni violencia que consiga un trato.",
      "forbids": { "subject": "cualquier entidad con agencia", "act": "obtener un trato sin que la casa lo dé" } },
    { "name": "cuarenta días", "enforced_by": "physics",
      "stated": "Cuarenta días sin comer y una casa se cierra entera, para siempre, con lo que haya adentro." },
    { "name": "crecen y se encogen sin aviso", "enforced_by": "physics",
      "stated": "Una casa contenta suma un cuarto; una incómoda cierra uno y se lo reabsorbe. Ninguna de las dos cosas se anuncia." },
    { "name": "la reputación es municipal", "enforced_by": "physics",
      "stated": "Las casas se cuentan por el subsuelo cómo te portaste. Lento, pero no se detiene y no se borra." },
    { "name": "dos noches no", "enforced_by": "physics",
      "stated": "La primera noche sin trato es cortesía y la casa la tolera; la segunda la vive como intrusión y cierra puertas por dentro." },
    { "name": "el esqueje no viaja", "enforced_by": "physics",
      "stated": "Un esqueje solo prende cerca de su cepa. Trasplantar mata el esqueje y enferma a la madre." },
    { "name": "licencia para plantar", "enforced_by": "office", "binds": ["los Plantadores"],
      "stated": "No se planta sin autorización de la Junta.",
      "precedent": "las-plantaciones-de-los-barrios-altos" },
    { "name": "contrabando de esquejes", "enforced_by": "office", "binds": [],
      "stated": "Sacar un esqueje de Grelda es el delito más grave del código, y se castiga con expulsión — perder el trato y no conseguir otro en ningún lado." },
    { "name": "el cuenco", "enforced_by": "persons", "binds": [],
      "stated": "Se deja el cuenco de grano en el umbral aunque la casa esté servida. Una casa nota si falta." },
    { "name": "tres inviernos", "enforced_by": "persons", "binds": [],
      "stated": "No se le pone nombre a una casa hasta que te aguantó tres inviernos." },
    { "name": "no se elogia otra casa", "enforced_by": "persons", "binds": [],
      "stated": "No se elogia otra casa delante de la propia." },
    { "name": "no se pregunta", "enforced_by": "persons", "binds": [],
      "stated": "No se le pregunta a un Sin Trato por qué no lo aceptan." },
    { "name": "no en voz alta", "enforced_by": "persons", "binds": [],
      "stated": "No se habla de las Casas Viejas en voz alta cerca del anillo interior. Nadie sabe justificarlo y todos lo cumplen." }
  ],

  "entities": [
    { "name": "Grelda", "facets": ["extent"], "extent_class": "vast",
      "medium": "aire de valle templado", "tension": "normal" },
    { "name": "anillo interior", "facets": ["extent"], "within": "Grelda", "extent_class": "large",
      "medium": "aire de valle templado", "tension": "tense" },
    { "name": "anillo medio", "facets": ["extent"], "within": "Grelda", "extent_class": "large",
      "medium": "aire de valle templado", "tension": "normal" },
    { "name": "anillo bajo", "facets": ["extent"], "within": "Grelda", "extent_class": "large",
      "medium": "aire de valle templado", "tension": "tense" },
    { "name": "Cuesta Menor", "facets": ["extent"], "within": "anillo medio", "extent_class": "medium",
      "medium": "aire de valle templado", "tension": "calm",
      "seen_as": "calles que suben, escaleras de piedra gastada, mucho ruido de chicos" },
    { "name": "plaza mayor", "facets": ["extent"], "within": "anillo interior", "extent_class": "medium",
      "medium": "aire de valle templado", "tension": "tense",
      "seen_as": "carpas en el centro exacto de la ciudad" },
    { "name": "plazoleta de los Cuencos", "facets": ["extent"], "within": "Cuesta Menor",
      "extent_class": "small", "medium": "aire de valle templado", "tension": "normal" },
    { "name": "oficina de la Junta en Cuesta Menor", "facets": ["extent", "matter"],
      "within": "Cuesta Menor", "extent_class": "intimate", "bulk_class": "immense", "integrity": "sound",
      "seen_as": "el único edificio muerto del barrio: piedra común, frío" },
    { "name": "el muro alto", "facets": ["matter", "passage"], "within": "Cuesta Menor",
      "connects": ["Cuesta Menor", "los campos de grano"], "bulk_class": "immense", "integrity": "sound",
      "obstructs": [ { "movement": "caminar" } ], "admits": [] },
    { "name": "los campos de grano", "facets": ["extent", "holding"], "within": "Grelda",
      "extent_class": "vast", "medium": "aire de valle templado", "tension": "calm",
      "capacity_class": "ample", "holds": [ { "substance": "grano", "abundance": "adequate" } ] },

    { "name": "la Cuarenta y Uno", "facets": ["extent", "matter", "agency", "demand"],
      "within": "Cuesta Menor", "extent_class": "small", "medium": "aire de valle templado",
      "tension": "calm", "bulk_class": "immense", "integrity": "sound",
      "seen_as": "una casa vieja y tolerante con el taller en el bajo",
      "demands": [ { "substance": "grano", "rate_class": "daily", "supplied_by": "draw",
                     "unmet": { "effect": "se cierra entera y para siempre", "onset_class": "forty days" } },
                   { "substance": "ruido de gente", "rate_class": "daily", "supplied_by": "emission",
                     "unmet": { "effect": "se pone huraña y deja de aceptar gente", "onset_class": "seasons" } } ],
      "disposition": [ { "trait": "tolerante", "strength": "strong",
                         "manner": "aguanta clientes entrando y saliendo todo el día" } ],
      "doing": "dejando la puerta del bajo abierta desde antes del alba" },

    { "name": "la Ochenta y Tres", "facets": ["extent", "matter", "agency", "demand"],
      "within": "Cuesta Menor", "extent_class": "medium", "medium": "aire de valle templado",
      "tension": "tense", "bulk_class": "immense", "integrity": "sound",
      "seen_as": "tres pisos pegados al muro, y un cuarto piso que aparece y desaparece",
      "note_numbers_read_by_players": "200 años; 140 solicitudes rechazadas; 60 años sin aceptar a nadie",
      "demands": [ { "substance": "grano", "rate_class": "daily", "supplied_by": "draw",
                     "unmet": { "effect": "cierre definitivo con contenido", "onset_class": "forty days" } },
                   { "substance": "ruido de gente", "rate_class": "daily", "supplied_by": "emission",
                     "unmet": { "effect": "huraña", "onset_class": "seasons" } } ],
      "disposition": [ { "trait": "cerrada", "strength": "defining",
                         "manner": "no abre, y no da motivo" } ],
      "doing": "golpeando el suelo todas las noches con el mismo ritmo, cada vez más temprano",
      "hiding": "Por qué abrió la puerta a un aprendiz que iba caminando, y qué pasó acá hace sesenta años." },

    { "name": "las casas de Cuesta Menor", "facets": ["extent", "matter", "agency", "demand", "magnitude"],
      "within": "Cuesta Menor", "magnitude_class": "many", "extent_class": "small",
      "medium": "aire de valle templado", "tension": "calm", "bulk_class": "immense", "integrity": "sound",
      "seen_as": "doscientas casas de sesenta a noventa años, de carácter tranquilo",
      "demands": [ { "substance": "grano", "rate_class": "daily", "supplied_by": "draw",
                     "unmet": { "effect": "cierre", "onset_class": "forty days" } },
                   { "substance": "ruido de gente", "rate_class": "daily", "supplied_by": "emission",
                     "unmet": { "effect": "huraña", "onset_class": "seasons" } } ] },

    { "name": "las Casas Viejas", "facets": ["extent", "matter", "agency", "demand", "magnitude", "collective"],
      "within": "anillo interior", "magnitude_class": "eleven", "extent_class": "large",
      "medium": "aire de valle templado", "tension": "frantic", "bulk_class": "immense", "integrity": "sound",
      "seen_as": "once edificios de más de trescientos años donde nadie entra",
      "legibility": "marked", "interest": "que las jóvenes les hagan caso",
      "vulnerability": "reciben ración completa por costumbre y por algo parecido al miedo",
      "demands": [ { "substance": "grano", "rate_class": "daily", "supplied_by": "draw",
                     "unmet": { "effect": "cierre", "onset_class": "forty days" } } ],
      "doing": "no aceptando a nadie desde hace cuatro generaciones",
      "hiding": "Qué son a esta altura, y por qué el barrio entero se queda quieto un día cuando una golpea el suelo." },

    { "name": "el Gremio de Tratantes", "facets": ["agency", "collective"],
      "within": "Grelda", "legibility": "marked",
      "seen_as": "unos ciento veinte profesionales con vara propia",
      "interest": "conservar el monopolio de la negociación",
      "vulnerability": "buena parte de sus técnicas son teatro; si alguien demostrara que el oficio se aprende en un año, el gremio se cae",
      "doing": "tomando examen de entrada a la camada nueva" },

    { "name": "la Junta de Alimento", "facets": ["agency", "collective", "holding"],
      "within": "anillo interior", "legibility": "marked",
      "seen_as": "ocho funcionarios y un registro",
      "capacity_class": "ample", "holds": [ { "substance": "grano", "abundance": "adequate" } ],
      "interest": "estabilidad presupuestaria y ninguna investigación",
      "vulnerability": "doce años de recorte al anillo bajo y treinta y una casas cerradas, y el registro que lo prueba lo llevan ellos",
      "doing": "repartiendo la ración del mes entre tres mil casas" },

    { "name": "los Sin Trato", "facets": ["agency", "collective", "magnitude"],
      "within": "plaza mayor", "magnitude_class": "many",
      "seen_as": "entre setecientas y novecientas personas bajo lona, con tres voceros rotativos",
      "legibility": "marked", "interest": "que se reconozca que hay miles de cuartos vacíos y el problema es de criterio, no de oferta",
      "vulnerability": "su única estrategia posible es la ocupación, y una casa que no te aceptó no te va a dejar salir",
      "conditions": ["sin trato"],
      "doing": "votando en asamblea una propuesta que va a salir que sí" },

    { "name": "los Plantadores", "facets": ["agency", "collective"], "within": "Grelda",
      "legibility": "concealed", "seen_as": "oficio pequeño y semilegal",
      "interest": "que nadie entienda la biología del asunto salvo ellos",
      "vulnerability": "lo que cultivan por lo bajo no lo controla nadie, ellos incluidos" },

    { "name": "Ordo Bes", "facets": ["matter", "agency"], "within": "Cuesta Menor",
      "seen_as": "un hombre de cincuenta y dos años, antipático, con la vara siempre encima",
      "capability": { "moves_by": ["caminar", "subir la cuesta"], "carry_class": "moderate" },
      "senses": { "el suelo": "absent", "sight": "acute" },
      "conditions": ["sin oído de suelo"],
      "disposition": [ { "trait": "impaciente", "strength": "defining",
                         "manner": "no tiene paciencia con clientes que quieren que la casa los quiera" } ],
      "doing": "prohibiéndote entrar a la Ochenta y Tres sin explicar por qué",
      "pursuing": [ { "horizon": "long_standing", "toward": "jubilarse dejando un aprendiz que sirva de verdad" } ],
      "hiding": "Perdió el oído del suelo hace cuatro años y sus resultados no bajaron, y eso lo tiene aterrado por lo que implica sobre todo lo que hizo antes." },

    { "name": "Perla Anís", "facets": ["matter", "agency"], "within": "Cuesta Menor",
      "seen_as": "una funcionaria de treinta y ocho años, cordial, con una carpeta",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "el suelo": "absent" },
      "disposition": [ { "trait": "cordial", "strength": "strong", "manner": "pregunta amablemente y anota todo" } ],
      "doing": "subiendo la Cuesta con preguntas sobre una casa que empezó a aceptar gente",
      "pursuing": [ { "horizon": "long_standing", "toward": "un ascenso al anillo interior" } ],
      "hiding": "Lleva la cuenta exacta —treinta y una— de las casas cerradas desde su recorte, en un cuaderno personal que no tiene ninguna razón administrativa para existir." },

    { "name": "Tomás", "facets": ["matter", "agency"], "within": "plaza mayor",
      "seen_as": "un hombre tranquilo de treinta y cuatro años, once en la carpa cuarenta y siete",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "el suelo": "faint" }, "conditions": ["sin trato"],
      "disposition": [ { "trait": "servicial", "strength": "strong", "manner": "ayuda antes de que le pidan" } ],
      "doing": "acercándose a la Cuesta cada tanto y no subiendo nunca",
      "pursuing": [ { "horizon": "long_standing", "toward": "una explicación, más que un techo" } ],
      "hiding": "A los veintitrés estuvo adentro de una casa cuando se cerró, con otras cuatro personas, y salió. Nunca contó cómo. Las casas lo saben." },

    { "name": "Vela Roncal", "facets": ["matter", "agency"], "within": "anillo bajo",
      "seen_as": "una plantadora de sesenta años con las manos siempre tibias",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "el suelo": "normal" },
      "disposition": [ { "trait": "reservada", "strength": "defining", "manner": "no comparte casi nada" } ],
      "doing": "manteniendo tibios unos esquejes que crecen el doble de rápido",
      "pursuing": [ { "horizon": "long_standing", "toward": "una cepa que aguante el frío del norte" } ],
      "hiding": "Sus últimos tres esquejes nacieron mal, y uno ya no está en su vivero." },

    { "name": "Halma Ruiz", "facets": ["matter", "agency"], "within": "plaza mayor",
      "seen_as": "una mujer de veintinueve años que nació en la plaza",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "el suelo": "absent" }, "conditions": ["sin trato"],
      "disposition": [ { "trait": "frontal", "strength": "defining", "manner": "propone lo que sabe que es suicidio" } ],
      "doing": "preparando la propuesta de ocupación de una Casa Vieja",
      "pursuing": [ { "horizon": "imminent", "toward": "que la asamblea vote la ocupación" } ],
      "hiding": "Sabe que es suicidio y no tiene otra carta." },

    { "name": "la vara de tratante", "facets": ["matter"], "within": "Ordo Bes",
      "bulk_class": "slight", "integrity": "sound", "size_class": "small",
      "seen_as": "madera de casa muerta, tallada por su dueño al recibirse",
      "confers": [ { "channel": "el suelo", "as": "emitter" } ],
      "note": "el único material que las casas escuchan con claridad; no se prestan" },
    { "name": "los cuencos del umbral", "facets": ["matter", "magnitude"], "within": "Grelda",
      "magnitude_class": "three thousand", "bulk_class": "negligible", "integrity": "sound",
      "seen_as": "un puñado de grano diario en cada puerta de la ciudad" },
    { "name": "el registro de la Junta", "facets": ["matter", "record"], "within": "anillo interior",
      "bulk_class": "moderate", "integrity": "sound",
      "asserts": [ { "claim": "qué come cada casa, cuánto y desde cuándo", "accurate": true } ],
      "access": { "via": "consulta pública", "delay_class": "none", "held_by": "la Junta de Alimento" },
      "authority": "la Junta de Alimento",
      "note": "cruzado con las fechas de cierre es un documento incómodo, y es público" },
    { "name": "el registro del Gremio", "facets": ["matter", "record"], "within": "Cuesta Menor",
      "bulk_class": "moderate", "integrity": "worn",
      "asserts": [ { "claim": "ciento cuarenta solicitudes rechazadas por la Ochenta y Tres", "accurate": true },
                   { "claim": "nada ocurrió durante dieciocho meses hace sesenta años", "accurate": false,
                     "plausible_because": "no hay ninguna entrada en esas fechas" } ],
      "access": { "via": "ser del Gremio", "delay_class": "none", "held_by": "el Gremio de Tratantes" },
      "authority": "el Gremio de Tratantes" },
    { "name": "los esquejes mal nacidos", "facets": ["matter", "agency", "demand", "magnitude"],
      "within": "anillo bajo", "magnitude_class": "three", "bulk_class": "slight", "integrity": "sound",
      "seen_as": "tallos de medio metro que crecen el doble de rápido",
      "demands": [ { "substance": "grano", "rate_class": "daily", "supplied_by": "draw",
                     "unmet": { "effect": "muere", "onset_class": "seasons" } } ],
      "doing": "comiendo más de lo que le corresponde a su tamaño" }
  ],

  "offices": [
    { "name": "Tratante recibido", "held_by": "Ordo Bes", "of": "el Gremio de Tratantes",
      "confers": [ { "act": "negociar un trato por encargo y cobrar comisión" },
                   { "act": "tallar y portar vara" } ],
      "succeeds_by": "examen de entrada del Gremio" },
    { "name": "Aprendiz de tratante", "held_by": "el personaje del usuario", "of": "el Gremio de Tratantes",
      "confers": [ { "act": "golpear el suelo con la vara y esperar" } ],
      "succeeds_by": "tres años con un tratante recibido" },
    { "name": "Funcionaria de reparto", "held_by": "Perla Anís", "of": "la Junta de Alimento",
      "confers": [ { "act": "fijar la ración de una casa" } ], "succeeds_by": "designación" },
    { "name": "Vocera de la plaza", "held_by": "Halma Ruiz", "of": "los Sin Trato",
      "confers": [ { "act": "hablar por la plaza en asamblea" } ], "succeeds_by": "rotación entre tres" }
  ],

  "standing": [
    { "from": "la Ochenta y Tres", "toward": "el personaje del usuario",
      "stance": "le abrió la puerta sin que golpeara y sigue golpeando el suelo todas las noches",
      "since": "la-puerta-abierta", "carried_by": "el suelo", "persistence": "until changed" },
    { "from": "las casas de Cuesta Menor", "toward": "cualquier inquilino",
      "stance": "reputación municipal: si te portás mal con la tuya, en un par de meses no te acepta ninguna",
      "since": null, "carried_by": "el suelo", "persistence": "no se borra" },
    { "from": "Ordo Bes", "toward": "el personaje del usuario",
      "stance": "lo quiere como sucesor y le prohíbe justo lo único que le importa",
      "since": null, "carried_by": null, "persistence": "until changed" },
    { "from": "las casas de Cuesta Menor", "toward": "Tomás",
      "stance": "no lo aceptan, ni una sola vez en toda su vida", "since": null,
      "carried_by": "el suelo", "persistence": "no se borra" }
  ],

  "opposition": [
    { "between": ["los Sin Trato", "las Casas Viejas"],
      "incompatible": "Hay miles de cuartos vacíos y ninguna casa que los ceda; la ocupación es la única estrategia y no funciona.",
      "stakes": "Ochocientas personas siguen bajo lona en el centro de la ciudad." },
    { "between": ["el personaje del usuario", "Ordo Bes"],
      "incompatible": "No se puede aprender a escuchar de verdad y obedecer la prohibición de entrar.",
      "stakes": "Lo que el Gremio vende, y si el oficio existe." },
    { "between": ["la Junta de Alimento", "el anillo bajo"],
      "incompatible": "La ración no alcanza para las tres mil casas y la que se recorta se cierra.",
      "stakes": "Treinta y una casas cerradas y contando." }
  ],

  "processes": [
    { "name": "crecer", "acts_on": "cualquier casa contenta", "direction": "grow",
      "rate_class": "generational", "terminus": null,
      "note": "suma un cuarto, agranda un pasillo, sube medio piso en una década; no se anuncia" },
    { "name": "encogerse", "acts_on": "cualquier casa incómoda", "direction": "degrade",
      "rate_class": "slow", "terminus": null,
      "note": "cierra un cuarto y se lo reabsorbe" },
    { "name": "madurar un esqueje", "acts_on": "cualquier esqueje", "direction": "grow",
      "rate_class": "generational", "terminus": "habitable entre los cuarenta y los sesenta años" }
  ],

  "cycles": [
    { "name": "el cuenco", "period_class": "daily", "starts_in_phase": "umbral",
      "phases": [ { "name": "umbral", "changes": [ { "substance": "grano", "to": "cada casa", "becomes": "cortesía entregada" } ] } ] },
    { "name": "mercado de grano", "period_class": "weekly", "starts_in_phase": "mercado",
      "phases": [ { "name": "mercado", "changes": [ { "tension_of": "plazoleta de los Cuencos", "becomes": "normal" } ] },
                  { "name": "vacío", "changes": [ { "tension_of": "plazoleta de los Cuencos", "becomes": "calm" } ] } ] },
    { "name": "el reparto de la cosecha", "period_class": "annual", "starts_in_phase": "discusión",
      "phases": [ { "name": "discusión", "changes": [] },
                  { "name": "misma proporción de siempre", "changes": [] } ] }
  ],

  "accumulators": [
    { "name": "días sin comer", "per": "each entity with demand", "starts_at": "none",
      "stated": "Cuántos días lleva una casa sin recibir grano.",
      "raised_by": [ { "event": "un día de ración no entregada" } ],
      "thresholds": [ { "at": "moderate", "then": "se pone huraña y deja de aceptar gente" },
                      { "at": "high", "then": "cierra puertas por dentro" },
                      { "at": "cuarenta días", "then": "se cierra entera y para siempre, con lo que haya adentro", "irreversible": true } ] },
    { "name": "noches sin trato", "per": "each pair of (agency, casa)", "starts_at": "none",
      "stated": "Cuántas noches seguidas durmió alguien en una casa que no le hizo trato.",
      "raised_by": [ { "event": "una noche dormida sin trato" } ],
      "thresholds": [ { "at": "una", "then": "la casa lo tolera como cortesía" },
                      { "at": "dos", "then": "lo vive como intrusión y cierra puertas por dentro" } ] },
    { "name": "casas cerradas desde el recorte", "per": "world", "starts_at": "moderate",
      "stated": "Cuántas casas del anillo bajo se cerraron desde que se recortó la ración. Van treinta y una.",
      "raised_by": [ { "event": "un cierre en el anillo bajo" } ],
      "thresholds": [ { "at": "high", "then": "el registro de la Junta se vuelve imposible de no cruzar" } ] },
    { "name": "el ritmo de la Sorda", "per": "each entity with agency", "starts_at": "low",
      "stated": "Cuánto se adelanta cada noche el golpeteo de la Ochenta y Tres.",
      "raised_by": [ { "event": "cada noche que golpea más temprano" } ],
      "thresholds": [ { "at": "high", "then": "el barrio deja de poder ignorarlo" } ] }
  ],

  "indicators": [
    { "of": "la predisposición de una casa a aceptar a alguien",
      "shows_as": ["tibieza al tacto", "paredes elásticas", "olor a grano cocido", "una puerta que se abre sola",
                   "familias grandes y fuego constante la predisponen bien", "silencio y mudanzas la predisponen mal"],
      "read_by": { "channel": "el suelo", "requires": { "office": "Tratante recibido", "thing": "la vara de tratante" } },
      "reliability_class": "poor",
      "note": "no es una fórmula, y los tratantes que juran tenerla son charlatanes" },
    { "of": "cuánto come realmente una casa",
      "shows_as": ["tibieza mayor de la que le corresponde por su ración"],
      "read_by": { "channel": "sight", "requires": {} }, "reliability_class": "moderate" },
    { "of": "si un tratante todavía oye el suelo",
      "shows_as": ["nada: sus resultados no bajan"],
      "read_by": { "channel": "el suelo", "requires": { "office": "Tratante recibido" } },
      "reliability_class": "none" }
  ],

  "traces": [
    { "of": "un cuarto reabsorbido", "leaves": "una pared donde estaba la habitación de invitados", "ages": "no se borra" },
    { "of": "una casa cerrada", "leaves": "puertas, ventanas y chimeneas selladas, y lo que quedó adentro", "ages": "no se borra" },
    { "of": "un trato negociado", "leaves": "un asiento en el registro del Gremio con el resultado", "ages": "no se borra" }
  ],

  "epochs": [
    { "name": "antes del recorte",
      "differed": [ { "topic": "substance", "subject": "grano", "then": "ración completa también en el anillo bajo" } ],
      "surviving_traces": ["treinta y una casas cerradas", "el cuaderno de Perla Anís"] },
    { "name": "hace sesenta años",
      "differed": [ { "topic": "entity", "subject": "la Ochenta y Tres", "then": "aceptaba gente" },
                    { "topic": "record", "subject": "el registro del Gremio", "then": "tenía entradas" } ],
      "surviving_traces": ["un hueco de dieciocho meses sin ninguna entrada", "la ración mínima que empezó en esas fechas"] },
    { "name": "las plantaciones de los barrios altos",
      "differed": [ { "topic": "law", "subject": "licencia para plantar", "then": "se ignoraba abiertamente" } ],
      "surviving_traces": ["media docena de barrios altos que nadie discute"] }
  ],

  "history": [
    { "name": "las-plantaciones-de-los-barrios-altos", "standing": "occurred",
      "what_happened": "La mitad de los barrios altos plantó sin autorización de la Junta y ya nadie lo discute.",
      "where": "anillo medio", "who": ["los Plantadores"],
      "knowledge": [ { "holder": "la Junta de Alimento", "channel": "sight", "path": "public",
                       "believes": "Ocurrió y no conviene reabrirlo." } ] },
    { "name": "el-cierre-con-Tomas-adentro", "standing": "disputed",
      "what_happened": "Una casa se cerró con cinco personas adentro y Tomás salió.",
      "where": "anillo bajo", "who": ["Tomás"],
      "knowledge": [
        { "holder": "Tomás", "channel": "sight", "path": "direct",
          "believes": "Sabe exactamente cómo salió y no lo cuenta." },
        { "holder": "las casas de Cuesta Menor", "channel": "el suelo", "path": "told",
          "believes": "Salió de una casa cerrada, y por eso no se lo acepta en ningún lado." },
        { "holder": "el Gremio de Tratantes", "channel": "sound", "path": "rumor",
          "believes": "Es una historia de la plaza y su historial es mala suerte estadística.",
          "accurate": false, "plausible_because": "nadie cruzó nunca su historial con la fecha del cierre" } ] },
    { "name": "el-recorte-del-anillo-bajo", "standing": "occurred",
      "what_happened": "Hace doce años se redujo la ración del anillo bajo. Desde entonces se cerraron treinta y una casas.",
      "where": "anillo bajo", "who": ["Perla Anís", "la Junta de Alimento"],
      "knowledge": [
        { "holder": "Perla Anís", "channel": "sight", "path": "direct",
          "believes": "Es la autora material y lleva la cuenta exacta." },
        { "holder": "los Sin Trato", "channel": "sound", "path": "rumor",
          "believes": "La Junta vacía barrios a propósito.", "accurate": false,
          "plausible_because": "el registro es público y nadie se molestó en cruzarlo" } ] },
    { "name": "la-puerta-abierta", "standing": "occurred",
      "what_happened": "La Ochenta y Tres abrió la puerta a un aprendiz que pasaba caminando, sin golpe y sin negociación, después de sesenta años y ciento cuarenta rechazos.",
      "where": "Cuesta Menor", "who": ["la Ochenta y Tres", "el personaje del usuario"],
      "knowledge": [
        { "holder": "Ordo Bes", "channel": "sight", "path": "direct",
          "believes": "Sabe algo sobre por qué y no lo dice." },
        { "holder": "Perla Anís", "channel": "sound", "path": "told",
          "believes": "Una casa vieja que empieza a aceptar altera el reparto de grano y sugiere que decidía algo que nadie le preguntó." },
        { "holder": "los Sin Trato", "channel": "la plazoleta", "path": "overheard",
          "believes": "La casa más cerrada de Grelda le abrió la puerta a alguien que ya tenía dónde dormir." } ] }
  ],

  "arrivals": [
    { "premise": "Sos aprendiz de tratante, veinticuatro años, tres con Ordo Bes. Sabés golpear el suelo y sabés esperar; no sabés escuchar todavía. Anteayer la Sorda te abrió la puerta.",
      "seen_as": "alguien joven con una vara demasiado nueva", "place": "Cuesta Menor",
      "capability": { "moves_by": ["caminar", "subir la cuesta"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "el suelo": "faint" } },
    { "premise": "Naciste en la plaza mayor y nunca dormiste bajo techo. Subís la Cuesta por primera vez porque se corrió que una Casa Vieja aceptó a alguien.",
      "seen_as": "alguien con la ropa gastada de once años de lona", "place": "plaza mayor",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "el suelo": "absent" } }
  ]
}
```

## 2. Facet lists, and where I hesitated

| Entity | Facets | Hesitation |
|---|---|---|
| Grelda, the three rings, Cuesta Menor, plaza mayor, plazoleta | `extent` | none |
| los campos de grano | `extent + holding` | none — `holds` a substance |
| oficina de la Junta | `extent + matter` | none; the brief's "único edificio muerto" is exactly the absence of `agency + demand` |
| el muro alto | `matter + passage` | **yes** — a wall is defined by what it *doesn't* join. Encoding a barrier as `passage` with `admits: []` reads like a door that is always shut, losing "this is a wall" |
| la Cuarenta y Uno, la Ochenta y Tres | `extent + matter + agency + demand` | none. This is v2's headline win — v1 could not hold it |
| las casas de Cuesta Menor | `extent + matter + agency + demand + magnitude` | **yes** — five facets on one row; and does `magnitude` distribute over `demand`? Are 200 houses one demand or 200? |
| las Casas Viejas | + `collective` | **yes, badly** — see A3 |
| Gremio, Junta, Plantadores | `agency + collective` (+`holding` for Junta) | none |
| los Sin Trato | `agency + collective + magnitude` | **yes** — see A3 |
| Ordo, Perla, Tomás, Vela, Halma | `matter + agency` | none |
| la vara de tratante | `matter` | **yes** — it `confers` a channel, and `confers` belongs to no facet |
| los cuencos del umbral | `matter + magnitude` | none |
| both registros | `matter + record` | none — a clean v2 win |
| los esquejes mal nacidos | `matter + agency + demand + magnitude` | **yes** — are they `agency`? They "don't respond to the ground", which is a decision or a defect |

## 3. Breaks

**B1 — An entity that changes its own extent. (i) inexpressible.** *"Una casa contenta suma un cuarto… una incómoda cierra un cuarto y se lo reabsorbe"* (§2). `extent_class` is a static key; `processes[]` can `act_on` an entity with `direction: grow`, but nothing says the direction is a *function of that entity's own satisfaction*. The Ochenta y Tres's *"cuarto piso que aparece y desaparece"* is prose in `seen_as`. **v1 had this break; v2 relocated it** — from "places are static containers" to "extent is a static key".

**B2 — Closure takes the contents. (ii) inert prose.** *"se cierra con lo que haya adentro"* (§2) and *"es la causa de muerte más común de Grelda después de la vejez"*. I wrote it in `unmet.effect` and an accumulator `then`. The cascade to everything `within` is not expressible. **Not fixed from v1** — same break, new location.

**B3 — Reputation as a slow-travelling standing. (iii) wrong-shaped.** *"la reputación de un inquilino es municipal. La transmisión es lenta pero no se detiene y no se borra"* (Rule 4), with the crucial *"hay una ventana durante la cual todavía nadie sabe lo que hiciste"* (§2). I used `standing[].carried_by: "el suelo"`, which is the schema's only hook, but a standing has no *position in transit*: nothing expresses that the house at the top of the Cuesta already knows and the one at the bottom does not. The window — the actual game — is inexpressible.

**B4 — A gap in a record. (i) inexpressible.** *"un hueco de dieciocho meses… sin ninguna entrada"* (§8). `asserts[]` holds claims; the absence of claims for a period is not a claim. I forged it as an `accurate: false` assertion that "nothing happened", which is my invention, not the brief's.

**B5 — A skill that may be theatre. (iii) wrong-shaped.** The Gremio's *"buena parte de sus técnicas son teatro"* and Ordo's *"sus resultados no bajaron"* (§5, §7). I encoded it as an `indicator` with `reliability_class: "none"`, which says the *world* is unreadable. The brief says something sharper: the **office** confers an ability that may not exist. `offices[].confers` cannot be conditional or fictitious.

**B6 — `confers` has no facet.** The rule change says things and offices may `confer`, but no facet in the table grants it, so `la vara de tratante` carries a key its facet list does not license. Same for `supports[]`, `seen_as`, `capability`, `senses`, `conditions[]` and `within` — all used in the schema's own example, none granted by any facet.

**B7 — Numbers.** *"Ciento cuarenta solicitudes rechazadas"*, *"treinta y una casas cerradas"*, *"cuarenta días"* are all read aloud by characters, so v2 permits them — but I could not tell whether `"at": "cuarenta días"` in a threshold ladder is a player-read figure (legal) or an engine-computed one (forbidden). I wrote it and flagged it.

## 4. Ambiguity report

**A1 — Containment: `within` (extent) vs `holds[]` (holding).** `extent` grants "things can be within it"; `holding` grants "it contains substances **or entities**". A person inside a house is defensibly either. I used `within` for entities and `holds` only for substances. Disambiguate: make `holding` substances-only, or delete `within` and make all containment `holds`.

**A2 — Are the 3.000 houses one `magnitude` entity or 3.000 entities?** I made "las casas de Cuesta Menor" one magnitude entity with one `demands[]` block, and promoted two named houses out of it. But the Junta's ration is *per house* and the closure counter is *per house*, so the aggregate needs to distribute. Disambiguate: state whether facet keys on a `magnitude` entity describe the group or each member.

**A3 — `collective` vs `magnitude` for a group of people.** The schema's own example puts "the Pactless" as `magnitude + agency` and "the Measure" as `agency + collective`, with no rule. Los Sin Trato are *both*: constituted of members with an interest and a vulnerability, and a crowd from which Halma and Tomás are promoted. I gave them all three. Another encoder could defensibly give them two. Disambiguate: `collective` = has an interest; `magnitude` = has uncounted members; say that they compose.

**A4 — Is a norm `law` or an `office`-enforced rule?** `enforced_by: persons | office` — *"no se planta sin autorización de la Junta"* is enforced by an office that half the city ignored eighty years ago. I chose `office`; `persons` is equally defensible. Disambiguate: `office` means a named office may sanction; `persons` means anyone may.

**A5 — Accumulator `per`.** The example shows `"per": "world"`. I needed per-entity ("días sin comer") and per-*pair* ("noches sin trato"). I invented `"each pair of (agency, casa)"`. Nothing in the contract says the scope grammar. This is the single most likely divergence between two encoders.

**A6 — A house's willingness: `indicator` or `hiding`?** *"Se sabe qué la predispone bien… pero no es una fórmula"* (§2). I made it an `indicator` of a hidden state. It is equally defensible as `hiding` on each house, or as a `standing` from house toward applicant. Three readings, all expressible.

**A7 — The knocking: channel emission, `doing`, or indicator?** *"golpea el suelo todas las noches… cada vez más temprano"*. I put it in `doing` and made its acceleration an accumulator. It is defensibly an `indicator` of the house's hidden state, or a `trace`.

**A8 — Where does a condition live?** I wrote `conditions: ["sin trato"]` on entities, following the v1 example. v2's facet table never lists `conditions` as a key of any facet.

## 5. Convergence check

**No new top-level section is needed for Grelda.** Every cluster landed somewhere, which is a real result for v2 — the living house, its diet, its records and its guilds all fit, and four of v1's breaks (records, conferral, offices, thresholds) are genuinely closed.

But the fixed-kind count is not 1. `offices`, `indicators`, `standing` and `layers` are **new fixed kinds with fixed key sets** — an office is exactly the "authority" noun v1 lacked, promoted to a section rather than to a facet. An office is an entity with `agency` that confers; a standing is a relation; an indicator is a record of signs. If facets were the answer for nouns, four of the five new sections are the same mistake at a different level: v2 traded four fixed noun-kinds for five fixed section-kinds, and my ambiguity list is dominated by *which section* a thing belongs in — A4, A6 and A7 are all "same fact, three legal homes".
