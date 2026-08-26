# Sueño Común — world_model/3 (re-encode)

```jsonc
{
  "world_model": "3",
  "world": { "name": "Orbe",
    "premise": "Cuatrocientas mil personas en veintiocho barrios de sueño. Todos los que duermen en un mismo barrio entran en el mismo sueño, todas las noches, sin disfraz posible, y lo que uno desea se manifiesta alrededor suyo a la vista de todos. Una oficina lo transcribe y el tomo es público a las nueve.",
    "mood": ["frío", "vigilado", "íntimo por obligación"] },

  "excluded": [
    "El sueño no predice. La anomalía del usuario es única y ninguna otra escena puede validar la predicción como regla.",
    "No se controla lo que uno sueña: no hay técnica, disciplina, sustancia ni maestro.",
    "No hay magia ni entidades dentro del sueño. Solo gente del barrio.",
    "Nadie muere ni se lastima adentro. El daño es siempre social.",
    "No se puede ocultar la identidad adentro. Ningún truco funciona.",
    "Las transcripciones son texto: no hay imagen, grabación ni forma de revisitar un sueño pasado.",
    "Fuera de Orbe el fenómeno no ocurre y ninguna escena debe sugerir que se extiende.",
    "Los niños menores de diez años no se conectan. Sin excepciones."
  ],

  "vocabulary": {
    "media": [
      { "name": "aire de Orbe", "affords": [ { "to": "sight", "degree": "full" } ] },
      { "name": "materia de sueño",
        "descriptor": "escenario generado por la mayoría, versión deformada del propio barrio",
        "affords": [ { "to": "sight", "degree": "full" }, { "to": "caminar", "degree": "full" } ],
        "resists": [ { "to": "lastimar", "degree": "total" }, { "to": "ocultarse", "degree": "total" } ] },
      { "name": "materia fragmentada",
        "descriptor": "trozos inconexos que no llegan a formar un lugar",
        "resists": [ { "to": "caminar", "degree": "severe" } ] }
    ],
    "movements": [ { "name": "caminar", "pace_class": "steady" } ],
    "channels": [
      { "name": "sight", "reach": "la misma extensión", "latency_class": "immediate",
        "decay": "never", "conceals": "none" },
      { "name": "la mirada del mercado", "descriptor": "el castigo real: nadie te dice nada",
        "emitted_by": "cualquiera que leyó el tomo", "received_by": "el barrio",
        "latency_class": "hours", "reach": "el barrio entero", "decay": "never", "conceals": "identity" },
      { "name": "estar en el sueño", "descriptor": "todos ven todo y cada uno aparece como es",
        "emitted_by": "cualquier durmiente presente", "received_by": "todo durmiente presente",
        "latency_class": "immediate", "reach": "el interior del sueño", "decay": "brief",
        "conceals": "none" },
      { "name": "el tomo", "descriptor": "lo que el transcriptor decidió escribir",
        "emitted_by": "el transcriptor del barrio", "received_by": "cualquiera en la sala de lectura",
        "latency_class": "days", "reach": "la ciudad entera", "decay": "never", "conceals": "none" }
    ],
    "conditions": [
      { "name": "solitario", "alters": [
        { "channel": "estar en el sueño", "effect": "immune", "class": "total" },
        { "channel": "estar en el sueño", "effect": "grant", "class": "total" } ] },
      { "name": "insomne", "alters": [
        { "channel": "estar en el sueño", "effect": "broadcast", "class": "moderate" } ] },
      { "name": "desconectado", "alters": [
        { "channel": "estar en el sueño", "effect": "hinder", "class": "total" } ] }
    ],
    "substances": [ { "name": "durmientes" } ]
  },

  "law": [
    { "name": "un barrio, un sueño", "enforced_by": "physics", "within": "Orbe",
      "stated": "Todos los que duermen en un mismo barrio entran en el mismo sueño, todas las noches, sin forma de evitarlo salvo no dormir." },
    { "name": "nadie se oculta adentro", "enforced_by": "physics", "within": "el sueño del Doce",
      "stated": "Cada uno aparece como es: cara, cuerpo, edad, ropa del día.",
      "forbids": { "subject": "cualquier entidad con agencia", "act": "ocultar su identidad" } },
    { "name": "el deseo se manifiesta", "enforced_by": "physics", "within": "el sueño del Doce",
      "stated": "Lo que uno desea se manifiesta a su alrededor, visible para todos. No se reprime. Es la mecánica central del mundo." },
    { "name": "sin dolor", "enforced_by": "physics", "within": "el sueño del Doce",
      "stated": "Nadie se lastima, nadie muere, nadie sangra adentro.",
      "forbids": { "subject": "cualquier entidad", "act": "causar daño físico" } },
    { "name": "memoria breve", "enforced_by": "physics", "within": "Orbe",
      "stated": "Nadie recuerda más de tres noches atrás. Al cuarto día se borra." },
    { "name": "el sueño no predice", "enforced_by": "physics", "within": "Orbe",
      "stated": "Sin excepciones registradas en trescientos años." },
    { "name": "sueño ajeno", "enforced_by": "office", "within": "Orbe", "binds": [],
      "stated": "Dormir fuera del barrio asignado es delito: multa, reasignación y publicación del expediente." },
    { "name": "el registro no se abandona", "enforced_by": "office", "within": "Orbe",
      "binds": ["los otros solitarios registrados"],
      "stated": "Un solitario no puede renunciar. Solicitudes presentadas: catorce. Concedidas: cero." },
    { "name": "asignación a los diez", "enforced_by": "office", "within": "Orbe", "binds": [],
      "stated": "Se asigna barrio a los diez años. Antes, los niños duermen desconectados." },
    { "name": "no se comenta en la calle", "enforced_by": "persons", "within": "Orbe", "binds": [],
      "stated": "El sueño ajeno se comenta puertas adentro." },
    { "name": "no se le pregunta a un solitario", "enforced_by": "persons", "within": "Orbe", "binds": [],
      "stated": "No se le pregunta a un solitario qué vio." },
    { "name": "no se mira a los ojos", "enforced_by": "persons", "within": "Barrio Doce", "binds": [],
      "stated": "No se mira a los ojos al vecino en el mercado de la mañana." },
    { "name": "matrimonios entre barrios", "enforced_by": "persons", "within": "Orbe", "binds": [],
      "stated": "Los matrimonios se arreglan entre barrios distintos. Nadie lo explica. Todos saben." }
  ],

  "entities": [
    { "name": "Orbe", "facets": ["extent"], "extent_class": "vast", "medium": "aire de Orbe",
      "tension": "tense",
      "seen_as": "cuatrocientos mil habitantes, veintiocho barrios registrados y uno sin registrar" },
    { "name": "Barrio Doce", "facets": ["extent", "holding"], "within": "Orbe", "extent_class": "large",
      "medium": "aire de Orbe", "tension": "tense", "capacity_class": "ample",
      "holds": [ { "substance": "durmientes", "abundance": "ample" } ],
      "seen_as": "cuarenta mil habitantes, barrio medio, sin historia notable hasta anoche" },
    { "name": "Barrio Veintinueve", "facets": ["extent", "holding"], "within": "Orbe",
      "extent_class": "large", "medium": "aire de Orbe", "tension": "frantic",
      "capacity_class": "ample", "holds": [ { "substance": "durmientes", "abundance": "thin" } ],
      "seen_as": "no registrado, sin transcriptor y sin tomo; unas seis mil personas que nadie cuenta" },
    { "name": "Oficina del Doce", "facets": ["extent", "matter"], "within": "Barrio Doce",
      "extent_class": "small", "medium": "aire de Orbe", "tension": "normal",
      "bulk_class": "immense", "integrity": "sound", "seen_as": "plaza central, archivo de nueve a dieciséis" },
    { "name": "sala de lectura", "facets": ["extent"], "within": "Oficina del Doce",
      "extent_class": "intimate", "medium": "aire de Orbe", "tension": "tense",
      "seen_as": "donde se lee qué soñó el vecino; silencio obligatorio; siempre llena" },
    { "name": "Pensión Tarma", "facets": ["extent", "matter"], "within": "Barrio Doce",
      "extent_class": "small", "medium": "aire de Orbe", "tension": "tense",
      "bulk_class": "immense", "integrity": "worn",
      "seen_as": "camas por noche para gente de otros barrios; ilegal y notoria" },
    { "name": "Campanario", "facets": ["matter"], "within": "Barrio Doce", "bulk_class": "immense",
      "integrity": "sound", "seen_as": "campana de vigilia; suena a las cinco y corta el sueño de golpe" },

    { "name": "el sueño del Doce", "facets": ["extent", "demand"], "within": "Barrio Doce",
      "extent_class": "large", "medium": "materia de sueño", "tension": "tense",
      "seen_as": "una versión deformada del propio barrio, generada por la mayoría",
      "demands": [ { "substance": "durmientes", "rate_class": "nightly", "supplied_by": "draw",
                     "unmet": { "effect": "el sueño deja de existir hasta la noche siguiente",
                                "onset_class": "immediate" } } ] },
    { "name": "el sueño del Veintinueve", "facets": ["extent", "demand"], "within": "Barrio Veintinueve",
      "extent_class": "medium", "medium": "materia fragmentada", "tension": "frantic",
      "seen_as": "fragmentos inconexos que no cuajan, y que no siempre son de esta noche",
      "demands": [ { "substance": "durmientes", "rate_class": "nightly", "supplied_by": "draw",
                     "unmet": { "effect": "no se forma nada en absoluto", "onset_class": "immediate" } } ] },
    { "name": "el cuerpo del sueño de anoche", "facets": ["matter"], "within": "el sueño del Doce",
      "bulk_class": "moderate", "integrity": "worn",
      "seen_as": "un muerto que nadie reconoce, en un lugar que no se parece a ningún sitio de Orbe" },

    { "name": "el dormirse", "facets": ["passage"], "connects": ["Barrio Doce", "el sueño del Doce"],
      "admits": [ { "movement": "caminar" } ],
      "obstructs": [ { "condition": "desconectado" } ], "hazard_class": "none",
      "seen_as": "no es una puerta: es cerrar los ojos en tu propia cama" },
    { "name": "el paso al Veintinueve", "facets": ["matter", "passage"], "within": "Barrio Doce",
      "connects": ["Barrio Doce", "Barrio Veintinueve"], "bulk_class": "negligible", "integrity": "sound",
      "admits": [ { "movement": "caminar" } ], "obstructs": [], "hazard_class": "moderate",
      "seen_as": "un callejón sin vigilancia al final de la calle Ruma que todo el mundo sabe cuál es" },

    { "name": "la Oficina de Vigilia", "facets": ["agency", "collective"], "within": "Orbe",
      "legibility": "marked", "seen_as": "veintiocho oficinas y un Archivo central",
      "interest": "que nadie audite las transcripciones",
      "vulnerability": "cada tomo depende de una sola persona, reemplazable pero no auditable",
      "disposition": [ { "trait": "administrativa", "strength": "defining",
                         "manner": "archiva al mediodía y no comenta" } ],
      "doing": "archivando el tomo de anoche",
      "pursuing": [ { "horizon": "long_standing", "toward": "que la transcripción siga siendo la única fuente" } ],
      "hiding": "Cuántos tomos tienen noches faltantes." },
    { "name": "el Consejo de Orbe", "facets": ["agency", "collective"], "within": "Orbe",
      "legibility": "marked", "seen_as": "gobierno civil, formalmente por encima de la Oficina",
      "interest": "legislar sobre el sueño",
      "vulnerability": "no puede: cualquier debate quedaría transcripto",
      "disposition": [ { "trait": "cauto", "strength": "strong", "manner": "no pone nada por escrito" } ],
      "doing": "no convocando la sesión que todos esperan",
      "pursuing": [ { "horizon": "long_standing", "toward": "una ley sobre el sueño que no quede transcripta" } ] },
    { "name": "los otros solitarios registrados", "facets": ["agency", "magnitude"], "within": "Orbe",
      "magnitude_class": "few", "conditions": ["solitario"],
      "seen_as": "registrados, vigilados, útiles, y ninguno quiere estar ahí",
      "disposition": [ { "trait": "resignados", "strength": "strong", "manner": "declaran lo mínimo" } ],
      "doing": "asistiendo a juicios como prueba plena",
      "pursuing": [ { "horizon": "long_standing", "toward": "la baja del registro" } ] },
    { "name": "los insomnes de Orbe", "facets": ["agency", "magnitude"], "within": "Orbe",
      "magnitude_class": "many", "conditions": ["insomne"],
      "seen_as": "sin organización; la cifra oficial dice trescientos cuarenta y la real se estima en tres mil",
      "disposition": [ { "trait": "esquivos", "strength": "defining", "manner": "no duermen y no lo dicen" } ],
      "doing": "evitando los sitios donde se los vería proyectar",
      "pursuing": [ { "horizon": "long_standing", "toward": "no ser vistos" } ] },

    { "name": "Rem Salas", "facets": ["matter", "agency"], "within": "Barrio Doce",
      "bulk_class": "moderate", "integrity": "sound",
      "seen_as": "una mujer de cuarenta y cuatro años que escribe a mano y no levanta la vista",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "estar en el sueño": "acute" },
      "disposition": [ { "trait": "escrupulosa", "strength": "defining", "manner": "corrige antes de entregar" } ],
      "doing": "cerrando el tomo de anoche antes del mediodía",
      "pursuing": [ { "horizon": "long_standing", "toward": "no equivocarse nunca" } ],
      "hiding": "Omitió material en once tomos, siempre para proteger a alguien, y tiene la lista en su casa." },
    { "name": "Onel", "facets": ["matter", "agency"], "within": "Barrio Doce",
      "bulk_class": "moderate", "integrity": "sound", "conditions": ["solitario"],
      "seen_as": "un hombre de treinta y ocho años que vive apartado por norma del Registro",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "estar en el sueño": "acute" },
      "disposition": [ { "trait": "reticente", "strength": "defining", "manner": "declara menos de lo que vio" } ],
      "doing": "redactando la solicitud de baja número quince",
      "pursuing": [ { "horizon": "imminent", "toward": "la baja del registro" } ],
      "hiding": "Ve el sueño con mucho más detalle del que declara y puede seguir a una persona toda la noche." },
    { "name": "Vira Cor", "facets": ["matter", "agency"], "within": "Barrio Doce",
      "bulk_class": "moderate", "integrity": "worn", "conditions": ["insomne"],
      "seen_as": "una mujer de veintiséis a la que se le nota",
      "capability": { "moves_by": ["caminar"], "carry_class": "slight" },
      "senses": { "sight": "normal", "estar en el sueño": "absent" },
      "disposition": [ { "trait": "resistente", "strength": "defining", "manner": "no se sienta y no cierra los ojos" } ],
      "doing": "llegando a la noche once y empezando a proyectar de día",
      "pursuing": [ { "horizon": "imminent", "toward": "no volver a dormir nunca" } ],
      "hiding": "Qué es lo que no quiere que vea el barrio, y que el barrio ya empieza a ver igual." },
    { "name": "Inspector Bald", "facets": ["matter", "agency"], "within": "Barrio Doce",
      "bulk_class": "moderate", "integrity": "sound",
      "seen_as": "un hombre de cincuenta y uno con carpeta de la sección sueño ajeno",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "estar en el sueño": "normal" },
      "disposition": [ { "trait": "insistente", "strength": "strong", "manner": "vuelve al mismo local seis años seguidos" } ],
      "doing": "preparando una redada en la Pensión Tarma",
      "pursuing": [ { "horizon": "imminent", "toward": "cerrar la Pensión Tarma" } ],
      "hiding": "Duerme en el Doce y su carné dice Barrio Siete: cuatro años en falta contra la ley que aplica." },
    { "name": "Nea Salas", "facets": ["matter", "agency"], "within": "Barrio Doce",
      "bulk_class": "slight", "integrity": "sound",
      "seen_as": "una chica de diecisiete que mira la calle Ruma",
      "capability": { "moves_by": ["caminar"], "carry_class": "slight" },
      "senses": { "sight": "acute", "estar en el sueño": "normal" },
      "disposition": [ { "trait": "callada", "strength": "strong", "manner": "no cuenta dónde durmió" } ],
      "doing": "volviendo del paso al Veintinueve de madrugada",
      "pursuing": [ { "horizon": "long_standing", "toward": "irse de Orbe" } ],
      "hiding": "Durmió tres veces en el Veintinueve y sabe que los fragmentos de ahí son de otras noches." },
    { "name": "Archivista Mayor Ossen", "facets": ["matter", "agency"], "within": "Orbe",
      "bulk_class": "moderate", "integrity": "worn",
      "seen_as": "un hombre de sesenta y tres a un año de jubilarse",
      "capability": { "moves_by": ["caminar"], "carry_class": "slight" },
      "senses": { "sight": "faint", "estar en el sueño": "normal" },
      "disposition": [ { "trait": "prolijo", "strength": "defining", "manner": "cuenta antes de opinar" } ],
      "doing": "contando por tercera vez los tomos con noches faltantes",
      "pursuing": [ { "horizon": "imminent", "toward": "jubilarse sin firmar nada raro" } ],
      "hiding": "Son cuarenta y uno, en seis barrios, todos en los últimos nueve años." },
    { "name": "la Solitaria Primera", "facets": ["matter", "agency"], "within": "Orbe",
      "bulk_class": "slight", "integrity": "worn", "conditions": ["solitario"],
      "seen_as": "una mujer de setenta y uno sin nombre público, registrada desde los doce años",
      "capability": { "moves_by": ["caminar"], "carry_class": "slight" },
      "senses": { "sight": "normal", "estar en el sueño": "acute" },
      "disposition": [ { "trait": "impasible", "strength": "defining", "manner": "responde con fechas" } ],
      "doing": "esperando que alguien pregunte por la reforma",
      "pursuing": [ { "horizon": "long_standing", "toward": "que alguien pregunte antes de que se muera" } ],
      "hiding": "Sabe cómo se seleccionaba a los solitarios antes de la reforma." },

    { "name": "el tomo de anoche del Doce", "facets": ["matter", "record"], "within": "Oficina del Doce",
      "bulk_class": "negligible", "integrity": "sound",
      "asserts": [ { "claim": "qué pasó en el sueño del Doce anoche y quién estaba junto al cuerpo",
                     "accurate": false, "plausible_because": "Rem sabe qué escribió y qué dejó afuera" } ],
      "access": { "via": "sala de lectura a las nueve", "delay_class": "hours", "held_by": "Rem Salas" },
      "authority": "el transcriptor del barrio" },
    { "name": "los tomos con noches faltantes", "facets": ["matter", "record", "magnitude"],
      "within": "Orbe", "magnitude_class": "many", "bulk_class": "moderate", "integrity": "worn",
      "asserts": [ { "claim": "esas noches no ocurrieron", "accurate": false,
                     "plausible_because": "no hay entrada, y nadie cruzó los seis barrios con los nueve años" } ],
      "access": { "via": "Archivo central", "delay_class": "days", "held_by": "la Oficina de Vigilia" },
      "authority": "el Archivo central",
      "seen_as": "cuarenta y un tomos identificados, seis barrios, nueve años" },
    { "name": "la lista de Rem", "facets": ["matter", "record"], "within": "Barrio Doce",
      "bulk_class": "negligible", "integrity": "sound",
      "asserts": [ { "claim": "once omisiones, once nombres", "accurate": true } ],
      "access": { "via": "estar en su casa", "delay_class": "immediate", "held_by": "Rem Salas" },
      "authority": "ninguna: no tiene razón administrativa para existir" },
    { "name": "los carnés de barrio", "facets": ["matter", "record", "magnitude"], "within": "Orbe",
      "magnitude_class": "multitude", "bulk_class": "negligible", "integrity": "sound",
      "asserts": [ { "claim": "dónde le toca dormir a cada uno", "accurate": true } ],
      "access": { "via": "se pide en pensiones, y casi nunca se pide", "delay_class": "immediate",
                  "held_by": "cada habitante" },
      "authority": "la Oficina de Vigilia" }
  ],

  "offices": [
    { "name": "Transcriptor del barrio", "held_by": "Rem Salas", "of": "la Oficina de Vigilia",
      "confers": [ { "act": "decidir qué entra en el tomo" }, { "act": "archivar al mediodía" } ],
      "succeeds_by": "designación de la Oficina" },
    { "name": "Solitario registrado", "held_by": "Onel", "of": "la Oficina de Vigilia",
      "confers": [ { "act": "testificar con prueba plena" } ],
      "succeeds_by": "registro forzoso; no se puede renunciar" },
    { "name": "Inspector de sueño ajeno", "held_by": "Inspector Bald", "of": "la Oficina de Vigilia",
      "confers": [ { "act": "abrir expediente y publicarlo" }, { "act": "pedir el carné" } ],
      "succeeds_by": "designación" },
    { "name": "Archivista Mayor", "held_by": "Archivista Mayor Ossen", "of": "la Oficina de Vigilia",
      "confers": [ { "act": "acceder a cualquier tomo de cualquier barrio" } ], "succeeds_by": "antigüedad" }
  ],

  "standing": [
    { "from": "Barrio Doce", "toward": "Vira Cor",
      "stance": "ya empieza a ver, despierto, lo que ella no quería que nadie viera",
      "since": null, "carried_by": "la mirada del mercado", "persistence": "never decays" },
    { "from": "la Oficina de Vigilia", "toward": "los otros solitarios registrados",
      "stance": "los necesita como prueba plena y no les concede la baja",
      "since": null, "carried_by": null, "persistence": "until changed" },
    { "from": "Rem Salas", "toward": "Nea Salas",
      "stance": "sospecha dónde duerme y no quiere confirmarlo",
      "since": null, "carried_by": null, "persistence": "until changed" },
    { "from": "Inspector Bald", "toward": "Pensión Tarma",
      "stance": "seis años queriendo cerrarla y sin poder explicar por qué le importa tanto",
      "since": null, "carried_by": null, "persistence": "until changed" }
  ],

  "opposition": [
    { "between": ["Onel", "la Oficina de Vigilia"],
      "incompatible": "No puede darle la baja y conservar prueba plena en los juicios.",
      "stakes": "Diecinueve años y catorce solicitudes denegadas." },
    { "between": ["Vira Cor", "Barrio Doce"],
      "incompatible": "No dormir es lo único que la protege y es exactamente lo que la expone.",
      "stakes": "La noche catorce es el colapso." },
    { "between": ["Inspector Bald", "Pensión Tarma"],
      "incompatible": "No puede cerrarla sin que aparezca su propio nombre en el libro de huéspedes.",
      "stakes": "Cuatro años en falta contra la ley que aplica." },
    { "between": ["Rem Salas", "Archivista Mayor Ossen"],
      "incompatible": "La lista de omisiones y el recuento de noches faltantes no pueden ser ambos privados.",
      "stakes": "Quién queda como responsable de nueve años de huecos." }
  ],

  "processes": [
    { "name": "acumulación de insomnio", "acts_on": "toda entidad con la condición insomne",
      "direction": "degrade", "rate_class": "nightly", "terminus": "colapso" },
    { "name": "borrado de la memoria propia", "acts_on": "toda entidad con agencia que durmió",
      "direction": "degrade", "rate_class": "nightly", "terminus": "no queda nada propio" }
  ],

  "cycles": [
    { "name": "la noche del barrio", "period_class": "daily", "starts_in_phase": "apertura",
      "phases": [
        { "name": "apertura", "changes": [ { "entity": "el sueño del Doce", "becomes": "existe" } ] },
        { "name": "sueño", "changes": [ { "tension_of": "el sueño del Doce", "becomes": "tense" } ] },
        { "name": "la campana", "changes": [ { "entity": "el sueño del Doce", "becomes": "cortado" } ] },
        { "name": "mercado", "changes": [ { "tension_of": "Barrio Doce", "becomes": "tense" } ] },
        { "name": "las nueve", "changes": [ { "entity": "el tomo de anoche del Doce", "becomes": "público" } ] } ] }
  ],

  "accumulators": [
    { "name": "noches sin dormir", "per": "each entity with agency", "starts_at": "none",
      "stated": "Cuántas noches seguidas lleva alguien sin dormir. A la cuarta empiezan las proyecciones breves; entre la quinta y la octava se ven a veinte metros; de la novena a la trece el barrio ve lo que la persona piensa; en la catorce es el colapso.",
      "raised_by": [ { "event": "una noche sin dormir" } ],
      "thresholds": [
        { "at": "low", "then": "proyecciones diurnas breves" },
        { "at": "moderate", "then": "proyecciones frecuentes, visibles a la distancia de una calle" },
        { "at": "high", "then": "proyección continua: el barrio ve lo que la persona piensa" },
        { "at": "extreme", "then": "colapso", "irreversible": true } ] },
    { "name": "omisiones de un transcriptor", "per": "each office", "starts_at": "moderate",
      "stated": "Cuántas veces un transcriptor dejó algo afuera. Rem lleva once.",
      "raised_by": [ { "event": "una omisión en un tomo" } ],
      "thresholds": [ { "at": "high", "then": "la lista existe y es prueba contra quien la escribió" } ] },
    { "name": "noches faltantes en el Archivo", "per": "world", "starts_at": "moderate",
      "stated": "Cuántas noches no tienen tomo. Van cuarenta y una, en seis barrios, en nueve años.",
      "raised_by": [ { "event": "una noche sin entrada" } ],
      "thresholds": [ { "at": "high", "then": "el patrón deja de poder atribuirse a descuido" } ] }
  ],

  "indicators": [
    { "of": "noches sin dormir",
      "shows_as": ["proyecciones breves", "proyecciones a la distancia de una calle", "proyección continua"],
      "read_by": { "channel": "sight", "requires": {} }, "reliability_class": "good" },
    { "of": "el hiding de Onel",
      "shows_as": ["declara siempre menos de lo que se le pregunta"],
      "read_by": { "channel": "sight", "requires": { "office": "Inspector de sueño ajeno" } },
      "reliability_class": "poor" },
    { "of": "omisiones de un transcriptor",
      "shows_as": ["un tomo más corto que la noche", "una lista en la casa de quien lo escribió"],
      "read_by": { "channel": "el tomo", "requires": { "office": "Archivista Mayor" } },
      "reliability_class": "poor" },
    { "of": "el hiding de Inspector Bald",
      "shows_as": ["un nombre en el libro de huéspedes de la pensión que persigue"],
      "read_by": { "channel": "sight", "requires": {} }, "reliability_class": "moderate" }
  ],

  "traces": [
    { "of": "una noche en el sueño", "leaves": "tres noches de memoria propia y un tomo permanente",
      "ages": "la memoria en días; el tomo nunca" },
    { "of": "haber deseado algo delante del barrio", "leaves": "la mirada del mercado a la mañana siguiente",
      "ages": "never" },
    { "of": "una noche sin tomo", "leaves": "un hueco en el Archivo central", "ages": "never" }
  ],

  "epochs": [
    { "name": "antes de la reforma del Registro",
      "differed": [ { "topic": "office", "subject": "Solitario registrado", "then": "se seleccionaba de otra manera" } ],
      "surviving_traces": ["la Solitaria Primera, registrada desde los doce años"] },
    { "name": "los ochenta años de estudio",
      "differed": [ { "topic": "law", "subject": "un barrio, un sueño", "then": "se investigaba por qué" } ],
      "surviving_traces": ["que nadie sabe por qué el fenómeno es urbano", "que se dejó de estudiar"] }
  ],

  "history": [
    { "name": "el-sueno-del-asesinato", "standing": "disputed",
      "what_happened": "El barrio soñó un asesinato en un lugar que no es Orbe, con un muerto que nadie reconoce, y cuarenta mil personas vieron a alguien parado junto al cuerpo.",
      "where": "el sueño del Doce", "who": ["Rem Salas", "Onel"],
      "knowledge": [
        { "holder": "Rem Salas", "channel": "estar en el sueño", "path": "direct",
          "believes": "Vio la escena y sabe qué dejó afuera del tomo." },
        { "holder": "Onel", "channel": "estar en el sueño", "path": "direct",
          "believes": "Vio el sueño completo y con más detalle del que va a declarar." },
        { "holder": "el Consejo de Orbe", "channel": "el tomo", "path": "told",
          "believes": "Si el sueño predijo, la regla es falsa y el mundo funciona distinto.",
          "accurate": false,
          "plausible_because": "la otra hipótesis —que un recuerdo cruzó— es peor y nadie la ha formulado" },
        { "holder": "los insomnes de Orbe", "channel": "la mirada del mercado", "path": "rumor",
          "believes": "Fue un insomne proyectando, y por eso el barrio los va a buscar.",
          "accurate": false, "plausible_because": "es la explicación que no obliga a revisar nada" } ] },
    { "name": "la-misma-escena-en-el-Siete", "standing": "disputed",
      "what_happened": "La misma escena apareció en el Barrio Siete tres noches después.",
      "where": "Orbe", "who": ["Archivista Mayor Ossen"],
      "knowledge": [
        { "holder": "Archivista Mayor Ossen", "channel": "el tomo", "path": "direct",
          "believes": "Distinto barrio, mismo sueño, y eso no debería poder ocurrir." },
        { "holder": "la Oficina de Vigilia", "channel": "el tomo", "path": "told",
          "believes": "Dos transcriptores copiaron el rumor del otro barrio.",
          "accurate": false, "plausible_because": "sería el error administrativo más barato de admitir" } ] }
  ],

  "arrivals": [
    { "premise": "Barrio Doce, treinta y un años, sin antecedentes, nunca apareciste en un tomo por nada relevante. Anoche cuarenta mil personas te vieron parado junto a un cuerpo y el tomo es público a las nueve.",
      "seen_as": "alguien corriente al que en el mercado nadie mira a los ojos", "place": "Barrio Doce",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "estar en el sueño": "normal", "el tomo": "normal" } },
    { "premise": "Venís de otro barrio y pagaste una cama en la Pensión Tarma. Anoche entraste en un sueño que no es el tuyo y alguien te vio.",
      "seen_as": "alguien con un carné que no conviene mostrar", "place": "Pensión Tarma",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "estar en el sueño": "normal" } }
  ]
}
```

## 2. Validity self-check

| # | Verdict |
|---|---|
| O1 | **pass** — 9 extents; `el dormirse` and `el paso al Veintinueve` are passages |
| O2 | **pass** — 10 agency entities |
| O3 | **pass** — every one of the 10 carries `pursuing`; this is what most of my v2 rewrite went on |
| O4 | **pass** — 8 `hiding` entries |
| O5 | **pass** — 4 oppositions |
| O6 | **pass** — every `matter` entity has `bulk_class` (added to all seven people, who lacked it in v2) |
| O7 | **pass** — `durmientes` is held by Barrio Doce and Barrio Veintinueve |
| O8 | **pass** — 3 accumulators, each with `raised_by` and thresholds |
| O9 | **pass** — 4 indicators, each naming an accumulator or a named `hiding` |
| O10 | **pass** — 2 arrivals |
| O11 | **pass** — 8 entries |
| R1 | **pass** — every reference resolves; I deleted the v2 references to "el personaje del usuario" and "el barrio" (see §4) |
| R2 | **pass** — both passages connect exactly 2 extents |
| R3 | **pass** — figures appear only in `stated`, `seen_as`, `premise`, `asserts.claim`; thresholds use `low…extreme` |
| R4 | **pass** — one ladder per field throughout |
| R5 | **pass** — the three magnitudes are never referenced individually; Onel, Vira and the Solitaria Primera are separate entities, not members drawn out |
| R6 | **pass** — nothing authored contradicts an `excluded` line; the anomaly is authored as `disputed`, not as a prediction |
| R7 | **pass** — `holds` only on `holding`; `connects` only on `passage`; `magnitude_class` only on `magnitude` |
| R8 | **pass** — containment is a tree (Orbe → barrios → oficina → sala) |
| R9 | **pass** — see O7 |
| R10 | **pass** — see O3 |
| R11 | **pass** — ladders ascend, no repeated `at` |
| R12 | **pass** — both disputed events now carry a disagreeing holder |

## 3. Audit of my own v2 encoding

Six refusals fire on the document I wrote last round: **R3** (`"at": "treinta por ciento"`, `"at": "cuatro"`), **R4** (`magnitude_class: "eleven"`, `"twelve thousand"` — not classes), **R10** (six agency entities with no `pursuing`), **R11** (the 30%/10% ladder ran *downward*), **R12** (`la-misma-escena-en-el-Siete` disputed with one holder), **R1** (`"el barrio"`, `"el personaje del usuario"` unresolved). Also O6 (no `bulk_class` on people) and O11-as-obligation were fine but O3 was not.

**Every one of those is a real error.** R11 is the most valuable: it forced me to notice that I had encoded the dream's 30/10 hysteresis as an accumulator *and* a demand, which was duplication hiding a modelling failure. Under v3 the demand alone carries it, and the encoding is better.

**One refusal I think is wrong: R5, as written.** It refuses "referencing a `magnitude` entity as an individual", but D3 defines magnitude by *promotability* — "play may need to promote an individual out of it". Those two rules pull opposite ways: the moment play promotes Onel, *any* reference to him is a reference to an individual formerly inside the magnitude. My workaround was to author `los otros solitarios registrados` — a magnitude defined by *subtraction* of the named ones. That is bookkeeping the author should not have to do, and it degrades: every time a new solitario is promoted in play, the authored magnitude silently means something different. R5 should refuse only *simultaneous* individual and magnitude membership in the same document, and the contract should say that promotion in play is legal and does not retro-invalidate the magnitude.

**A second, milder objection: R2 + `el dormirse`.** Falling asleep passes O1's real test — there are two extents and a way between them — but calling it a `passage` with `admits: [{movement: "caminar"}]` is a lie: you do not walk into the dream. R2 is right; the gap is that `admits` has no way to name "the act of sleeping" as the admitting predicate.

## 4. Residual ambiguity

**RA1 — The player is not an entity, and things want to point at them.** My v2 document referenced "el personaje del usuario" in `offices.held_by`, `standing`, `opposition` and `history.who`. Under R1 those are unresolved names, so I deleted them all — and lost the world's central opposition (the player against Rem's tomo) and the standing that *is* the premise (forty thousand people saw you). `arrivals[]` is a premise, not a referenceable name. Two encoders will diverge on whether to author a placeholder entity for the arriving character. **This is the single largest remaining hole.**

**RA2 — `demands[].supplied_by`: `draw` or `emission`?** The dream consumes sleepers by existing. I wrote `draw` because `holds` makes Barrio Doce the supplier; `emission` (sleeping *produces* the dream) is equally defensible and inverts the reader obligation for `holds[].abundance`.

**RA3 — Is `el dormirse` an entity?** A passage with no `matter` and no `within`. Legal by the facet gate, strange as a noun. The alternative — no passage at all, the dream simply being entered by sleeping as a law — fails O1 if the dream is one of only two extents.

**RA4 — `accumulator.per` scope grammar is still unspecified.** I wrote `per: "each entity with agency"`, `per: "each office"`, `per: "world"`. Nothing in v3 constrains that string, and it is engine-facing.

**RA5 — `indicator.of` targets.** I pointed two indicators at *`hiding` strings* ("el hiding de Onel"). O9 says an indicator must name "a hidden state some accumulator or property actually holds" — `hiding` is a property, so this passes, but "the hidden state" and "the string describing it" are not the same object, and a builder cannot tell which to compare against.

## 5. Three under-specified reader obligations

**`channel.conceals: none` ⇒ "renders present entities' `pursuing` and `hiding` to all receivers".** This is the strongest reader obligation in the contract and it is doing enormous unstated work here — it is the only thing that makes the dream's central mechanic mechanical. But it says *renders*, not *when* or *to whom in what form*: does the receiver acquire a belief with `path: direct`? A permanent one, given the channel's `decay: brief`? Two builders will differ on whether the market's gaze the next morning is a *belief* or a *standing*.

**`tension` ⇒ "a beat budget; acts exceeding it become extended rather than refused".** In an entity whose existence is nightly and whose `demand` can end it mid-scene, "extended" has no defined meaning. One builder ends the act at the campana; another carries it into vigilia.

**`demand.unmet` ⇒ "apply the effect after `onset_class`, and go on applying it".** For the dream, the effect *is* ceasing to exist, and "go on applying it" is meaningless. The contract has no notion of an effect that is terminal-but-recurring, which is exactly what a nightly reality is.
