# Las Casas de Grelda — world_model/3 (re-encode)

```jsonc
{
  "world_model": "3",
  "world": { "name": "Grelda",
    "premise": "Veinte mil personas y tres mil casas, y las casas están vivas: comen todos los días, crecen, se encogen y deciden quién duerme adentro. Ochocientas personas que ninguna casa aceptó nunca duermen bajo lona en el centro de una ciudad llena de cuartos vacíos.",
    "mood": ["doméstico", "cálido", "humor de vecindario"] },

  "excluded": [
    "Las casas no hablan con palabras: no tienen diálogo, voz interior ni intención explicada. Todo lo que se sepa de una casa se sabe por lo que hace.",
    "No hay magia, hechizos, magos ni escuelas. Hay oficio, oído, costumbre y biología rara.",
    "No hay forma de forzar un trato. Todo intento fracasa y el fracaso es siempre la escena.",
    "No hay casas malignas ni casas con plan. La mayor parte del daño que causan es indiferencia.",
    "Una casa cerrada no se reabre nunca, por ningún medio. Que Tomás saliera es la única anomalía conocida y debe seguir siendo única.",
    "No hay solución técnica al problema de los Sin Trato: la única vía es convencer casa por casa."
  ],

  "vocabulary": {
    "media": [ { "name": "aire de valle templado", "affords": [ { "to": "sight", "degree": "full" } ] } ],
    "movements": [ { "name": "caminar", "pace_class": "steady" },
                   { "name": "subir la cuesta", "pace_class": "slow" } ],
    "channels": [
      { "name": "sight", "emitted_by": "cualquier entidad", "received_by": "quien esté presente",
        "latency_class": "immediate", "reach": "la misma extensión", "decay": "never", "conceals": "none" },
      { "name": "el suelo", "descriptor": "golpes y respuesta transmitidos por el subsuelo",
        "emitted_by": "cualquier casa, y cualquiera con una vara de tratante",
        "received_by": "toda casa del mismo subsuelo",
        "latency_class": "seasonal", "reach": "la ciudad entera", "decay": "never", "conceals": "identity",
        "note": "de dos a seis semanas para cruzar un barrio; meses de extremo a extremo" },
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
      { "name": "calor de fuego", "note": "gratis, y por eso no se discute" },
      { "name": "ruido de gente", "note": "lo que más les gusta; no se raciona ni se finge" }
    ]
  },

  "law": [
    { "name": "una casa no se fuerza", "enforced_by": "physics", "within": "Grelda",
      "stated": "No hay dinero, ley, herramienta ni violencia que consiga un trato. Se ha intentado de todo durante siglos.",
      "forbids": { "subject": "cualquier entidad con agencia", "act": "obtener un trato que la casa no dio" } },
    { "name": "cuarenta días", "enforced_by": "physics", "within": "Grelda",
      "stated": "Cuarenta días sin comer y una casa se cierra entera —puertas, ventanas, chimeneas— para siempre y con lo que haya adentro." },
    { "name": "crecen y se encogen sin aviso", "enforced_by": "physics", "within": "Grelda",
      "stated": "Una casa contenta suma un cuarto o sube medio piso en una década; una incómoda cierra un cuarto y se lo reabsorbe. Ninguna de las dos cosas se anuncia." },
    { "name": "la reputación es municipal", "enforced_by": "physics", "within": "Grelda",
      "stated": "Las casas se cuentan por el subsuelo cómo te portaste. Lento, pero no se detiene y no se borra." },
    { "name": "dos noches no", "enforced_by": "physics", "within": "Grelda",
      "stated": "La primera noche sin trato es cortesía y la casa la tolera; la segunda la vive como intrusión y cierra puertas por dentro." },
    { "name": "el esqueje no viaja", "enforced_by": "physics", "within": "Grelda",
      "stated": "Un esqueje solo prende cerca de su cepa. Trasplantar entre ciudades mata el esqueje y enferma a la madre." },
    { "name": "licencia para plantar", "enforced_by": "office", "within": "Grelda",
      "binds": ["los Plantadores"],
      "stated": "No se planta sin autorización de la Junta.",
      "precedent": "las-plantaciones-de-los-barrios-altos" },
    { "name": "contrabando de esquejes", "enforced_by": "office", "within": "Grelda", "binds": [],
      "stated": "Sacar un esqueje de Grelda es el delito más grave del código y se castiga con expulsión: perder el trato con tu casa y no conseguir otro en ningún lado." },
    { "name": "el cuenco", "enforced_by": "persons", "within": "Grelda", "binds": [],
      "stated": "Se deja el cuenco de grano en el umbral aunque la casa esté servida. Una casa nota si falta." },
    { "name": "tres inviernos", "enforced_by": "persons", "within": "Grelda", "binds": [],
      "stated": "No se le pone nombre a una casa hasta que te aguantó tres inviernos." },
    { "name": "no se elogia otra casa", "enforced_by": "persons", "within": "Grelda", "binds": [],
      "stated": "No se elogia otra casa delante de la propia." },
    { "name": "no se pregunta", "enforced_by": "persons", "within": "Grelda", "binds": [],
      "stated": "No se le pregunta a un Sin Trato por qué no lo aceptan." },
    { "name": "no en voz alta", "enforced_by": "persons", "within": "anillo interior", "binds": [],
      "stated": "No se habla de las Casas Viejas en voz alta cerca del anillo interior. Nadie sabe justificarlo y todos lo cumplen." }
  ],

  "entities": [
    { "name": "Grelda", "facets": ["extent"], "extent_class": "vast", "medium": "aire de valle templado",
      "tension": "normal", "seen_as": "un valle templado con campos de grano alrededor y tres anillos de casas" },
    { "name": "anillo interior", "facets": ["extent"], "within": "Grelda", "extent_class": "large",
      "medium": "aire de valle templado", "tension": "tense" },
    { "name": "anillo medio", "facets": ["extent"], "within": "Grelda", "extent_class": "large",
      "medium": "aire de valle templado", "tension": "normal" },
    { "name": "anillo bajo", "facets": ["extent"], "within": "Grelda", "extent_class": "large",
      "medium": "aire de valle templado", "tension": "tense",
      "seen_as": "el más pobre y el más nuevo, con casas jóvenes de carácter inestable" },
    { "name": "Cuesta Menor", "facets": ["extent", "holding"], "within": "anillo medio",
      "extent_class": "medium", "medium": "aire de valle templado", "tension": "calm",
      "capacity_class": "ample",
      "holds": [ { "substance": "ruido de gente", "abundance": "ample" },
                 { "substance": "calor de fuego", "abundance": "ample" } ],
      "seen_as": "calles que suben, escaleras de piedra gastada, mucho ruido de chicos" },
    { "name": "plaza mayor", "facets": ["extent"], "within": "anillo interior", "extent_class": "medium",
      "medium": "aire de valle templado", "tension": "tense",
      "seen_as": "carpas en el centro exacto de la ciudad" },
    { "name": "plazoleta de los Cuencos", "facets": ["extent"], "within": "Cuesta Menor",
      "extent_class": "small", "medium": "aire de valle templado", "tension": "normal" },
    { "name": "los campos de grano", "facets": ["extent", "holding"], "within": "Grelda",
      "extent_class": "vast", "medium": "aire de valle templado", "tension": "calm",
      "capacity_class": "ample", "holds": [ { "substance": "grano", "abundance": "adequate" } ] },
    { "name": "oficina de la Junta en Cuesta Menor", "facets": ["extent", "matter"],
      "within": "Cuesta Menor", "extent_class": "intimate", "medium": "aire de valle templado",
      "tension": "normal", "bulk_class": "immense", "integrity": "sound",
      "seen_as": "el único edificio muerto del barrio: piedra común, frío, y a todos les deprime entrar" },
    { "name": "el muro alto", "facets": ["matter", "passage"], "within": "Cuesta Menor",
      "connects": ["Cuesta Menor", "los campos de grano"], "bulk_class": "immense", "integrity": "sound",
      "admits": [], "obstructs": [ { "movement": "caminar" }, { "movement": "subir la cuesta" } ],
      "hazard_class": "none", "seen_as": "cierra la subida por arriba; del otro lado hay campo" },
    { "name": "la escalera de la Cuesta", "facets": ["matter", "passage"], "within": "Cuesta Menor",
      "connects": ["plazoleta de los Cuencos", "Cuesta Menor"], "bulk_class": "immense",
      "integrity": "worn", "admits": [ { "movement": "subir la cuesta" } ], "obstructs": [],
      "hazard_class": "none", "seen_as": "piedra gastada, y la subida entera hasta el muro" },

    { "name": "la Cuarenta y Uno", "facets": ["extent", "matter", "agency", "demand"],
      "within": "Cuesta Menor", "extent_class": "small", "medium": "aire de valle templado",
      "tension": "calm", "bulk_class": "immense", "integrity": "sound",
      "seen_as": "una casa vieja y tolerante con el taller de un tratante en el bajo",
      "demands": [
        { "substance": "grano", "rate_class": "daily", "supplied_by": "draw",
          "unmet": { "effect": "se cierra entera y para siempre, con lo que haya adentro", "onset_class": "seasons" } },
        { "substance": "ruido de gente", "rate_class": "daily", "supplied_by": "draw",
          "unmet": { "effect": "se pone huraña y deja de aceptar gente", "onset_class": "seasons" } } ],
      "disposition": [ { "trait": "tolerante", "strength": "strong",
                         "manner": "aguanta clientes entrando y saliendo todo el día" } ],
      "doing": "dejando la puerta del bajo abierta desde antes del alba",
      "pursuing": [ { "horizon": "long_standing", "toward": "más gente adentro y más ruido" } ] },

    { "name": "la Ochenta y Tres", "facets": ["extent", "matter", "agency", "demand"],
      "within": "Cuesta Menor", "extent_class": "medium", "medium": "aire de valle templado",
      "tension": "tense", "bulk_class": "immense", "integrity": "sound",
      "seen_as": "doscientos años, tres pisos pegados al muro y un cuarto piso que aparece y desaparece; en el barrio le dicen la Sorda",
      "demands": [
        { "substance": "grano", "rate_class": "daily", "supplied_by": "draw",
          "unmet": { "effect": "cierre definitivo con contenido", "onset_class": "seasons" } },
        { "substance": "ruido de gente", "rate_class": "daily", "supplied_by": "draw",
          "unmet": { "effect": "huraña", "onset_class": "seasons" } } ],
      "disposition": [ { "trait": "cerrada", "strength": "defining", "manner": "no abre, y no da motivo" } ],
      "doing": "golpeando el suelo todas las noches con el mismo ritmo, cada vez más temprano",
      "pursuing": [ { "horizon": "long_standing", "toward": "que alguien traduzca el ritmo antes de que se le acabe el tiempo" } ],
      "hiding": "Por qué abrió la puerta, y qué pasó acá hace sesenta años." },

    { "name": "las casas de Cuesta Menor", "facets": ["extent", "matter", "agency", "demand", "magnitude"],
      "within": "Cuesta Menor", "magnitude_class": "many", "extent_class": "small",
      "medium": "aire de valle templado", "tension": "calm", "bulk_class": "immense", "integrity": "sound",
      "seen_as": "unas doscientas casas de sesenta a noventa años, de carácter tranquilo",
      "demands": [
        { "substance": "grano", "rate_class": "daily", "supplied_by": "draw",
          "unmet": { "effect": "cierre definitivo con contenido", "onset_class": "seasons" } },
        { "substance": "ruido de gente", "rate_class": "daily", "supplied_by": "draw",
          "unmet": { "effect": "huraña", "onset_class": "seasons" } } ],
      "disposition": [ { "trait": "tranquilas", "strength": "strong", "manner": "aceptan familias y rechazan silencio" } ],
      "doing": "creciendo despacio porque el barrio tiene chicos",
      "pursuing": [ { "horizon": "long_standing", "toward": "seguir llenas" } ] },

    { "name": "la Casa Vieja del portón sur", "facets": ["extent", "matter", "agency", "demand"],
      "within": "anillo interior", "extent_class": "large", "medium": "aire de valle templado",
      "tension": "frantic", "bulk_class": "immense", "integrity": "sound",
      "seen_as": "más de trescientos años, y nadie sabe cuánto mide adentro porque nadie entra",
      "demands": [ { "substance": "grano", "rate_class": "daily", "supplied_by": "draw",
                     "unmet": { "effect": "cierre", "onset_class": "seasons" } } ],
      "disposition": [ { "trait": "inmóvil", "strength": "defining", "manner": "no ha abierto en cuatro generaciones" } ],
      "doing": "recibiendo ración completa por costumbre",
      "pursuing": [ { "horizon": "long_standing", "toward": "que el anillo interior siga vacío de gente" } ],
      "hiding": "Qué es a esta altura, y por qué las casas jóvenes le hacen caso." },
    { "name": "las otras Casas Viejas", "facets": ["extent", "matter", "agency", "demand", "magnitude"],
      "within": "anillo interior", "magnitude_class": "few", "extent_class": "large",
      "medium": "aire de valle templado", "tension": "frantic", "bulk_class": "immense",
      "integrity": "sound", "seen_as": "edificios de más de trescientos años donde nadie entra",
      "demands": [ { "substance": "grano", "rate_class": "daily", "supplied_by": "draw",
                     "unmet": { "effect": "cierre", "onset_class": "seasons" } } ],
      "disposition": [ { "trait": "inmóviles", "strength": "defining", "manner": "no aceptan a nadie" } ],
      "doing": "quedándose calladas",
      "pursuing": [ { "horizon": "long_standing", "toward": "que nadie entre" } ] },

    { "name": "el Gremio de Tratantes", "facets": ["agency", "collective"], "within": "Grelda",
      "legibility": "marked", "seen_as": "unos ciento veinte profesionales, cada uno con su vara",
      "interest": "conservar el monopolio de la negociación",
      "vulnerability": "buena parte de sus técnicas son teatro, y si alguien demostrara que el oficio se aprende en un año el gremio se cae",
      "disposition": [ { "trait": "corporativo", "strength": "strong", "manner": "examina antes de admitir" } ],
      "doing": "tomando examen de entrada a la camada nueva",
      "pursuing": [ { "horizon": "long_standing", "toward": "que nadie audite qué parte del oficio es real" } ],
      "hiding": "Hay tratantes que consiguen tratos sin entender qué hicieron." },
    { "name": "la Junta de Alimento", "facets": ["agency", "collective", "holding"],
      "within": "anillo interior", "legibility": "marked", "capacity_class": "ample",
      "holds": [ { "substance": "grano", "abundance": "adequate" } ],
      "seen_as": "ocho funcionarios y un registro",
      "interest": "estabilidad presupuestaria y ninguna investigación",
      "vulnerability": "doce años de recorte al anillo bajo, treinta y una casas cerradas, y el registro que lo prueba lo llevan ellos",
      "disposition": [ { "trait": "burocrática", "strength": "defining", "manner": "reparte igual todos los meses" } ],
      "doing": "repartiendo la ración del mes entre tres mil casas",
      "pursuing": [ { "horizon": "long_standing", "toward": "que nadie cruce el registro con las fechas de cierre" } ] },
    { "name": "los Sin Trato", "facets": ["agency", "collective", "magnitude"], "within": "plaza mayor",
      "magnitude_class": "many", "conditions": ["sin trato"], "legibility": "marked",
      "seen_as": "entre setecientas y novecientas personas bajo lona, con tres voceros rotativos",
      "interest": "que se reconozca que hay miles de cuartos vacíos y que el problema es de criterio, no de oferta",
      "vulnerability": "su única estrategia posible es la ocupación, y una casa que no te aceptó no te va a dejar salir",
      "disposition": [ { "trait": "organizados", "strength": "strong", "manner": "votan en asamblea antes de actuar" } ],
      "doing": "votando una propuesta que va a salir que sí",
      "pursuing": [ { "horizon": "imminent", "toward": "ocupar una Casa Vieja del anillo interior" } ] },
    { "name": "los Plantadores", "facets": ["agency", "collective"], "within": "Grelda",
      "legibility": "concealed", "seen_as": "un oficio pequeño y semilegal",
      "interest": "que nadie más entienda la biología del asunto",
      "vulnerability": "lo que cultivan por lo bajo no lo controla nadie, ellos incluidos",
      "disposition": [ { "trait": "cerrados", "strength": "defining", "manner": "no comparten casi nada" } ],
      "doing": "cultivando para la Junta y por lo bajo para quien pague",
      "pursuing": [ { "horizon": "long_standing", "toward": "una cepa que aguante otro clima" } ],
      "hiding": "Cuántos esquejes salieron del vivero sin figurar en ningún papel." },

    { "name": "Ordo Bes", "facets": ["matter", "agency"], "within": "Cuesta Menor",
      "bulk_class": "moderate", "integrity": "worn", "conditions": ["sin oído de suelo"],
      "seen_as": "un hombre de cincuenta y dos años, antipático, con la vara siempre encima",
      "capability": { "moves_by": ["caminar", "subir la cuesta"], "carry_class": "moderate" },
      "senses": { "sight": "acute", "el suelo": "absent" },
      "disposition": [ { "trait": "impaciente", "strength": "defining",
                         "manner": "no tiene paciencia con quien quiere que la casa lo quiera" } ],
      "doing": "prohibiendo entrar a la Ochenta y Tres sin explicar el motivo",
      "pursuing": [ { "horizon": "long_standing", "toward": "jubilarse dejando un aprendiz que sirva de verdad" } ],
      "hiding": "Perdió el oído del suelo hace cuatro años y sus resultados no bajaron, y eso lo tiene aterrado por lo que implica sobre todo lo que hizo antes." },
    { "name": "Perla Anís", "facets": ["matter", "agency"], "within": "Cuesta Menor",
      "bulk_class": "moderate", "integrity": "sound",
      "seen_as": "una funcionaria de treinta y ocho años, cordial, con una carpeta",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "el suelo": "absent" },
      "disposition": [ { "trait": "cordial", "strength": "strong", "manner": "pregunta amablemente y anota todo" } ],
      "doing": "subiendo la Cuesta con preguntas sobre una casa que empezó a aceptar gente",
      "pursuing": [ { "horizon": "imminent", "toward": "un ascenso al anillo interior" } ],
      "hiding": "Lleva la cuenta exacta —treinta y una— de las casas cerradas desde su recorte, en un cuaderno que no tiene ninguna razón administrativa para existir." },
    { "name": "Tomás", "facets": ["matter", "agency"], "within": "plaza mayor",
      "bulk_class": "moderate", "integrity": "sound", "conditions": ["sin trato"],
      "seen_as": "un hombre tranquilo de treinta y cuatro años, once en la carpa cuarenta y siete",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "el suelo": "faint" },
      "disposition": [ { "trait": "servicial", "strength": "strong", "manner": "ayuda antes de que le pidan" } ],
      "doing": "acercándose a la Cuesta cada tanto y no subiendo nunca",
      "pursuing": [ { "horizon": "long_standing", "toward": "una explicación, más que un techo" } ],
      "hiding": "A los veintitrés estuvo adentro de una casa cuando se cerró, con otras cuatro personas, y salió. Nunca contó cómo." },
    { "name": "Vela Roncal", "facets": ["matter", "agency"], "within": "anillo bajo",
      "bulk_class": "moderate", "integrity": "worn",
      "seen_as": "una plantadora de sesenta años con las manos siempre tibias",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "el suelo": "normal" },
      "disposition": [ { "trait": "reservada", "strength": "defining", "manner": "no comparte casi nada" } ],
      "doing": "manteniendo tibios unos esquejes que crecen el doble de rápido",
      "pursuing": [ { "horizon": "long_standing", "toward": "una cepa que aguante el frío del norte" } ],
      "hiding": "Sus últimos tres esquejes nacieron mal y uno ya no está en su vivero." },
    { "name": "Halma Ruiz", "facets": ["matter", "agency"], "within": "plaza mayor",
      "bulk_class": "moderate", "integrity": "sound", "conditions": ["sin trato"],
      "seen_as": "una mujer de veintinueve años que nació en la plaza y nunca durmió bajo techo",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "el suelo": "absent" },
      "disposition": [ { "trait": "frontal", "strength": "defining", "manner": "propone lo que sabe que es suicidio" } ],
      "doing": "preparando la propuesta de ocupación",
      "pursuing": [ { "horizon": "imminent", "toward": "que la asamblea vote la ocupación de una Casa Vieja" } ],
      "hiding": "Sabe que es suicidio y no le queda otra carta." },

    { "name": "la vara de Ordo Bes", "facets": ["matter"], "within": "Cuesta Menor",
      "bulk_class": "slight", "integrity": "worn", "size_class": "small",
      "seen_as": "madera de casa muerta, tallada por su dueño al recibirse; no se prestan",
      "confers": [ { "channel": "el suelo" } ] },
    { "name": "los cuencos del umbral", "facets": ["matter", "magnitude"], "within": "Grelda",
      "magnitude_class": "multitude", "bulk_class": "negligible", "integrity": "sound",
      "seen_as": "un puñado de grano diario en cada puerta de la ciudad" },
    { "name": "el registro de la Junta", "facets": ["matter", "record"], "within": "anillo interior",
      "bulk_class": "moderate", "integrity": "sound",
      "asserts": [ { "claim": "qué come cada casa, cuánto y desde cuándo", "accurate": true } ],
      "access": { "via": "consulta pública", "delay_class": "immediate", "held_by": "la Junta de Alimento" },
      "authority": "la Junta de Alimento",
      "seen_as": "cruzado con las fechas de cierre es un documento incómodo, y es público" },
    { "name": "el registro del Gremio", "facets": ["matter", "record"], "within": "Cuesta Menor",
      "bulk_class": "moderate", "integrity": "worn",
      "asserts": [ { "claim": "ciento cuarenta solicitudes rechazadas por la Ochenta y Tres", "accurate": true },
                   { "claim": "no ocurrió nada durante dieciocho meses hace sesenta años", "accurate": false,
                     "plausible_because": "no hay ninguna entrada en esas fechas y nadie preguntó por qué" } ],
      "access": { "via": "ser del Gremio", "delay_class": "immediate", "held_by": "el Gremio de Tratantes" },
      "authority": "el Gremio de Tratantes" },
    { "name": "el esqueje que falta", "facets": ["matter", "agency", "demand"], "within": "anillo bajo",
      "bulk_class": "slight", "integrity": "sound",
      "seen_as": "un tallo de medio metro creciendo entre dos casas, y crece el doble de rápido",
      "demands": [ { "substance": "grano", "rate_class": "daily", "supplied_by": "draw",
                     "unmet": { "effect": "muere", "onset_class": "seasons" } } ],
      "disposition": [ { "trait": "voraz", "strength": "defining", "manner": "come más de lo que le corresponde por su tamaño" } ],
      "doing": "creciendo entre dos casas que empezaron a encogerse",
      "pursuing": [ { "horizon": "long_standing", "toward": "más grano del que le toca" } ] },
    { "name": "los otros esquejes mal nacidos", "facets": ["matter", "agency", "demand", "magnitude"],
      "within": "anillo bajo", "magnitude_class": "few", "bulk_class": "slight", "integrity": "sound",
      "seen_as": "tallos que crecen rápido y no responden al suelo",
      "demands": [ { "substance": "grano", "rate_class": "daily", "supplied_by": "draw",
                     "unmet": { "effect": "mueren", "onset_class": "seasons" } } ],
      "disposition": [ { "trait": "sordos", "strength": "strong", "manner": "no responden al suelo" } ],
      "doing": "creciendo en el vivero de Vela Roncal",
      "pursuing": [ { "horizon": "long_standing", "toward": "crecer" } ] }
  ],

  "offices": [
    { "name": "Tratante recibido", "held_by": "Ordo Bes", "of": "el Gremio de Tratantes",
      "confers": [ { "act": "negociar un trato por encargo y cobrar comisión" },
                   { "act": "tallar y portar vara" } ],
      "succeeds_by": "examen de entrada del Gremio" },
    { "name": "Funcionaria de reparto", "held_by": "Perla Anís", "of": "la Junta de Alimento",
      "confers": [ { "act": "fijar la ración de una casa" } ], "succeeds_by": "designación" },
    { "name": "Vocera de la plaza", "held_by": "Halma Ruiz", "of": "los Sin Trato",
      "confers": [ { "act": "hablar por la plaza en asamblea" } ], "succeeds_by": "rotación entre tres" },
    { "name": "Plantadora autorizada", "held_by": "Vela Roncal", "of": "los Plantadores",
      "confers": [ { "act": "cultivar esquejes para la Junta" } ], "succeeds_by": "autorización de la Junta" }
  ],

  "standing": [
    { "from": "la Ochenta y Tres", "toward": "Ordo Bes",
      "stance": "le golpea el suelo todas las noches y él es el único del barrio que podría traducirlo",
      "since": null, "carried_by": "el suelo", "persistence": "until changed" },
    { "from": "las casas de Cuesta Menor", "toward": "Tomás",
      "stance": "no lo aceptan, ni una sola vez en toda su vida",
      "since": "el-cierre-con-Tomas-adentro", "carried_by": "el suelo", "persistence": "never decays" },
    { "from": "Ordo Bes", "toward": "el Gremio de Tratantes",
      "stance": "les debe el oficio y sospecha que el oficio no existe",
      "since": null, "carried_by": null, "persistence": "until changed" },
    { "from": "los Sin Trato", "toward": "la Casa Vieja del portón sur",
      "stance": "la eligieron para la ocupación porque es la que da a la plaza",
      "since": null, "carried_by": "la plazoleta", "persistence": "until changed" },
    { "from": "Perla Anís", "toward": "la Ochenta y Tres",
      "stance": "una casa vieja que empieza a aceptar gente le desordena el reparto",
      "since": "la-puerta-abierta", "carried_by": null, "persistence": "until changed" }
  ],

  "opposition": [
    { "between": ["los Sin Trato", "la Casa Vieja del portón sur"],
      "incompatible": "Hay miles de cuartos vacíos y ninguna casa que los ceda; la ocupación es la única estrategia y no funciona.",
      "stakes": "Ochocientas personas siguen bajo lona en el centro de la ciudad." },
    { "between": ["la Junta de Alimento", "las casas de Cuesta Menor"],
      "incompatible": "La ración no alcanza para tres mil casas y la que se recorta se cierra.",
      "stakes": "Treinta y una casas cerradas y contando." },
    { "between": ["Ordo Bes", "el Gremio de Tratantes"],
      "incompatible": "No puede enseñar el oficio de verdad sin mostrar que buena parte es teatro.",
      "stakes": "Si el oficio se aprende en un año, el gremio se cae." },
    { "between": ["el esqueje que falta", "las casas de Cuesta Menor"],
      "incompatible": "Lo que come el esqueje sale de la ración del sector.",
      "stakes": "Dos casas del anillo bajo ya empezaron a encogerse." } ],

  "processes": [
    { "name": "crecer", "acts_on": "toda casa contenta", "direction": "grow",
      "rate_class": "generational", "terminus": null,
      "note": "suma un cuarto, agranda un pasillo, sube medio piso en una década, y no lo anuncia" },
    { "name": "encogerse", "acts_on": "toda casa incómoda", "direction": "degrade",
      "rate_class": "slow", "terminus": null, "note": "cierra un cuarto y se lo reabsorbe" },
    { "name": "madurar un esqueje", "acts_on": "todo esqueje plantado", "direction": "grow",
      "rate_class": "generational", "terminus": "habitable" }
  ],

  "cycles": [
    { "name": "el cuenco", "period_class": "daily", "starts_in_phase": "umbral",
      "phases": [ { "name": "umbral", "changes": [ { "substance": "grano", "becomes": "cortesía entregada" } ] },
                  { "name": "día", "changes": [] } ] },
    { "name": "mercado de grano", "period_class": "weekly", "starts_in_phase": "mercado",
      "phases": [ { "name": "mercado", "changes": [ { "tension_of": "plazoleta de los Cuencos", "becomes": "normal" } ] },
                  { "name": "vacío", "changes": [ { "tension_of": "plazoleta de los Cuencos", "becomes": "calm" } ] } ] },
    { "name": "el reparto de la cosecha", "period_class": "annual", "starts_in_phase": "discusión",
      "phases": [ { "name": "discusión", "changes": [] },
                  { "name": "la misma proporción de siempre", "changes": [ { "substance": "grano", "becomes": "repartido" } ] } ] }
  ],

  "accumulators": [
    { "name": "días sin comer", "per": "each entity with demand", "starts_at": "none",
      "stated": "Cuántos días lleva una casa sin recibir grano. A los cuarenta se cierra entera, con lo que haya adentro.",
      "raised_by": [ { "event": "un día de ración no entregada" } ],
      "thresholds": [ { "at": "low", "then": "se pone huraña y deja de aceptar gente" },
                      { "at": "moderate", "then": "cierra cuartos por dentro" },
                      { "at": "high", "then": "se cierra entera y para siempre, con contenido", "irreversible": true } ] },
    { "name": "noches sin trato", "per": "each entity with agency", "starts_at": "none",
      "stated": "Cuántas noches seguidas durmió alguien en una casa que no le hizo trato.",
      "raised_by": [ { "event": "una noche dormida sin trato" } ],
      "thresholds": [ { "at": "low", "then": "la casa lo tolera como cortesía" },
                      { "at": "moderate", "then": "lo vive como intrusión y cierra puertas por dentro" } ] },
    { "name": "casas cerradas desde el recorte", "per": "world", "starts_at": "moderate",
      "stated": "Cuántas casas del anillo bajo se cerraron desde el recorte de raciones. Van treinta y una.",
      "raised_by": [ { "event": "un cierre en el anillo bajo" } ],
      "thresholds": [ { "at": "high", "then": "el registro de la Junta se vuelve imposible de no cruzar" } ] },
    { "name": "el adelanto del ritmo", "per": "each entity with agency", "starts_at": "low",
      "stated": "Cuánto se adelanta cada noche el golpeteo de la Ochenta y Tres.",
      "raised_by": [ { "event": "una noche golpeando más temprano" } ],
      "thresholds": [ { "at": "moderate", "then": "el barrio deja de poder ignorarlo" },
                      { "at": "high", "then": "el ritmo cambia por otro, distinto" } ] }
  ],

  "indicators": [
    { "of": "el hiding de la Ochenta y Tres",
      "shows_as": ["tibieza mayor de la que le corresponde por su ración",
                   "un cuarto piso que aparece y desaparece",
                   "un ritmo en el suelo que ningún tratante traduce"],
      "read_by": { "channel": "el suelo", "requires": { "office": "Tratante recibido" } },
      "reliability_class": "poor" },
    { "of": "días sin comer",
      "shows_as": ["paredes menos elásticas", "olor a grano que se apaga", "una puerta que ya no abre de par en par"],
      "read_by": { "channel": "sight", "requires": {} }, "reliability_class": "moderate" },
    { "of": "el hiding de Ordo Bes",
      "shows_as": ["nada: sus resultados no bajaron"],
      "read_by": { "channel": "el suelo", "requires": { "office": "Tratante recibido" } },
      "reliability_class": "none" },
    { "of": "el adelanto del ritmo",
      "shows_as": ["la hora a la que empieza el golpeteo cada noche"],
      "read_by": { "channel": "el suelo", "requires": {} }, "reliability_class": "good" }
  ],

  "traces": [
    { "of": "un cuarto reabsorbido", "leaves": "una pared donde estaba la habitación de invitados", "ages": "never" },
    { "of": "una casa cerrada", "leaves": "puertas, ventanas y chimeneas selladas, y lo que quedó adentro", "ages": "never" },
    { "of": "un trato negociado", "leaves": "un asiento en el registro del Gremio con el resultado", "ages": "never" }
  ],

  "epochs": [
    { "name": "antes del recorte",
      "differed": [ { "topic": "substance", "subject": "grano", "then": "ración completa también en el anillo bajo" } ],
      "surviving_traces": ["treinta y una casas cerradas", "el cuaderno de Perla Anís"] },
    { "name": "hace sesenta años",
      "differed": [ { "topic": "entity", "subject": "la Ochenta y Tres", "then": "aceptaba gente" } ],
      "surviving_traces": ["un hueco de dieciocho meses sin ninguna entrada en el registro del Gremio",
                           "la ración mínima que empezó en esas fechas"] },
    { "name": "las plantaciones de los barrios altos",
      "differed": [ { "topic": "law", "subject": "licencia para plantar", "then": "se ignoraba abiertamente" } ],
      "surviving_traces": ["media docena de barrios altos que ya nadie discute"] }
  ],

  "history": [
    { "name": "las-plantaciones-de-los-barrios-altos", "standing": "occurred",
      "what_happened": "La mitad de los barrios altos plantó sin autorización de la Junta hace ochenta años y ya nadie lo discute.",
      "where": "anillo medio", "who": ["los Plantadores"],
      "knowledge": [ { "holder": "la Junta de Alimento", "channel": "sight", "path": "public",
                       "believes": "Ocurrió, y no conviene reabrirlo." } ] },
    { "name": "el-cierre-con-Tomas-adentro", "standing": "disputed",
      "what_happened": "Una casa se cerró con cinco personas adentro y Tomás salió.",
      "where": "anillo bajo", "who": ["Tomás"],
      "knowledge": [
        { "holder": "Tomás", "channel": "sight", "path": "direct",
          "believes": "Sabe exactamente cómo salió y no lo cuenta." },
        { "holder": "las casas de Cuesta Menor", "channel": "el suelo", "path": "told",
          "believes": "Salió de una casa cerrada, y por eso no se lo acepta en ningún lado." },
        { "holder": "el Gremio de Tratantes", "channel": "la plazoleta", "path": "rumor",
          "believes": "Es una historia de la plaza y su historial es mala suerte estadística.",
          "accurate": false, "plausible_because": "nadie cruzó nunca su historial con la fecha del cierre" } ] },
    { "name": "el-recorte-del-anillo-bajo", "standing": "occurred",
      "what_happened": "Hace doce años se redujo la ración del anillo bajo; desde entonces se cerraron treinta y una casas.",
      "where": "anillo bajo", "who": ["Perla Anís", "la Junta de Alimento"],
      "knowledge": [
        { "holder": "Perla Anís", "channel": "sight", "path": "direct",
          "believes": "Es la autora material y lleva la cuenta exacta." },
        { "holder": "los Sin Trato", "channel": "la plazoleta", "path": "rumor",
          "believes": "La Junta vacía barrios a propósito.", "accurate": false,
          "plausible_because": "el registro es público y nadie se molestó en cruzarlo" } ] },
    { "name": "la-puerta-abierta", "standing": "occurred",
      "what_happened": "La Ochenta y Tres abrió la puerta a un aprendiz que pasaba caminando, sin golpe y sin negociación, después de sesenta años y ciento cuarenta rechazos.",
      "where": "Cuesta Menor", "who": ["la Ochenta y Tres", "Ordo Bes"],
      "knowledge": [
        { "holder": "Ordo Bes", "channel": "sight", "path": "direct",
          "believes": "Sabe algo sobre por qué y no lo dice." },
        { "holder": "Perla Anís", "channel": "la plazoleta", "path": "told",
          "believes": "Una casa vieja que acepta altera el reparto, y sugiere que decidía algo que nadie le preguntó." },
        { "holder": "los Sin Trato", "channel": "la plazoleta", "path": "overheard",
          "believes": "La casa más cerrada de Grelda le abrió la puerta a alguien que ya tenía dónde dormir." } ] }
  ],

  "arrivals": [
    { "premise": "Sos aprendiz de tratante, veinticuatro años, tres con Ordo Bes. Sabés golpear el suelo y sabés esperar; no sabés escuchar todavía. Anteayer la Sorda te abrió la puerta y Ordo te prohibió entrar.",
      "seen_as": "alguien joven con una vara demasiado nueva", "place": "Cuesta Menor",
      "capability": { "moves_by": ["caminar", "subir la cuesta"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "el suelo": "faint" } },
    { "premise": "Naciste bajo lona y nunca dormiste bajo techo. Subís la Cuesta por primera vez porque se corrió que la casa más cerrada de Grelda aceptó a alguien.",
      "seen_as": "alguien con la ropa gastada de once años de plaza", "place": "plaza mayor",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "el suelo": "absent" } }
  ]
}
```

## 2. Validity self-check

O1 **pass** (10 extents, 2 passages) · O2 **pass** (13 agency) · O3 **pass** — all 13 carry `pursuing`; in v2 *nine* did not · O4 **pass** (8 `hiding`) · O5 **pass** (4) · O6 **pass** — `bulk_class` added to all six people and both esqueje entities · O7 **pass** — `grano` held by the Junta and the fields; `ruido de gente` and `calor de fuego` held by Cuesta Menor · O8 **pass** (4 accumulators, all with `raised_by` + thresholds) · O9 **pass** — each indicator points at an accumulator or a named `hiding` · O10 **pass** (2) · O11 **pass** (6).

R1 **pass** — all names resolve; the v2 references to "el personaje del usuario" are gone · R2 **pass** — both passages join exactly 2 extents · R3 **pass** — "cuarenta días", "treinta y una", "ciento cuarenta" appear only in `stated`/`seen_as`/`asserts.claim`/`premise` · R4 **pass** · R5 **pass** — the three magnitudes are never addressed individually; I promoted `la Casa Vieja del portón sur` and `el esqueje que falta` out of theirs · R6 **pass** — nothing authored reopens a closed house or forces a deal · R7 **pass** · R8 **pass** (tree) · R9 **pass** · R10 **pass** · R11 **pass** — ladders ascend, `at` unique · R12 **pass** — the one disputed event has three holders who disagree.

## 3. Audit of my own v2 encoding

Fires: **R10** on nine entities (every house, guild, junta, Sin Trato, Plantadores, esquejes — I gave them `doing` and no `pursuing`); **R4** on `magnitude_class: "three thousand" | "eleven" | "three"` and on `"at": "cuarenta días" | "una" | "dos"`; **R3** on the same thresholds; **R1** on `"el personaje del usuario"` in four sections; **R5** on `las Casas Viejas` (Halma's plot promotes one) and on `los esquejes mal nacidos` (*"uno ya no está en su vivero"*); **O6** on all six people.

All real. **R10 was the most productive refusal in either world**: forcing a `pursuing` onto a *house* is what turned Grelda's houses from reactive scenery into agents — the Ochenta y Tres now pursues *"que alguien traduzca el ritmo antes de que se le acabe el tiempo"*, which is the whole plot and which I had left implicit in `hiding`.

**The refusal I think is wrong — R3, on `demand.unmet.onset_class`.** The forty days is the single hardest, most quoted number in this brief (*"Cuarenta días sin comer y una casa se cierra"*), it is what every character counts, and it is exactly the kind of figure v3 elsewhere permits as "player-read fiction". But `onset_class` and threshold `at` are engine-computed, so I had to demote it to `seasons`/`high` and re-state the number in prose. The document now says the number twice and means it once, and a builder reading only the machine fields gets a *vaguer* world than the brief specifies. R3 should permit an authored figure *alongside* a class in the same field (`{class, as_stated}`), rather than forcing duplication.

**Second, milder: R5 versus D3.** Same objection as in the Sueño paper — magnitude is defined by promotability, then referencing a promoted individual is refused, so I had to author magnitudes by subtraction ("las otras Casas Viejas", "los otros esquejes mal nacidos"). That is bookkeeping, and it silently rots as play promotes more.

## 4. Residual ambiguity

**RA1 — the player has no name.** Identical to the Sueño finding and worse here: the brief's premise is a relation *between the house and the apprentice*, and I cannot author it. I moved the standing to `la Ochenta y Tres → Ordo Bes`, which is a different story.

**RA2 — `process.acts_on` takes a predicate, not a name.** I wrote `"toda casa contenta"`. Under R1 that is either a legal predicate or an unresolved reference; nothing says which. And "contenta" is a hidden state no accumulator holds, so a builder cannot evaluate it.

**RA3 — where does the vara live?** `within: "Cuesta Menor"` (a place) or `within: "Ordo Bes"` (a person)? D1 makes both legal since `within` is the sole containment relation, and they mean very different things for whether he can be disarmed.

**RA4 — `confers` on a `matter` entity.** v3 keeps "capability is conferrable" but `confers` is still granted by no facet, so R7 arguably fires on my vara. I kept it and flag it: this is a contract hole, not a choice.

**RA5 — `accumulator.per` grammar,** unchanged from v2: I wrote `each entity with demand`, `each entity with agency`, `world`. Unconstrained and engine-facing.

## 5. Three under-specified reader obligations

**`process.rate_class` ⇒ "how fast state moves, without an event per tick".** Grelda's houses grow and shrink by *gaining and losing rooms* — a change in `extent_class`. The obligation says state moves; it does not say a process may change a facet key, so one builder grows a number and another adds a room. Visibly different worlds.

**`accumulator.thresholds` ⇒ "fire once, in order, at each crossing; `irreversible` never un-fires".** For "días sin comer", feeding the house should *lower* it. Nothing says whether a threshold that has fired at `low` re-arms when the value falls back, so one builder's fed house recovers its temper and another's stays sullen forever.

**`channel.latency_class` ⇒ "the delay before a fact is knowable to a receiver".** `el suelo` is `seasonal`, and the brief's whole point is *"hay una ventana durante la cual todavía nadie sabe lo que hiciste"*. Latency alone cannot say the window is *spatial* — near houses first, far houses months later. One builder makes reputation arrive everywhere after one delay; another makes it spread. The second is the brief; the contract does not distinguish them.
