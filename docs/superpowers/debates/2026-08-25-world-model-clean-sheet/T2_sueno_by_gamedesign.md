# Sueño Común — world_model/2 encoding (detailed tier, then the basic-tier diff)

## 1. Encoding — detailed tier

```jsonc
{
  "world_model": "2",
  "world": { "name": "Orbe",
    "premise": "Cuatrocientas mil personas en veintiocho barrios de sueño. Todos los que duermen en un mismo barrio entran en el mismo sueño, todas las noches, sin disfraz posible, y lo que uno desea se manifiesta alrededor suyo a la vista de todos. Una oficina lo transcribe y el tomo es público a las nueve.",
    "mood": ["frío", "vigilado", "íntimo por obligación"] },

  "excluded": [
    "El sueño no predice. La anomalía del usuario es única y ninguna otra escena puede validar la predicción como regla.",
    "No se controla lo que uno sueña. No hay técnica, disciplina, sustancia ni maestro.",
    "No hay magia ni entidades dentro del sueño. Solo gente del barrio.",
    "Nadie muere ni se lastima adentro. El daño es siempre social.",
    "No se puede ocultar la identidad adentro. Ningún truco funciona.",
    "Las transcripciones son texto. No existe imagen, grabación ni forma de revisitar un sueño pasado.",
    "Fuera de Orbe el fenómeno no ocurre y ninguna escena debe sugerir que se extiende.",
    "Los niños menores de diez años no se conectan. Sin excepciones."
  ],

  "layers": [
    { "name": "vigilia", "default": true },
    { "name": "el sueño del barrio",
      "note": "// AMBIGUITY A1: podría ser 28 layers (uno por barrio) o una layer con 28 entidades. Elegí lo segundo." },
    { "name": "el sueño roto",
      "note": "el Veintinueve: no cuaja, se fragmenta, y los fragmentos son de otras noches" }
  ],

  "vocabulary": {
    "media": [
      { "name": "aire de Orbe", "layer": "vigilia", "affords": [ { "to": "sight", "degree": "full" } ] },
      { "name": "materia de sueño", "layer": "el sueño del barrio",
        "descriptor": "escenario generado por la mayoría, versión deformada del propio barrio",
        "affords": [ { "to": "sight", "degree": "full" }, { "to": "caminar", "degree": "full" },
                     { "to": "hablar", "degree": "full" } ],
        "resists": [ { "to": "lastimar", "degree": "total" }, { "to": "ocultarse", "degree": "total" } ] },
      { "name": "materia fragmentada", "layer": "el sueño roto",
        "descriptor": "trozos inconexos que no llegan a formar un lugar",
        "resists": [ { "to": "caminar", "degree": "severe" } ] }
    ],
    "movements": [ { "name": "caminar", "pace_class": "steady" } ],
    "channels": [
      { "name": "sight", "layer": "vigilia" },
      { "name": "la mirada del mercado", "layer": "vigilia",
        "descriptor": "el castigo real: nadie te dice nada",
        "emitted_by": "cualquiera que leyó el tomo", "received_by": "el barrio",
        "latency_class": "hours", "reach": "el barrio", "decay": "never", "conceals": "identity" },
      { "name": "estar en el sueño", "layer": "el sueño del barrio",
        "descriptor": "todos ven todo, y cada uno aparece como es",
        "emitted_by": "cualquier durmiente del barrio", "received_by": "todos los durmientes del barrio",
        "latency_class": "immediate", "reach": "el sueño entero", "decay": "three nights",
        "conceals": "none",
        "note": "// el decay de tres noches es la memoria propia — ver A2" },
      { "name": "el tomo", "layer": "vigilia",
        "descriptor": "lo que el transcriptor decidió escribir",
        "emitted_by": "el transcriptor del barrio", "received_by": "cualquiera en la sala de lectura",
        "latency_class": "one day", "reach": "la ciudad, vía Archivo central", "decay": "never",
        "conceals": "none" }
    ],
    "conditions": [
      { "name": "solitario",
        "alters": [ { "channel": "estar en el sueño", "effect": "immune", "class": "total" },
                    { "channel": "estar en el sueño", "effect": "grant", "class": "total",
                      "note": "ve el sueño completo sin aparecer en él" } ] },
      { "name": "insomne",
        "alters": [ { "channel": "estar en el sueño", "effect": "broadcast", "class": "moderate",
                      "note": "proyecta en vigilia lo que piensa, visible al barrio" } ] },
      { "name": "desconectado",
        "alters": [ { "channel": "estar en el sueño", "effect": "hinder", "class": "total" } ],
        "note": "los menores de diez años y quien duerme fuera de Orbe" }
    ],
    "substances": [ { "name": "durmientes", "note": "lo que el sueño necesita para cuajar" } ]
  },

  "law": [
    { "name": "un barrio, un sueño", "enforced_by": "physics",
      "stated": "Todos los que duermen en un mismo barrio entran en el mismo sueño, todas las noches, sin esfuerzo y sin forma de evitarlo salvo no dormir." },
    { "name": "nadie se oculta adentro", "enforced_by": "physics",
      "stated": "Cada uno aparece como es: cara, cuerpo, edad, ropa del día. Sin máscara, sin ausencia, sin mentira visual.",
      "forbids": { "subject": "cualquier entidad con agencia en el sueño del barrio", "act": "ocultar su identidad" } },
    { "name": "el deseo se manifiesta", "enforced_by": "physics",
      "stated": "Lo que uno desea se manifiesta a su alrededor, visible para todos. No se reprime. Es la mecánica central del mundo.",
      "note": "// BREAK B1: esto inhabilita `hiding` dentro de una layer y el esquema no puede decirlo" },
    { "name": "memoria de tres noches", "enforced_by": "physics",
      "stated": "Nadie recuerda más de tres noches atrás. Al cuarto día se borra." },
    { "name": "sin dolor", "enforced_by": "physics",
      "stated": "Nadie se lastima, nadie muere, nadie sangra adentro.",
      "forbids": { "subject": "cualquier entidad", "act": "causar daño físico en el sueño del barrio" } },
    { "name": "el sueño no predice", "enforced_by": "physics",
      "stated": "Sin excepciones registradas en trescientos años." },
    { "name": "no se elige lo que se sueña", "enforced_by": "physics",
      "stated": "No hay técnica ni sustancia que lo consiga." },
    { "name": "sueño ajeno", "enforced_by": "office", "binds": [],
      "stated": "Dormir fuera del barrio asignado es delito. Pena: multa, reasignación, y publicación del expediente." },
    { "name": "el registro no se abandona", "enforced_by": "office", "binds": ["los solitarios"],
      "stated": "Un solitario no puede renunciar al registro. Solicitudes presentadas: catorce. Concedidas: cero." },
    { "name": "asignación a los diez", "enforced_by": "office", "binds": [],
      "stated": "Se asigna barrio a los diez años. Antes, los niños duermen desconectados." },
    { "name": "no se comenta en la calle", "enforced_by": "persons", "binds": [],
      "stated": "El sueño ajeno se comenta puertas adentro, nunca en la calle." },
    { "name": "no se le pregunta a un solitario", "enforced_by": "persons", "binds": [],
      "stated": "No se le pregunta a un solitario qué vio." },
    { "name": "no se mira a los ojos", "enforced_by": "persons", "binds": [],
      "stated": "No se mira a los ojos al vecino en el mercado de la mañana." },
    { "name": "matrimonios entre barrios", "enforced_by": "persons", "binds": [],
      "stated": "Los matrimonios se arreglan entre barrios distintos. Nadie lo explica. Todos saben." },
    { "name": "los transcriptores no se saludan", "enforced_by": "persons", "binds": ["la Oficina de Vigilia"],
      "stated": "Los transcriptores no se saludan entre barrios." }
  ],

  "entities": [
    { "name": "Orbe", "facets": ["extent"], "layer": "vigilia", "extent_class": "vast",
      "medium": "aire de Orbe", "tension": "tense",
      "note_numbers_read_by_players": "400.000 habitantes, 28 barrios registrados, uno sin registrar" },
    { "name": "Barrio Doce", "facets": ["extent"], "within": "Orbe", "layer": "vigilia",
      "extent_class": "large", "medium": "aire de Orbe", "tension": "tense",
      "seen_as": "cuarenta mil habitantes, barrio medio, sin historia notable hasta anoche" },
    { "name": "Oficina del Doce", "facets": ["extent", "matter"], "within": "Barrio Doce", "layer": "vigilia",
      "extent_class": "small", "medium": "aire de Orbe", "tension": "normal", "bulk_class": "immense",
      "integrity": "sound", "seen_as": "plaza central, archivo abierto de nueve a dieciséis" },
    { "name": "sala de lectura", "facets": ["extent"], "within": "Oficina del Doce", "layer": "vigilia",
      "extent_class": "intimate", "medium": "aire de Orbe", "tension": "tense",
      "seen_as": "donde se lee qué soñó el vecino; silencio obligatorio; siempre llena" },
    { "name": "Pensión Tarma", "facets": ["extent", "matter"], "within": "Barrio Doce", "layer": "vigilia",
      "extent_class": "small", "medium": "aire de Orbe", "tension": "tense", "bulk_class": "immense",
      "integrity": "worn", "seen_as": "camas por noche para gente de otros barrios; ilegal y notoria" },
    { "name": "casa de Onel", "facets": ["extent", "matter"], "within": "Barrio Doce", "layer": "vigilia",
      "extent_class": "intimate", "medium": "aire de Orbe", "tension": "calm", "bulk_class": "immense",
      "integrity": "sound", "seen_as": "apartada por norma del Registro" },
    { "name": "paso al Veintinueve", "facets": ["matter", "passage"], "within": "Barrio Doce",
      "layer": "vigilia", "connects": ["Barrio Doce", "Barrio Veintinueve"], "bulk_class": "negligible",
      "integrity": "sound", "seen_as": "un callejón sin vigilancia al final de la calle Ruma",
      "admits": [ { "movement": "caminar" } ], "obstructs": [], "hazard_class": "none",
      "note": "todo el mundo sabe cuál es" },
    { "name": "Campanario", "facets": ["matter"], "within": "Barrio Doce", "layer": "vigilia",
      "bulk_class": "immense", "integrity": "sound",
      "seen_as": "campana de vigilia; suena a las cinco y corta el sueño de golpe" },
    { "name": "Barrio Veintinueve", "facets": ["extent"], "within": "Orbe", "layer": "vigilia",
      "extent_class": "large", "medium": "aire de Orbe", "tension": "frantic",
      "seen_as": "no registrado, sin transcriptor, sin tomo; población estimada seis mil y nadie la cuenta" },
    { "name": "el campo, fuera de Orbe", "facets": ["extent"], "layer": "vigilia", "extent_class": "vast",
      "medium": "aire de Orbe", "tension": "calm",
      "seen_as": "donde se sueña solo, y nadie sabe por qué" },

    { "name": "el sueño del Doce", "facets": ["extent", "demand"], "layer": "el sueño del barrio",
      "extent_class": "large", "medium": "materia de sueño", "tension": "tense",
      "seen_as": "una versión deformada del propio barrio, generada por la mayoría",
      "demands": [ { "substance": "durmientes", "rate_class": "nightly", "supplied_by": "emission",
                     "unmet": { "effect": "el sueño se cierra", "onset_class": "immediate" },
                     "note": "// BREAK B2: arranca al 30% del barrio y termina bajo el 10%; un umbral de existencia, no una carencia" } ] },
    { "name": "el sueño del Veintinueve", "facets": ["extent", "demand"], "layer": "el sueño roto",
      "extent_class": "medium", "medium": "materia fragmentada", "tension": "frantic",
      "seen_as": "fragmentos inconexos que no cuajan",
      "hiding": "Los fragmentos son de otras noches." },
    { "name": "el cuerpo del sueño de anoche", "facets": ["matter"], "within": "el sueño del Doce",
      "layer": "el sueño del barrio", "bulk_class": "moderate", "integrity": "worn",
      "seen_as": "un muerto que nadie del barrio reconoce, en un lugar que no se parece a ningún sitio conocido" },

    { "name": "la Oficina de Vigilia", "facets": ["agency", "collective"], "within": "Orbe",
      "layer": "vigilia", "legibility": "marked",
      "seen_as": "veintiocho oficinas y un Archivo central",
      "interest": "que nadie audite las transcripciones",
      "vulnerability": "cada tomo depende de una sola persona, y esa persona es reemplazable pero no auditable",
      "doing": "archivando el tomo de anoche al mediodía" },
    { "name": "el Registro de Solitarios", "facets": ["agency", "collective", "record"], "within": "Orbe",
      "layer": "vigilia", "legibility": "marked",
      "seen_as": "once nombres, dependencia directa del Archivo central",
      "interest": "mantener el número: un solitario menos es un juicio menos",
      "vulnerability": "son once, están identificados, y ninguno quiere estar ahí",
      "asserts": [ { "claim": "once solitarios en toda la ciudad", "accurate": true } ],
      "access": { "via": "reservado", "delay_class": "none", "held_by": "el Archivo central" },
      "authority": "el Archivo central" },
    { "name": "los Insomnes", "facets": ["agency", "magnitude"], "within": "Orbe", "layer": "vigilia",
      "magnitude_class": "many", "seen_as": "sin organización; cifra oficial 340, cifra real estimada tres mil o más",
      "conditions": ["insomne"], "interest": "no ser vistos",
      "vulnerability": "el método los destruye",
      "note": "// AMBIGUITY A4: `collective`? Tienen interés y debilidad pero no están constituidos" },
    { "name": "los solitarios", "facets": ["agency", "magnitude"], "within": "Orbe", "layer": "vigilia",
      "magnitude_class": "eleven", "conditions": ["solitario"],
      "seen_as": "registrados, vigilados, útiles", "interest": "la baja",
      "vulnerability": "su testimonio es prueba plena y por eso no los sueltan" },
    { "name": "Consejo de Orbe", "facets": ["agency", "collective"], "within": "Orbe", "layer": "vigilia",
      "legibility": "marked", "seen_as": "gobierno civil, formalmente por encima de la Oficina",
      "interest": "legislar sobre el sueño",
      "vulnerability": "no puede: cualquier debate quedaría transcripto" },

    { "name": "Rem Salas", "facets": ["matter", "agency"], "within": "Barrio Doce", "layer": "vigilia",
      "seen_as": "una mujer de cuarenta y cuatro años que escribe a mano y no levanta la vista",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "estar en el sueño": "acute" },
      "disposition": [ { "trait": "escrupulosa", "strength": "defining", "manner": "corrige antes de entregar" } ],
      "doing": "cerrando el tomo de anoche antes del mediodía",
      "pursuing": [ { "horizon": "long_standing", "toward": "no equivocarse nunca" } ],
      "hiding": "Omitió material en once tomos, siempre para proteger a alguien, y tiene la lista de las once omisiones en su casa." },
    { "name": "Onel", "facets": ["matter", "agency"], "within": "casa de Onel", "layer": "vigilia",
      "seen_as": "un hombre de treinta y ocho años que vive apartado",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "estar en el sueño": "acute" }, "conditions": ["solitario"],
      "disposition": [ { "trait": "reticente", "strength": "defining", "manner": "declara menos de lo que vio" } ],
      "doing": "redactando la solicitud de baja número quince",
      "pursuing": [ { "horizon": "long_standing", "toward": "la baja" } ],
      "hiding": "Ve el sueño con mucho más detalle del que declara y puede seguir a una persona concreta toda la noche. La Oficina no lo sabe." },
    { "name": "Vira Cor", "facets": ["matter", "agency"], "within": "Barrio Doce", "layer": "vigilia",
      "seen_as": "una mujer de veintiséis años a la que se le nota",
      "capability": { "moves_by": ["caminar"], "carry_class": "slight" },
      "senses": { "sight": "normal", "estar en el sueño": "absent" }, "conditions": ["insomne"],
      "disposition": [ { "trait": "resistente", "strength": "defining", "manner": "no se sienta, no cierra los ojos" } ],
      "doing": "llegando a la noche once y empezando a proyectar de día",
      "pursuing": [ { "horizon": "imminent", "toward": "no volver a dormir nunca" } ],
      "hiding": "Qué es lo que no quiere que vea el barrio, que el barrio ya empieza a ver igual." },
    { "name": "Inspector Bald", "facets": ["matter", "agency"], "within": "Barrio Doce", "layer": "vigilia",
      "seen_as": "un hombre de cincuenta y uno con carpeta de la sección sueño ajeno",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "estar en el sueño": "normal" },
      "disposition": [ { "trait": "insistente", "strength": "strong", "manner": "vuelve al mismo local seis años seguidos" } ],
      "doing": "preparando una redada en la Pensión Tarma",
      "pursuing": [ { "horizon": "long_standing", "toward": "cerrar la Pensión Tarma" } ],
      "hiding": "Duerme en el Doce y su carné dice Barrio Siete. Lleva cuatro años en falta contra la ley que aplica." },
    { "name": "Nea Salas", "facets": ["matter", "agency"], "within": "Barrio Doce", "layer": "vigilia",
      "seen_as": "una chica de diecisiete que mira la calle Ruma",
      "capability": { "moves_by": ["caminar"], "carry_class": "slight" },
      "senses": { "sight": "acute", "estar en el sueño": "normal" },
      "disposition": [ { "trait": "callada", "strength": "strong", "manner": "no cuenta dónde durmió" } ],
      "doing": "volviendo del paso al Veintinueve de madrugada",
      "pursuing": [ { "horizon": "long_standing", "toward": "irse de Orbe" } ],
      "hiding": "Durmió tres veces en el Veintinueve y sabe que los fragmentos de ahí son de otras noches." },
    { "name": "Archivista Mayor Ossen", "facets": ["matter", "agency"], "within": "Orbe", "layer": "vigilia",
      "seen_as": "un hombre de sesenta y tres a un año de jubilarse",
      "capability": { "moves_by": ["caminar"], "carry_class": "slight" },
      "senses": { "sight": "faint", "estar en el sueño": "normal" },
      "disposition": [ { "trait": "prolijo", "strength": "defining", "manner": "cuenta antes de opinar" } ],
      "doing": "contando por tercera vez los tomos con noches faltantes",
      "pursuing": [ { "horizon": "imminent", "toward": "jubilarse" } ],
      "hiding": "Son cuarenta y uno, repartidos en seis barrios, todos en los últimos nueve años." },
    { "name": "Solitaria n.º 1", "facets": ["matter", "agency"], "within": "Orbe", "layer": "vigilia",
      "seen_as": "una mujer de setenta y uno sin nombre público",
      "capability": { "moves_by": ["caminar"], "carry_class": "slight" },
      "senses": { "sight": "normal", "estar en el sueño": "acute" }, "conditions": ["solitario"],
      "disposition": [ { "trait": "impasible", "strength": "defining", "manner": "responde con fechas" } ],
      "doing": "esperando que alguien pregunte por la reforma",
      "hiding": "Sabe cómo se seleccionaba a los solitarios antes de la reforma." },

    { "name": "los tomos de transcripción", "facets": ["matter", "record", "magnitude"],
      "within": "Orbe", "layer": "vigilia", "magnitude_class": "twelve thousand",
      "bulk_class": "moderate", "integrity": "sound",
      "asserts": [ { "claim": "lo que soñó cada barrio cada noche", "accurate": false,
                     "plausible_because": "el transcriptor decide qué entra y no hay auditoría ni segunda fuente" } ],
      "access": { "via": "sala de lectura", "delay_class": "one day", "held_by": "la Oficina de Vigilia" },
      "authority": "el transcriptor del barrio" },
    { "name": "el tomo de anoche del Doce", "facets": ["matter", "record"], "within": "Oficina del Doce",
      "layer": "vigilia", "bulk_class": "negligible", "integrity": "sound",
      "asserts": [ { "claim": "qué pasó en el sueño del Doce anoche, y quién estaba parado al lado del cuerpo",
                     "accurate": false, "plausible_because": "Rem sabe qué escribió y qué no" } ],
      "access": { "via": "sala de lectura, a las nueve", "delay_class": "hours", "held_by": "Rem Salas" },
      "authority": "Rem Salas" },
    { "name": "los tomos incompletos", "facets": ["matter", "record", "magnitude"], "within": "Orbe",
      "layer": "vigilia", "magnitude_class": "forty-one", "bulk_class": "moderate", "integrity": "worn",
      "asserts": [ { "claim": "esas noches no ocurrieron", "accurate": false,
                     "plausible_because": "no hay entrada, y nadie cruzó los seis barrios con los nueve años" } ],
      "access": { "via": "Archivo central", "delay_class": "none", "held_by": "Archivista Mayor Ossen" },
      "authority": "el Archivo central",
      "note": "// BREAK B3: una noche faltante es la ausencia de una afirmación, no una afirmación" },
    { "name": "la lista de Rem", "facets": ["matter", "record"], "within": "Rem Salas", "layer": "vigilia",
      "bulk_class": "negligible", "integrity": "sound",
      "asserts": [ { "claim": "once omisiones, once nombres", "accurate": true } ],
      "access": { "via": "estar en su casa", "delay_class": "none", "held_by": "Rem Salas" },
      "authority": "ninguna: no tiene razón administrativa para existir" },
    { "name": "los carnés de barrio", "facets": ["matter", "record", "magnitude"], "within": "Orbe",
      "layer": "vigilia", "magnitude_class": "four hundred thousand", "bulk_class": "negligible",
      "integrity": "sound",
      "asserts": [ { "claim": "dónde te toca dormir", "accurate": true } ],
      "access": { "via": "se pide en pensiones", "delay_class": "none", "held_by": "cada habitante" },
      "authority": "la Oficina de Vigilia", "note": "casi nunca se pide" }
  ],

  "offices": [
    { "name": "Transcriptor del barrio", "held_by": "Rem Salas", "of": "la Oficina de Vigilia",
      "confers": [ { "act": "decidir qué entra en el tomo" }, { "act": "archivar al mediodía" } ],
      "succeeds_by": "designación de la Oficina",
      "note": "sin auditoría, sin segunda fuente, sin control cruzado" },
    { "name": "Solitario registrado", "held_by": "Onel", "of": "el Registro de Solitarios",
      "confers": [ { "act": "testificar en juicio con prueba plena" } ],
      "succeeds_by": "registro forzoso; no se puede renunciar" },
    { "name": "Inspector de sueño ajeno", "held_by": "Inspector Bald", "of": "la Oficina de Vigilia",
      "confers": [ { "act": "abrir expediente y publicarlo" }, { "act": "pedir el carné" } ],
      "succeeds_by": "designación" },
    { "name": "Archivista Mayor", "held_by": "Archivista Mayor Ossen", "of": "la Oficina de Vigilia",
      "confers": [ { "act": "acceder a cualquier tomo de cualquier barrio" } ], "succeeds_by": "antigüedad" }
  ],

  "standing": [
    { "from": "Barrio Doce", "toward": "el personaje del usuario",
      "stance": "cuarenta mil personas te vieron parado al lado del cuerpo y nadie te va a decir nada",
      "since": "el-sueno-del-asesinato", "carried_by": "la mirada del mercado", "persistence": "never decays" },
    { "from": "Rem Salas", "toward": "el personaje del usuario",
      "stance": "sabe qué escribió y qué no, y todavía no decidió qué hacer con eso",
      "since": "el-sueno-del-asesinato", "carried_by": null, "persistence": "until changed" },
    { "from": "la Oficina de Vigilia", "toward": "los solitarios",
      "stance": "los necesita como prueba plena y no les concede la baja",
      "since": null, "carried_by": null, "persistence": "until changed" },
    { "from": "el barrio", "toward": "Vira Cor",
      "stance": "ya empieza a ver lo que ella no quería que nadie viera",
      "since": null, "carried_by": "la mirada del mercado", "persistence": "never decays" }
  ],

  "opposition": [
    { "between": ["el personaje del usuario", "Rem Salas"],
      "incompatible": "El tomo no puede leerse antes de las nueve y no puede corregirse después.",
      "stakes": "Lo que cuarenta mil personas van a creer que hiciste." },
    { "between": ["Onel", "el Registro de Solitarios"],
      "incompatible": "No puede dar la baja y seguir teniendo prueba plena.",
      "stakes": "Diecinueve años y catorce solicitudes." },
    { "between": ["Vira Cor", "el Barrio Doce"],
      "incompatible": "No dormir es lo único que la protege y es lo que la expone.",
      "stakes": "La noche catorce es el colapso." },
    { "between": ["Inspector Bald", "Pensión Tarma"],
      "incompatible": "No puede cerrarla sin que aparezca su propio nombre en el libro de huéspedes.",
      "stakes": "Cuatro años en falta contra la ley que aplica." } ],

  "processes": [
    { "name": "acumulación de insomnio", "acts_on": "cualquier entidad con la condición insomne",
      "direction": "degrade", "rate_class": "nightly", "terminus": "colapso en la noche catorce",
      "note": "no se revierte solo; dormir significa exponerse a todo lo acumulado de golpe" },
    { "name": "borrado de la memoria propia", "acts_on": "cualquier durmiente",
      "direction": "degrade", "rate_class": "nightly", "terminus": "al cuarto día no queda nada propio" }
  ],

  "cycles": [
    { "name": "la noche del barrio", "period_class": "daily", "starts_in_phase": "apertura",
      "phases": [
        { "name": "apertura", "changes": [ { "entity": "el sueño del Doce", "becomes": "existe" } ] },
        { "name": "sueño", "changes": [ { "tension_of": "el sueño del Doce", "becomes": "tense" } ] },
        { "name": "campana de las cinco", "changes": [ { "entity": "el sueño del Doce", "becomes": "cortado de golpe" } ] },
        { "name": "mercado de la mañana", "changes": [ { "tension_of": "Barrio Doce", "becomes": "tense" } ] },
        { "name": "las nueve", "changes": [ { "entity": "el tomo de anoche del Doce", "becomes": "público" } ] } ] }
  ],

  "accumulators": [
    { "name": "noches sin dormir", "per": "each entity with agency", "starts_at": "none",
      "stated": "Cuántas noches seguidas lleva alguien sin dormir.",
      "raised_by": [ { "event": "una noche sin dormir" } ],
      "thresholds": [
        { "at": "tres", "then": "normal" },
        { "at": "cuatro", "then": "primeras proyecciones diurnas, breves" },
        { "at": "cinco a ocho", "then": "proyecciones frecuentes, visibles a veinte metros" },
        { "at": "nueve a trece", "then": "proyección continua: el barrio ve lo que la persona piensa" },
        { "at": "catorce", "then": "colapso", "irreversible": true } ] },
    { "name": "durmientes del barrio esta noche", "per": "each entity with extent",
      "starts_at": "none", "stated": "Qué proporción del barrio está durmiendo ahora.",
      "raised_by": [ { "aggregate_of": "durmientes", "over": "todo el que duerme en el barrio" } ],
      "thresholds": [ { "at": "treinta por ciento", "then": "el sueño arranca" },
                      { "at": "diez por ciento", "then": "el sueño termina" } ],
      "note": "// BREAK B2: un umbral que va en los dos sentidos y que gobierna la EXISTENCIA de una entidad" },
    { "name": "omisiones de un transcriptor", "per": "each office", "starts_at": "moderate",
      "stated": "Cuántas veces un transcriptor dejó algo afuera. Rem lleva once.",
      "raised_by": [ { "event": "una omisión" } ],
      "thresholds": [ { "at": "high", "then": "la lista existe y es una prueba contra quien la escribió" } ] },
    { "name": "noches faltantes en el Archivo", "per": "world", "starts_at": "moderate",
      "stated": "Cuántas noches no tienen tomo. Van cuarenta y una en seis barrios en nueve años.",
      "raised_by": [ { "event": "una noche sin entrada" } ],
      "thresholds": [ { "at": "high", "then": "el patrón deja de poder atribuirse a descuido" } ] }
  ],

  "indicators": [
    { "of": "cuántas noches lleva alguien sin dormir",
      "shows_as": ["proyecciones breves", "proyecciones visibles a veinte metros", "proyección continua"],
      "read_by": { "channel": "sight", "requires": {} }, "reliability_class": "good" },
    { "of": "qué vio realmente un solitario",
      "shows_as": ["lo que declara, y nada más"],
      "read_by": { "channel": "sight", "requires": { "office": "Inspector de sueño ajeno" } },
      "reliability_class": "poor",
      "note": "Onel puede seguir a una persona toda la noche y no lo declara" },
    { "of": "qué omitió un tomo",
      "shows_as": ["nada en el propio tomo", "una lista en la casa de quien lo escribió"],
      "read_by": { "channel": "el tomo", "requires": { "office": "Archivista Mayor" } },
      "reliability_class": "none" },
    { "of": "si alguien durmió en el barrio equivocado",
      "shows_as": ["aparecer en un sueño que no es el propio", "el libro de huéspedes de una pensión"],
      "read_by": { "channel": "estar en el sueño", "requires": {} }, "reliability_class": "good" }
  ],

  "traces": [
    { "of": "una noche en el sueño", "leaves": "tres noches de memoria propia y un tomo permanente", "ages": "la memoria a los tres días; el tomo nunca" },
    { "of": "haber deseado algo delante del barrio", "leaves": "la mirada del mercado a la mañana siguiente", "ages": "never" },
    { "of": "una noche sin tomo", "leaves": "un hueco en el Archivo central", "ages": "never" }
  ],

  "epochs": [
    { "name": "antes de la reforma del Registro",
      "differed": [ { "topic": "office", "subject": "Solitario registrado", "then": "se seleccionaba de otra manera" } ],
      "surviving_traces": ["Solitaria n.º 1, registrada desde los doce años"] },
    { "name": "los ochenta años de estudio",
      "differed": [ { "topic": "law", "subject": "un barrio, un sueño", "then": "se investigaba por qué" } ],
      "surviving_traces": ["que nadie sabe por qué el fenómeno es urbano", "que se dejó de estudiar"] }
  ],

  "history": [
    { "name": "el-sueno-del-asesinato", "standing": "disputed",
      "what_happened": "El barrio soñó un asesinato en un lugar que no es Orbe, con un muerto que nadie reconoce, y cuarenta mil personas vieron al usuario parado al lado del cuerpo.",
      "where": "el sueño del Doce", "who": ["el personaje del usuario", "Rem Salas", "Onel"],
      "knowledge": [
        { "holder": "el personaje del usuario", "channel": "estar en el sueño", "path": "direct",
          "believes": "Estuvo ahí y no recuerda haber hecho nada." },
        { "holder": "Barrio Doce", "channel": "estar en el sueño", "path": "direct",
          "believes": "Lo vieron parado al lado del cuerpo, mirando, quieto." },
        { "holder": "Rem Salas", "channel": "estar en el sueño", "path": "direct",
          "believes": "Sabe qué escribió y qué dejó afuera." },
        { "holder": "Onel", "channel": "estar en el sueño", "path": "direct",
          "believes": "Vio el sueño completo y con más detalle del que va a declarar." },
        { "holder": "Consejo de Orbe", "channel": "el tomo", "path": "told",
          "believes": "Si el sueño predijo, la regla siete es falsa.", "accurate": false,
          "plausible_because": "la otra hipótesis —que un recuerdo cruzó— es peor y nadie la ha formulado" } ] },
    { "name": "la-misma-escena-en-el-Siete", "standing": "disputed",
      "what_happened": "La misma escena apareció en el Barrio Siete tres noches después.",
      "where": "Orbe", "who": [],
      "knowledge": [ { "holder": "Archivista Mayor Ossen", "channel": "el tomo", "path": "direct",
                       "believes": "Distinto barrio, mismo sueño, y eso no debería poder ocurrir." } ] }
  ],

  "arrivals": [
    { "premise": "Barrio Doce, treinta y un años, sin antecedentes, nunca apareciste en un tomo por nada relevante. Anoche cuarenta mil personas te vieron parado al lado de un cuerpo y el tomo es público a las nueve.",
      "seen_as": "alguien corriente al que en el mercado nadie mira a los ojos",
      "place": "Barrio Doce",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "estar en el sueño": "normal", "el tomo": "normal" } },
    { "premise": "Venís de otro barrio y pagaste una cama en la Pensión Tarma. Anoche entraste en un sueño que no es el tuyo y alguien te vio.",
      "seen_as": "alguien con un carné que no conviene mostrar", "place": "Pensión Tarma",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "estar en el sueño": "normal" } },
    { "premise": "Sos solitario registrado de otro barrio y te citaron como testigo en el Doce. Ves todo y no apareces en nada.",
      "seen_as": "alguien a quien nadie le pregunta qué vio", "place": "Oficina del Doce",
      "capability": { "moves_by": ["caminar"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "estar en el sueño": "acute" } }
  ]
}
```

## 2. Facet lists and hesitations

`extent` alone: Orbe, Barrio Doce, sala de lectura, Barrio Veintinueve, el campo. `extent + matter`: Oficina del Doce, Pensión Tarma, casa de Onel. `matter + passage`: paso al Veintinueve. `matter`: Campanario, el cuerpo. **`extent + demand`: el sueño del Doce, el sueño del Veintinueve** — the load-bearing choice of the whole encoding. `agency + collective`: Oficina de Vigilia, Consejo. `agency + collective + record`: Registro de Solitarios. `agency + magnitude`: los Insomnes, los solitarios. `matter + agency`: all seven named people. `matter + record` (+`magnitude`): the five document entities.

Where I hesitated:

- **The dream itself.** `extent + demand` treats it as a place that needs sleepers to keep going. Defensible alternatives: `extent + motion` (it appears and vanishes nightly), or not an entity at all — a *layer* that opens and closes. I took `demand` because "it requires something to keep going" is the closest facet gloss, but `demands[].unmet` describes *starvation*, not *non-existence*.
- **Los solitarios / los Insomnes: `collective` or not?** Both have `interest` and `vulnerability` — which the table lists under `collective` — but the Insomnes are explicitly *"sin organización"*. I gave both `magnitude + agency` and wrote `interest`/`vulnerability` anyway, which means I used collective keys without the facet.
- **Rem's list and the carnés.** `record` on an entity that is also `magnitude` (400.000 carnés): does `asserts[]` describe each card or the set?
- **The body in the dream.** `matter` inside a `medium` that `resists: lastimar totally`. A corpse in a world where nobody can be hurt — I could not tell whether it should have `matter` at all.

## 3. Breaks

**B1 — A layer cannot invert the epistemic default. (i) inexpressible.** *"Lo que uno desea se manifiesta a su alrededor, visible para todos"* and *"Nadie se oculta adentro"* (II.2, III). `hiding` is an `agency` key that exists unconditionally; the dream layer makes it inoperative for every agent inside it. `conditions[].alters` has `broadcast`, but that is per-condition — I would need to attach a condition to all 400.000 people. This is the world's central mechanic and it lives in `law[].stated` as prose the engine cannot act on.

**B2 — Existence conditioned on an aggregate, with hysteresis. (iii) wrong-shaped.** *"El sueño arranca cuando duerme el 30% del barrio. Termina bajo el 10%"* (III). I wrote it twice — as a `demands[]` entry and as an accumulator with two thresholds — because neither fits: `demand` models starvation, not existence, and `accumulators[].thresholds` is an *ascending ladder*, so a threshold that fires on the way **down** is not expressible. The 30/10 gap is deliberate hysteresis and the schema has no direction key.

**B3 — A missing night. (i) inexpressible.** *"cuarenta y un tomos con noches faltantes"* (VI, VII). Same as Grelda's record gap: `asserts[]` holds claims and the absence of a claim is not one. I encoded *"esas noches no ocurrieron"* as an inaccurate assertion, which is my fabrication.

**B4 — Memory with a lifetime. (iii) wrong-shaped.** *"Memoria propia: tres noches. Al cuarto día se borra"* (II.3). `history[].knowledge` has no expiry, so I put the three nights on the *channel's* `decay`. That says the medium forgets, not the holder — and it cannot express the consequence the brief cares about: *"después solo queda el tomo, y el tomo lo escribió otra persona"*.

**B5 — A layer that leaks across time. (ii) inert prose.** *"se fragmenta, pero los fragmentos son de otras noches"* (VI, Nea). I put it in the entity's `hiding`. Nothing in `layers[]` expresses a relation between layers, let alone a temporal one.

**B6 — Numbers, again.** `"at": "treinta por ciento"` and `"at": "cinco a ocho"` are thresholds the engine must compare on, so under v2's rule they should be classes; but the brief's whole insomnia table *is* the player-facing text. I wrote the figures and flagged them. The rule "numbers only where a player reads" does not resolve a field that is *both*.

Against v1: `record`, `offices`, `indicators`, threshold ladders and `excluded` all genuinely land — this world would have been unencodable in v1. The new breaks are all in `layers[]`, which is the one section that got no facet treatment.

## 4. Ambiguity report

**A1 — 28 dreams: 28 layers, or one layer with 28 entities?** I chose one layer (`el sueño del barrio`) containing per-barrio dream entities. The other reading — one layer per barrio — is equally supported, since a layer is a "rule-reality" and each dream is causally sealed from the others. Consequence: under my reading, "the same scene appeared in Barrio Siete" is two entities in one layer; under the other it is a cross-layer leak. **The most likely divergence between two encoders, and it changes the shape of the anomaly at the centre of the plot.** Disambiguate: state whether a layer is a *rule-set* or a *causal container*.

**A2 — Three-night memory: channel `decay`, a `process`, or a knowledge property?** All three are expressible. I used channel `decay` *and* a process, which is duplication I could not avoid.

**A3 — Where does the Veintinueve live?** I gave it both a `layer` ("el sueño roto") and an `extent` entity in vigilia. It is equally defensible as one entity with a strange `medium` and no third layer at all — which is what the basic tier forces (see §5).

**A4 — `collective` without organisation.** The Insomnes have an interest and a weakness but *"sin organización"*. I used `magnitude + agency` and wrote collective keys anyway. Nothing says whether `interest`/`vulnerability` require the facet.

**A5 — Solitario: a `condition` or an `office`?** The brief gives it both faces — a physiological fact (dreams alone, invisible) and a registry post with duties you cannot resign. I encoded *both*, which double-counts. Disambiguate: say whether a fact about a body may also be an office.

**A6 — `standing` vs `accumulator` for the market's gaze.** *"Se castiga en el mercado, al día siguiente, con la mirada"* — a standing carried by a channel, or an accumulator that rises with each exposure? I used standing.

**A7 — Accumulator `per` scope, again.** `per: "each office"` (Rem's omissions) and `per: "each entity with extent"` (sleepers per barrio) are my inventions. Same defect as in Grelda: the scope grammar is unspecified.

## 5. Convergence check

**No new top-level section is needed** — but `layers[]` needs the facet treatment the nouns got. Every one of my six breaks except B3 is a layer problem: a layer cannot have an existence condition, cannot suppress a facet key, and cannot relate to another layer. A layer is an entity (it has `extent`, it has `demand`) and it should be one.

## 6. Basic-tier diff — subset or different document?

I encoded `1-basico` as well. **It is a clean subset in 15 of 17 sections, and structurally different in two.**

Subset, straightforwardly: `world`, `vocabulary` (same media, one fewer channel — no `el tomo` detail), `law` (all six hard rules and all four soft ones are present in basic, with less text), `entities` (Orbe, Barrio Doce, Oficina, the dream, and four people instead of seven), `offices` (transcriptor, solitario, inspector), `standing`, `opposition`, `indicators`, `traces`, `history` (the murder only), `arrivals`, `excluded`. Nothing in the basic tier needed a key the detailed tier did not also use. Sparse authors *less*, not *worse* — the tier ladder works.

The two structural differences:

1. **`layers[]` collapses from three to two.** Barrio Veintinueve does not exist in the basic tier, so there is no second dream-reality and no third layer. Under my A1 reading that is just a row; under the other reading (one layer per barrio) the basic tier has 28 and the detailed 29. Either way, the *machinery* of multiple non-default layers is exercised only by detail.
2. **The dream's existence mechanism migrates section.** Basic says only *"un barrio, un sueño, todas las noches"* — a `cycle` phase, nothing else. Detailed adds the 30%/10% thresholds, which forced me into `demands[]` **and** an `accumulator`. So the same fact is a cycle at tier 1 and a demand-plus-accumulator at tier 3. That is detail read as **kind**, not depth, and it is the defect signal we agreed to watch for — though narrowly: one fact, two sections, caused by the same weakness B2 names.

Everything else scales by row count. My verdict: the tier ladder is sound; the leak is not in the entity model but in the sections that never received facets — `layers[]` above all.
