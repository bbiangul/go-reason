package eval

// ALTAVisionEasyDataset returns 30 easy (single-fact lookup) test cases
// from the ALTAVision AV-FM/AV-FF technical manual.
//
// Expected facts use pipe-separated alternatives (e.g. "Spanish|English")
// so accuracy scoring works regardless of the LLM's answer language.
func ALTAVisionEasyDataset() Dataset {
	return Dataset{
		Name:       "ALTAVision Easy - Single Fact Lookup",
		Difficulty: DifficultyEasy,
		Tests: []TestCase{
			{
				Question:      "What is the operating temperature range?",
				ExpectedFacts: []string{"5°|5 degree|5 °", "40°|40 degree|40 °"},
				Category:      "specs",
				Explanation:   "[p28] Condiciones ambientales: 'Temperatura: 5° a 40° Celsius'.",
			},
			{
				Question:      "What is the part number of the Tracker board?",
				ExpectedFacts: []string{"E1375"},
				Category:      "components",
				Explanation:   "[p45] Tabla de componentes del Tracker: 'Número de parte: E1375'.",
			},
			{
				Question:      "What voltage does the equipment operate at?",
				ExpectedFacts: []string{"120", "240"},
				Category:      "specs",
				Explanation:   "[p28] Especificaciones eléctricas: '120 VCA o 240 VCA, monofásico'.",
			},
			{
				Question:      "What is the noise emission level?",
				ExpectedFacts: []string{"70", "dB"},
				Category:      "specs",
				Explanation:   "[p30] Emisiones de ruido: 'Menos de 70 dB(A) medido a 1 metro'.",
			},
			{
				Question:      "What is the air pressure requirement?",
				ExpectedFacts: []string{"75", "85", "PSIG|psig|bar"},
				Category:      "specs",
				Explanation:   "[p31] Requisitos de aire comprimido: '75 – 85 PSIG'.",
			},
			{
				Question:      "What material is the inspection bridge made of?",
				ExpectedFacts: []string{"inoxidable|stainless steel|stainless", "304"},
				Category:      "specs",
				Explanation:   "[p32] Materiales: 'Acero inoxidable 304'.",
			},
			{
				Question:      "What is the weight of Model A Standard?",
				ExpectedFacts: []string{"153"},
				Category:      "specs",
				Explanation:   "[p32] Tabla de especificaciones Modelo A: 'Peso: 153 kg'.",
			},
			{
				Question:      "What is the IP protection rating?",
				ExpectedFacts: []string{"IP54"},
				Category:      "specs",
				Explanation:   "[p28] Grado de protección: 'IP54'.",
			},
			{
				Question:      "What THD level is required?",
				ExpectedFacts: []string{"5%"},
				Category:      "specs",
				Explanation:   "[p29] Calidad de energía: 'THD menor o igual al 5%'.",
			},
			{
				Question:      "What is the Tracker board IP address?",
				ExpectedFacts: []string{"68.178.1.11"},
				Category:      "components",
				Explanation:   "[p51] Configuración de red del Tracker: 'IP: 68.178.1.11'.",
			},
			{
				Question:      "What is the subnet mask of the Tracker?",
				ExpectedFacts: []string{"255.255.255.0"},
				Category:      "components",
				Explanation:   "[p51] Configuración de red del Tracker: 'Máscara de subred: 255.255.255.0'.",
			},
			{
				Question:      "What fuse rating protects the Tracker outputs?",
				ExpectedFacts: []string{"6.3", "250V|250 V"},
				Category:      "components",
				Explanation:   "[p47] Protección del Tracker: 'Fusible de 6.3A, 250V'.",
			},
			{
				Question:      "What processor does the CPU CUBE use?",
				ExpectedFacts: []string{"Intel", "i7"},
				Category:      "components",
				Explanation:   "[p49] CPU CUBE: 'Procesador Intel Core i7'.",
			},
			{
				Question:      "What operating system is preinstalled?",
				ExpectedFacts: []string{"Windows", "10"},
				Category:      "components",
				Explanation:   "[p49] Sistema operativo: 'Windows 10'.",
			},
			{
				Question:      "What is the encoder part number?",
				ExpectedFacts: []string{"E-1306"},
				Category:      "components",
				Explanation:   "[p61] Encoder: 'Número de parte: E-1306'.",
			},
			{
				Question:      "What is the trigger sensor part number?",
				ExpectedFacts: []string{"E-1024"},
				Category:      "components",
				Explanation:   "[p64] Sensor trigger: 'Número de parte: E-1024'.",
			},
			{
				Question:      "What is the standard cap diameter?",
				ExpectedFacts: []string{"28 mm|28mm|28 millimeter"},
				Category:      "specs",
				Explanation:   "[p126] Parámetros del contenedor: 'El diámetro de la tapa estándar es 28 mm'.",
			},
			{
				Question:      "How many inputs does the Tracker board have?",
				ExpectedFacts: []string{"16"},
				Category:      "components",
				Explanation:   "[p45] Tracker board: '16 entradas digitales'.",
			},
			{
				Question:      "How many outputs does the Tracker board have?",
				ExpectedFacts: []string{"16"},
				Category:      "components",
				Explanation:   "[p45] Tracker board: '16 salidas digitales'.",
			},
			{
				Question:      "What is the Tracker power supply voltage?",
				ExpectedFacts: []string{"5 Vdc|5Vdc|5V|+5V|5 V DC|5 V dc"},
				Category:      "components",
				Explanation:   "[p45] Alimentación del Tracker: '5 Vdc'.",
			},
			{
				Question:      "What is the power consumption without AC?",
				ExpectedFacts: []string{"1.2", "KVA|kVA"},
				Category:      "specs",
				Explanation:   "[p28] Consumo sin aire acondicionado: '1.2 KVA'.",
			},
			{
				Question:      "What is the power consumption with AC?",
				ExpectedFacts: []string{"1.7", "KVA|kVA"},
				Category:      "specs",
				Explanation:   "[p28] Consumo con aire acondicionado: '1.7 KVA'.",
			},
			{
				Question:      "What cut protection level is recommended for gloves?",
				ExpectedFacts: []string{"level 5|nivel 5|cut 5|level-5", "EN 388"},
				Category:      "safety",
				Explanation:   "[p25] EPP: 'Guantes con nivel de corte 5 según EN 388'.",
			},
			{
				Question:      "What wire gauge is recommended for electrical cabling?",
				ExpectedFacts: []string{"AWG", "14"},
				Category:      "specs",
				Explanation:   "[p29] Cableado: 'Calibre mínimo 14 AWG'.",
			},
			{
				Question:      "What color is the cabinet painted?",
				ExpectedFacts: []string{"negro|black|matte black"},
				Category:      "specs",
				Explanation:   "[p32] Acabado del gabinete: 'Negro mate'.",
			},
			{
				Question:      "What type of glass is used for protection windows?",
				ExpectedFacts: []string{"templado|tempered"},
				Category:      "specs",
				Explanation:   "[p32] Ventanas de protección: 'Vidrio templado'.",
			},
			{
				Question:      "What is the weight of Model B cabinet?",
				ExpectedFacts: []string{"80 kg|80 kilo|80kg"},
				Category:      "specs",
				Explanation:   "[p34] Modelo B: 'Peso: 80 kg'.",
			},
			{
				Question:      "What is the weight of Model C Standard XL?",
				ExpectedFacts: []string{"300 kg|300 kilo|300kg|300 pound"},
				Category:      "specs",
				Explanation:   "[p36] Modelo C Standard XL: 'Peso: 300 kg'.",
			},
			{
				Question:      "What is the document revision date?",
				ExpectedFacts: []string{"08/2025"},
				Category:      "specs",
				Explanation:   "[p1] Portada: 'Rev2 - 08/2025'.",
			},
			{
				Question:      "How many pulses per revolution is the encoder set to?",
				ExpectedFacts: []string{"550"},
				Category:      "components",
				Explanation:   "[p61] Encoder: '550 pulsos por revolución'.",
			},
		},
	}
}

// ALTAVisionMediumDataset returns 30 medium (multi-fact, context-dependent) test cases.
func ALTAVisionMediumDataset() Dataset {
	return Dataset{
		Name:       "ALTAVision Medium - Multi-fact Retrieval",
		Difficulty: DifficultyMedium,
		Tests: []TestCase{
			{
				Question:      "What are all the inspection types supported by AV-FM/AV-FF?",
				ExpectedFacts: []string{"nivel de llenado|fill level", "tapa|cap", "objeto flotante|floating object"},
				Category:      "multi-fact",
				Explanation:   "[p7-8] Tipos de inspección: nivel de llenado, seguimiento de tapa, objeto flotante. [p70] AV-FF añade inspección de objetos flotantes al AV-FM base.",
			},
			{
				Question:      "What components are tracked by the Tracker board?",
				ExpectedFacts: []string{"encoder", "trigger", "rechazador|rejector|rejection"},
				Category:      "multi-fact",
				Explanation:   "[p45] El Tracker gestiona: encoder, trigger, rechazador, sensores y señalización. [p94-96] Controla la sincronización entre estos subsistemas.",
			},
			{
				Question:      "What are the differences between AV-FM and AV-FF models?",
				ExpectedFacts: []string{"AV-FM", "AV-FF", "objeto flotante|floating object|floater"},
				Category:      "multi-fact",
				Explanation:   "[p7] AV-FM: inspección de nivel de llenado y tapa. AV-FF: agrega inspección de objetos flotantes (floaters) al AV-FM.",
			},
			{
				Question:      "Describe the sequence of steps in the inspection process",
				ExpectedFacts: []string{"trigger", "encoder", "cámara|camera", "rechazo|rejection|reject"},
				Category:      "multi-fact",
				Explanation:   "[p70-73] Secuencia: 1) Trigger detecta botella, 2) Encoder cuenta pulsos, 3) Cámara captura imagen en posición calculada, 4) IPC procesa, 5) Rechazador actúa si hay defecto.",
			},
			{
				Question:      "What user roles exist in the system?",
				ExpectedFacts: []string{"Operator|Operador", "Mantenimiento|Maintenance", "Administrador|Administrator"},
				Category:      "multi-fact",
				Explanation:   "[p80-81] Roles: Operador (monitoreo), Mantenimiento (ajustes), Administrador (configuración completa).",
			},
			{
				Question:      "What are the three SKU states?",
				ExpectedFacts: []string{"Progreso|Progress", "Estático|Static", "Dinámico|Dynamic"},
				Category:      "multi-fact",
				Explanation:   "[p102] Estados de SKU: 'En Progreso' (en edición), 'Ajuste Estático' (sin producción), 'Ajuste Dinámico' (en producción).",
			},
			{
				Question:      "What screens are available in the main menu?",
				ExpectedFacts: []string{"Inicio|Home", "Seguimiento|Tracking", "Alarmas|Alarm"},
				Category:      "GUI",
				Explanation:   "[p78-79] Menú principal: Inicio, Seguimiento, Alarmas, entre otras pantallas de configuración y diagnóstico.",
			},
			{
				Question:      "What are the signal beacon light colors and their meanings?",
				ExpectedFacts: []string{"verde|green", "azul|blue", "rojo|red"},
				Category:      "components",
				Explanation:   "[p59] Baliza de señales: Verde (operación normal), Azul (rechazo activo), Rojo (alarma/falla).",
			},
			{
				Question:      "What safety standards does the equipment comply with?",
				ExpectedFacts: []string{"2006/42/EC", "EN ISO 12100", "IEC60825|IEC 60825"},
				Category:      "safety",
				Explanation:   "[p22-24] Normas: Directiva 2006/42/EC (Maquinaria), EN ISO 12100 (seguridad general), IEC 60825 (seguridad láser).",
			},
			{
				Question:      "What types of cooling systems are available?",
				ExpectedFacts: []string{"vortex|Vortex", "Rittal|air conditioning|aire acondicionado"},
				Category:      "components",
				Explanation:   "[p55-57] Sistemas de refrigeración: Tubo Vortex (aire comprimido) y Rittal (aire acondicionado eléctrico).",
			},
			{
				Question:      "What are the electrical current specifications at 120V?",
				ExpectedFacts: []string{"10 Amp|10A|10 amp", "14 Amp|14A|14 amp"},
				Category:      "specs",
				Explanation:   "[p28] Corriente a 120V: '10A sin AC, 14A con AC'.",
			},
			{
				Question:      "What information is shown in the GUI header?",
				ExpectedFacts: []string{"usuario|user", "SKU", "versión|version"},
				Category:      "GUI",
				Explanation:   "[p78] Encabezado del GUI: muestra usuario actual, SKU activo y versión del software.",
			},
			{
				Question:      "What parameters does the cap tracking tool configure?",
				ExpectedFacts: []string{"Ventana|Window|window width", "Borde|Edge|edge size", "Posición|Position|vertical position"},
				Category:      "multi-fact",
				Explanation:   "[p131-132] Herramienta de seguimiento de tapa: Ventana (ancho/alto), Tamaño de Borde, Posición Vertical.",
			},
			{
				Question:      "What data is recorded by the inspection system?",
				ExpectedFacts: []string{"inspeccionados|inspected|containers inspected", "rechazados|rejected", "defectos|defect"},
				Category:      "multi-fact",
				Explanation:   "[p90-91] Datos registrados: contenedores inspeccionados, rechazados por tipo de defecto, contadores de producción.",
			},
			{
				Question:      "What are the equipment models described in the manual?",
				ExpectedFacts: []string{"AV-FM|AV FM", "AV-FF|AV FF"},
				Category:      "multi-fact",
				Explanation:   "[p7] Modelos: AV-FM (Fill Monitor) y AV-FF (Fill + Floater).",
			},
			{
				Question:      "How does the system handle power failures?",
				ExpectedFacts: []string{"UPS", "apagado|shutdown", "Windows"},
				Category:      "multi-fact",
				Explanation:   "[p42] UPS proporciona energía para apagado seguro de Windows tras corte eléctrico.",
			},
			{
				Question:      "What communication protocols are used between components?",
				ExpectedFacts: []string{"Ethernet", "RS232|RS-232"},
				Category:      "components",
				Explanation:   "[p51] Protocolos: Ethernet (IPC-Tracker), RS232 (comunicaciones serie con dispositivos auxiliares).",
			},
			{
				Question:      "What are the environmental conditions for operation?",
				ExpectedFacts: []string{"5°|5 degree|5 °", "40°|40 degree|40 °", "95%"},
				Category:      "specs",
				Explanation:   "[p28] Condiciones: Temperatura 5°-40°C, humedad relativa máxima 95% sin condensación.",
			},
			{
				Question:      "What personal protective equipment is required?",
				ExpectedFacts: []string{"gafas|glasses|safety glasses|eyewear", "guantes|gloves"},
				Category:      "safety",
				Explanation:   "[p25] EPP requerido: gafas de seguridad, guantes de protección con nivel de corte 5.",
			},
			{
				Question:      "How does the Global Learning function work?",
				ExpectedFacts: []string{"diámetro|diameter", "tapa|cap", "píxeles|pixel"},
				Category:      "multi-fact",
				Explanation:   "[p87] Aprendizaje Global: compara diámetro conocido de tapa (28mm) con píxeles en imagen para calibrar la relación píxel/mm.",
			},
			{
				Question:      "What types of container parameters need configuration?",
				ExpectedFacts: []string{"altura|height", "diámetro|diameter", "tapa|cap"},
				Category:      "multi-fact",
				Explanation:   "[p126] Parámetros del contenedor: altura real, diámetro de la botella, diámetro de tapa.",
			},
			{
				Question:      "What happens when the inactivity timeout is exceeded?",
				ExpectedFacts: []string{"sesión|session", "automáticamente|automatically|automatic"},
				Category:      "GUI",
				Explanation:   "[p117] Timeout: la sesión se cierra automáticamente tras 20 minutos de inactividad.",
			},
			{
				Question:      "What is the grounding specification?",
				ExpectedFacts: []string{"tierra|ground", "1 VCA|1VCA|1 VAC", "14 AWG"},
				Category:      "specs",
				Explanation:   "[p29] Tierra: voltaje entre tierra y neutro menor a 1 VCA, cable mínimo 14 AWG.",
			},
			{
				Question:      "What are the IMAGO outputs used for?",
				ExpectedFacts: []string{"verde|green", "azul|blue", "roja|red"},
				Category:      "components",
				Explanation:   "[p59] Salidas IMAGO para baliza: verde (operando), azul (rechazando), roja (alarma).",
			},
			{
				Question:      "How does the encoder function in the inspection process?",
				ExpectedFacts: []string{"pulsos|pulse", "trigger"},
				Category:      "components",
				Explanation:   "[p61] El encoder genera pulsos proporcionales al movimiento de la línea. El tracker cuenta pulsos desde el trigger para calcular posiciones de cámaras y rechazadores.",
			},
			{
				Question:      "What types of inspections can be configured simultaneously?",
				ExpectedFacts: []string{"cuatro|four|4 inspection", "inspección|inspection"},
				Category:      "multi-fact",
				Explanation:   "[p70] Se pueden configurar hasta cuatro inspecciones simultáneas por contenedor.",
			},
			{
				Question:      "What are the Multicam options for cap opacity?",
				ExpectedFacts: []string{"Suma|Sum", "Máximo|Maximum"},
				Category:      "multi-fact",
				Explanation:   "[p141] Opciones Multicam para opacidad de tapa: Suma (acumula valores) y Máximo (toma el mayor).",
			},
			{
				Question:      "What laser sensor configuration is used in the Profiler?",
				ExpectedFacts: []string{"450nm|450 nm", "192.168.178.100"},
				Category:      "components",
				Explanation:   "[p62] Profiler: láser de 450nm, IP 192.168.178.100.",
			},
			{
				Question:      "What causes a bottle to be rejected in fill level inspection?",
				ExpectedFacts: []string{"nivel de llenado|fill level", "límite|limit|threshold"},
				Category:      "multi-fact",
				Explanation:   "[p166-168] Rechazo por nivel de llenado: cuando el nivel medido excede los límites superior o inferior configurados.",
			},
			{
				Question:      "What are the steps to copy a SKU?",
				ExpectedFacts: []string{"copiar|copy", "confirmación|confirmation|confirm"},
				Category:      "GUI",
				Explanation:   "[p102] Copiar SKU: seleccionar SKU origen, usar función Copiar, confirmar. Nota: SKUs en estado 'En Progreso' no pueden ser copiados.",
			},
		},
	}
}

// ALTAVisionHardDataset returns 30 hard (multi-hop reasoning) test cases.
func ALTAVisionHardDataset() Dataset {
	return Dataset{
		Name:       "ALTAVision Hard - Multi-hop Reasoning",
		Difficulty: DifficultyHard,
		Tests: []TestCase{
			{
				Question:      "How does the tracker board coordinate with both the encoder and the rejection system?",
				ExpectedFacts: []string{"pulsos|pulse", "encoder", "rechazador|rejector|rejection"},
				Category:      "multi-hop",
				Explanation:   "[p45] Tracker board gestiona subsistemas. [p61] Encoder genera pulsos. [p94-96] Tracker cuenta pulsos de encoder desde trigger hasta rechazador para calcular momento exacto de rechazo.",
			},
			{
				Question:      "Explain the relationship between the Profiler sensor, camera system, and IPC in image processing",
				ExpectedFacts: []string{"láser|laser", "Profiler", "cámara|camera", "IPC"},
				Category:      "multi-hop",
				Explanation:   "[p62] Profiler usa láser 3D. [p53] Cámaras capturan imágenes 2D. [p49] IPC procesa ambas fuentes. El Profiler complementa las cámaras con datos de profundidad para inspección de nivel.",
			},
			{
				Question:      "What safety measures protect against electrical hazards during maintenance?",
				ExpectedFacts: []string{"tierra|ground", "interruptor|switch|breaker", "fusibles|fuse"},
				Category:      "safety",
				Explanation:   "[p29] Tierra <1VCA. [p40] Interruptor principal para desconexión. [p47] Fusibles protegen circuitos del Tracker. Conjunto forma protección eléctrica multicapa.",
			},
			{
				Question:      "How do the cap tracking and fill level inspection tools work together?",
				ExpectedFacts: []string{"seguimiento|tracking", "tapa|cap", "nivel de llenado|fill level"},
				Category:      "multi-hop",
				Explanation:   "[p131-132] Seguimiento de tapa localiza la tapa. [p166] Inspección de nivel mide altura del líquido. Ambos usan la misma imagen pero diferentes ventanas de detección.",
			},
			{
				Question:      "Compare the hardware differences between CPU CUBE and CPU IPC models",
				ExpectedFacts: []string{"CUBE", "Fanless|fanless", "IPC"},
				Category:      "multi-hop",
				Explanation:   "[p49] CUBE: Intel i7, fanless, compacto. [p50] IPC: rack mount, mayor capacidad de expansión. CUBE para instalaciones estándar, IPC para configuraciones complejas.",
			},
			{
				Question:      "Describe the complete data flow from bottle detection to rejection",
				ExpectedFacts: []string{"trigger", "tracker", "encoder", "cámara|camera", "rechazo|rejection|reject"},
				Category:      "multi-hop",
				Explanation:   "[p64] Trigger detecta botella. [p45] Tracker registra posición. [p61] Encoder mide avance. [p53] Cámara captura en posición correcta. [p94-96] Tracker activa rechazo tras distancia configurada.",
			},
			{
				Question:      "What are all the conditions that would void the manufacturer warranty?",
				ExpectedFacts: []string{"modificación|modification", "ALTAVision"},
				Category:      "multi-hop",
				Explanation:   "[p20-21] Garantía se anula por: modificaciones no autorizadas, uso fuera de especificaciones, falta de mantenimiento según manual ALTAVision.",
			},
			{
				Question:      "Explain how the foam/opacity inspection relates to fill level inspection",
				ExpectedFacts: []string{"nivel de llenado|fill level", "porcentaje|percentage", "espuma|foam"},
				Category:      "multi-hop",
				Explanation:   "[p166] Nivel de llenado mide altura del líquido. [p196-198] Espuma/opacidad evalúa porcentaje de píxeles oscuros en ventana. La espuma afecta la medición de nivel, por eso existe compensación de espuma.",
			},
			{
				Question:      "How does the system adapt to different bottle sizes when switching SKUs?",
				ExpectedFacts: []string{"Aprendizaje Global|Global Learning", "diámetro|diameter", "contenedor|container"},
				Category:      "multi-hop",
				Explanation:   "[p87] Aprendizaje Global recalibra píxel/mm usando diámetro de tapa. [p126] Contenedor almacena dimensiones. Al cambiar SKU, el sistema carga nuevo contenedor y puede requerir nuevo aprendizaje.",
			},
			{
				Question:      "What are all the network connections and their IP addresses?",
				ExpectedFacts: []string{"68.178.1.11", "192.168.178.100", "Ethernet"},
				Category:      "multi-hop",
				Explanation:   "[p51] Tracker: 68.178.1.11 vía Ethernet. [p62] Profiler: 192.168.178.100. Todos conectados al IPC por red Ethernet interna.",
			},
			{
				Question:      "Describe the role-based access control system",
				ExpectedFacts: []string{"Operator|Operador", "Mantenimiento|Maintenance", "Administrador|Administrator", "permisos|permission"},
				Category:      "multi-hop",
				Explanation:   "[p80-81] Tres roles con permisos crecientes: Operador (solo monitoreo), Mantenimiento (ajustes de producción), Administrador (configuración completa del sistema).",
			},
			{
				Question:      "How does the auxiliary inspection detect whether a bottle is full or empty?",
				ExpectedFacts: []string{"escala de grises|grayscale|gray scale", "píxeles|pixel"},
				Category:      "multi-hop",
				Explanation:   "[p196-198] Inspección auxiliar evalúa escala de grises en ventana de inspección. Cuenta píxeles por debajo del umbral de opacidad para determinar contenido.",
			},
			{
				Question:      "What happens during the machine startup sequence after a power failure?",
				ExpectedFacts: []string{"UPS", "interruptor|switch|power switch"},
				Category:      "multi-hop",
				Explanation:   "[p42] UPS mantiene energía. Tras falla, encender interruptor principal, esperar arranque de Windows, software se inicia automáticamente.",
			},
			{
				Question:      "Explain the code percentage inspection and when a bottle gets rejected",
				ExpectedFacts: []string{"porcentaje|percentage", "píxeles|pixel", "1500"},
				Category:      "multi-hop",
				Explanation:   "[p190-192] Inspección de código: cuenta píxeles que cumplen criterio. Umbral de 1500 píxeles típico. Si porcentaje está fuera de rango, botella se rechaza.",
			},
			{
				Question:      "Compare the floating object and foam/opacity inspections",
				ExpectedFacts: []string{"opacidad|opacity", "objeto flotante|floating object", "escala de grises|grayscale|gray scale"},
				Category:      "multi-hop",
				Explanation:   "[p196-198] Opacidad mide porcentaje de píxeles oscuros en escala de grises. [p200-202] Objetos flotantes buscan partículas en la zona del líquido. Ambos usan análisis de imagen pero con ventanas y criterios diferentes.",
			},
			{
				Question:      "How do environmental conditions affect the cooling system choice?",
				ExpectedFacts: []string{"vortex|Vortex", "Rittal|air conditioning"},
				Category:      "multi-hop",
				Explanation:   "[p55] Vortex usa aire comprimido, adecuado para ambientes con aire disponible. [p57] Rittal AC para ambientes de alta temperatura donde el Vortex no alcanza. Elección depende de temperatura ambiente y disponibilidad de aire.",
			},
			{
				Question:      "What are the complete steps to configure a new product for inspection?",
				ExpectedFacts: []string{"SKU", "contenedor|container", "Aprendizaje Global|Global Learning"},
				Category:      "multi-hop",
				Explanation:   "[p100] Crear nuevo SKU. [p126] Configurar contenedor con dimensiones. [p87] Ejecutar Aprendizaje Global para calibrar. Luego configurar herramientas de inspección específicas.",
			},
			{
				Question:      "How does the system prevent unauthorized access and accidental modifications?",
				ExpectedFacts: []string{"sesión|session", "roles|role", "permisos|permission"},
				Category:      "multi-hop",
				Explanation:   "[p80-81] Control por roles con permisos diferenciados. [p117] Sesión se cierra automáticamente tras 20 min de inactividad. Cada acción verifica permisos del rol actual.",
			},
			{
				Question:      "What are all the risk group classifications and their safety requirements?",
				ExpectedFacts: []string{"grupo de riesgo|risk group", "IEC 62471", "gafas|glasses|safety glasses"},
				Category:      "safety",
				Explanation:   "[p24] Clasificación según IEC 62471 para emisiones ópticas. Grupos de riesgo determinan EPP requerido, incluyendo gafas de seguridad.",
			},
			{
				Question:      "Explain how the cylindrical transformation tool works with label inspection",
				ExpectedFacts: []string{"cilíndrica|cylindrical", "etiquetas|label"},
				Category:      "multi-hop",
				Explanation:   "[p186-188] Transformación cilíndrica desenvuelve la imagen curva de la botella para inspección plana de etiquetas. Corrige distorsión causada por la curvatura del envase.",
			},
			{
				Question:      "How do the UPS and noise filter protect the system?",
				ExpectedFacts: []string{"UPS", "filtro de ruido|noise filter"},
				Category:      "multi-hop",
				Explanation:   "[p42] UPS protege contra cortes de energía con apagado seguro. [p43] Filtro de ruido elimina interferencias eléctricas que podrían afectar componentes electrónicos sensibles.",
			},
			{
				Question:      "What is the relationship between tracking parameters and inspection accuracy?",
				ExpectedFacts: []string{"seguimiento|tracking", "tapa|cap", "precisión|precision|accuracy"},
				Category:      "multi-hop",
				Explanation:   "[p131-132] Parámetros de seguimiento de tapa definen ventana de detección. [p94-96] Precisión depende de correcta calibración de distancias en pulsos. Tracking incorrecto = imagen descentrada = baja precisión.",
			},
			{
				Question:      "Describe all alarm classes and their significance",
				ExpectedFacts: []string{"alarma|alarm", "advertencia|warning|yellow", "crítica|critical|red"},
				Category:      "multi-hop",
				Explanation:   "[p105] Tres clases: Tipo 1 Informativas (verde), Tipo 2 Advertencias (amarillo, requieren verificación), Tipo 3 Críticas (rojo, requieren acción correctiva inmediata).",
			},
			{
				Question:      "How does the RGB learning feature calibrate color inspections?",
				ExpectedFacts: []string{"RGB", "100 botellas|100 bottles"},
				Category:      "multi-hop",
				Explanation:   "[p148-150] Aprendizaje RGB analiza 100 botellas para establecer valores de referencia de color en cada canal R, G, B. Define umbrales de aceptación/rechazo.",
			},
			{
				Question:      "What determines whether bottles with the same size can share a container configuration?",
				ExpectedFacts: []string{"altura|height", "diámetro|diameter", "tapa|cap", "contenedor|container"},
				Category:      "multi-hop",
				Explanation:   "[p126] Contenedor define: altura real, diámetro de botella, diámetro de tapa. Botellas pueden compartir contenedor si estas tres dimensiones coinciden, aunque los SKUs sean diferentes.",
			},
			{
				Question:      "Explain the relationship between encoder calibration and rejection timing",
				ExpectedFacts: []string{"encoder", "pulsos|pulse", "rechazador|rejector|rejection"},
				Category:      "multi-hop",
				Explanation:   "[p61] Encoder genera pulsos por revolución. [p94-96] Distancia al rechazador se configura en pulsos de encoder. Calibración incorrecta del encoder causa timing incorrecto del rechazo.",
			},
			{
				Question:      "How does the multi-camera system consolidate inspection results?",
				ExpectedFacts: []string{"Multicam", "Suma|Sum", "Máximo|Maximum"},
				Category:      "multi-hop",
				Explanation:   "[p141] Multicam consolida resultados de múltiples cámaras con dos métodos: Suma (acumula valores de todas las cámaras) o Máximo (usa el valor más alto entre cámaras).",
			},
			{
				Question:      "What maintenance activities require factory-level permissions?",
				ExpectedFacts: []string{"Configuración|Configuration|Factory"},
				Category:      "multi-hop",
				Explanation:   "[p80-81] Actividades de nivel fábrica: calibración de hardware, actualización de firmware del Tracker, configuración de red. Requieren nivel Administrador o superior.",
			},
			{
				Question:      "How does the system handle transparent containers vs opaque containers?",
				ExpectedFacts: []string{"trigger", "retroreflectante|retroreflective", "transparentes|transparent"},
				Category:      "multi-hop",
				Explanation:   "[p64] Para contenedores transparentes se usa trigger retroreflectante (la botella interrumpe el haz). Para opacos se puede usar trigger directo.",
			},
			{
				Question:      "Explain the complete workflow for commissioning a new ALTAVision system",
				ExpectedFacts: []string{"transporte|transport", "montaje|mounting|assembly|installation", "tierra|ground"},
				Category:      "multi-hop",
				Explanation:   "[p38-39] Transporte cuidadoso del equipo. [p40] Montaje en línea de producción. [p29] Verificar conexión a tierra. Luego configuración de software, aprendizaje global, y validación.",
			},
		},
	}
}

// ALTAVisionSuperHardDataset returns 50 super-hard (synthesis/inference) test cases.
// Includes the original 30 (with fixes to Q2, Q19, Q25, Q30) plus 20 new tests
// in categories: graph-multi-hop, anti-hallucination, numerical, reasoning.
func ALTAVisionSuperHardDataset() Dataset {
	return Dataset{
		Name:       "ALTAVision Super Hard - Synthesis & Inference",
		Difficulty: DifficultySuperHard,
		Tests: []TestCase{
			// --- Original 30 (with fixes) ---
			{
				Question:      "Design a troubleshooting procedure for when bottles are being incorrectly rejected",
				ExpectedFacts: []string{"parámetros|parameter", "calibración|calibration|calibrate|aprendizaje|learning|learn", "tracking|seguimiento|track|monitor"},
				Category:      "synthesis",
				Explanation:   "[p94-96] Tracker params control rejection: 'Distancia del Trigger al Rechazador' sets distance in encoder pulses. 'Ancho de Pulso de Rechazo' range 1-999ms. [p87] Global Learning recalibrates píxel/mm. [p92-93] Tracking screen shows bottle positions. Troubleshooting: verify tracker distances, recalibrate via Global Learning, check tracking sync.",
			},
			{ // Q2 FIXED: removed PET-specific facts, now tests SKU type awareness
				Question:      "What would need to change in the system configuration when switching between different SKU container types?",
				ExpectedFacts: []string{"SKU", "tipo|type|Vidrio|Glass", "contenedor|container"},
				Category:      "synthesis",
				Explanation:   "[p124] SKU types include 'Vidrio o PET' as container material. [p126] Container params (height, diameter, cap) must be reconfigured. [p87] Global Learning may need to be re-run for new container geometry.",
			},
			{
				Question:      "Explain how all five inspection tools work together to ensure complete quality control",
				ExpectedFacts: []string{"seguimiento|tracking|cap tracking", "nivel|level|fill", "objeto flotante|floating object"},
				Category:      "synthesis",
				Explanation:   "[p131-132] Cap tracking positions tapa. [p166] Fill level checks height. [p196-198] Foam/opacity detects discoloration. [p200-202] Floating object finds particles. [p190-192] Code inspection verifies labels. Together they provide multi-dimensional quality control.",
			},
			{
				Question:      "What is the complete set of infrastructure requirements to install this system?",
				ExpectedFacts: []string{"VCA|VAC|volt", "tierra|ground", "aire|air|compressed air"},
				Category:      "synthesis",
				Explanation:   "[p28] Electrical: 120/240 VCA. [p29] Grounding: <1VCA, 14AWG. [p31] Air: 75-85 PSIG, 11 SCFM. [p28] Environment: 5-40°C, <95% humidity. Complete infrastructure includes power, ground, compressed air, and suitable environment.",
			},
			{
				Question:      "How would you configure the system to detect a partially filled beer bottle with excessive foam?",
				ExpectedFacts: []string{"espuma|foam", "opacidad|opacity", "nivel de llenado|fill level"},
				Category:      "synthesis",
				Explanation:   "[p170-171] Foam compensation (C6) for beer: mm of foam per 1mm fill, typical 9-10. [p169] Edge size ~10 normal, ~5 low contrast. [p196-198] Foam/Opacity thresholds use grayscale. [p166] Fill level tool registers liquid height.",
			},
			{
				Question:      "Analyze the safety architecture of the system including electrical, optical, and mechanical protections",
				ExpectedFacts: []string{"tierra|ground", "fusibles|fuse", "UPS", "gafas|glasses|safety glasses"},
				Category:      "synthesis",
				Explanation:   "[p29] Electrical: ground connection, [p47] fuses. [p42] Power: UPS for safe shutdown. [p24] Optical: IEC 62471/IEC 60825 compliance, safety glasses. [p32] Mechanical: tempered glass windows, IP54 enclosure.",
			},
			{
				Question:      "What are all the possible failure modes and their corresponding alarm responses?",
				ExpectedFacts: []string{"alarma|alarm", "fallo|failure|fault"},
				Category:      "synthesis",
				Explanation:   "[p105-108] Alarm list covers: TCP communication failure (Type 3), camera loss (Type 3), encoder failure (Type 2), elevation movement (Type 2), parameter load success (Type 1). Each type has specific response protocol.",
			},
			{
				Question:      "Describe how the system maintains inspection accuracy across a 12-hour production shift",
				ExpectedFacts: []string{"calibración|calibration|compensación|compensation", "aprendizaje|learning|aprend"},
				Category:      "synthesis",
				Explanation:   "[p87] Global Learning calibrates píxel/mm at start. [p170-171] Foam compensation adapts to product variation. [p148-150] RGB learning from 100 bottles sets color baselines. These calibration mechanisms maintain accuracy during extended production.",
			},
			{
				Question:      "Compare the advantages and limitations of the Vortex cooling vs Rittal AC system",
				ExpectedFacts: []string{"vortex|Vortex", "Rittal"},
				Category:      "synthesis",
				Explanation:   "[p55] Vortex: uses compressed air, no moving parts, simple, but requires continuous air supply and limited cooling capacity. [p57] Rittal AC: electrical, better cooling capacity, but more complex, requires maintenance.",
			},
			{
				Question:      "What sequence of configuration steps would optimize the system for inspecting small medicine bottles?",
				ExpectedFacts: []string{"SKU", "contenedor|container", "aprendizaje|learning"},
				Category:      "synthesis",
				Explanation:   "[p100] Create SKU for medicine bottle. [p126] Configure container with small dimensions. [p87] Run Global Learning with new cap diameter. [p131-132] Adjust tracking window for small profile. [p166] Set tight fill level limits for pharmaceutical precision.",
			},
			{
				Question:      "How does the MDIR 2006/42/EC directive influence the design of the safety systems?",
				ExpectedFacts: []string{"2006/42/EC", "seguridad|safety"},
				Category:      "synthesis",
				Explanation:   "[p22] Directive 2006/42/EC (Machinery Directive) requires conformity assessment, safety analysis. [p22-24] Influences: emergency stops, protective enclosures, electrical safety design, risk assessment per EN ISO 12100.",
			},
			{
				Question:      "Explain the complete data architecture of the system including image processing, tracking, and communication",
				ExpectedFacts: []string{"IPC", "Ethernet", "tracker", "cámara|camera"},
				Category:      "synthesis",
				Explanation:   "[p49] IPC processes images. [p51] Ethernet connects IPC to Tracker. [p45] Tracker manages timing and I/O. [p53] Cameras send images to IPC via GigE. Architecture: cameras→IPC (processing), Tracker↔IPC (synchronization via Ethernet).",
			},
			{
				Question:      "What risks and mitigations exist for each type of maintenance activity?",
				ExpectedFacts: []string{"eléctrico|electrical", "EPP|PPE", "gafas|glasses|safety glasses"},
				Category:      "synthesis",
				Explanation:   "[p25] All maintenance requires EPP: safety glasses, cut-resistant gloves. [p29] Electrical: risk of shock, mitigated by grounding and lockout. [p24] Optical: laser risk, mitigated by safety glasses per IEC 60825.",
			},
			{
				Question:      "How would you validate that the system is correctly calibrated after installation?",
				ExpectedFacts: []string{"Aprendizaje Global|Global Learning", "trigger", "encoder"},
				Category:      "synthesis",
				Explanation:   "[p87] Run Global Learning to calibrate píxel/mm. [p64] Verify trigger detects every bottle. [p61] Confirm encoder pulses are consistent. [p92-93] Check tracking screen for correct bottle positioning.",
			},
			{
				Question:      "Synthesize all references to normative standards in the document",
				ExpectedFacts: []string{"EN 60204", "2006/42/EC", "EN ISO 12100"},
				Category:      "synthesis",
				Explanation:   "[p22] 2006/42/EC: Machinery Directive. [p22] EN ISO 12100: safety of machinery general principles. [p22] EN 60204: electrical safety of machinery. [p24] IEC 62471: photobiological safety. [p24] IEC 60825: laser safety.",
			},
			{
				Question:      "What is the relationship between image resolution, lens choice, and inspection accuracy?",
				ExpectedFacts: []string{"lente|lens", "montura C|C-mount|C mount", "píxeles|pixel"},
				Category:      "synthesis",
				Explanation:   "[p53] Cameras use montura C (C-mount) lenses. Resolution in pixels determines detail level. [p87] Píxel/mm ratio (from Global Learning) links physical dimensions to pixel counts. Higher resolution + correct lens = better accuracy.",
			},
			{
				Question:      "How does the system handle concurrent inspections from multiple cameras?",
				ExpectedFacts: []string{"Multicam", "cámaras|camera", "Ethernet"},
				Category:      "synthesis",
				Explanation:   "[p141] Multicam mode combines results from multiple cameras. [p53] Cameras connected via GigE Ethernet. [p45] Tracker synchronizes capture timing across cameras. Results consolidated by Sum or Maximum methods.",
			},
			{
				Question:      "Describe the complete user management lifecycle from creation to deletion",
				ExpectedFacts: []string{"Registrar|Register|create|creation", "rol|role", "Eliminar|Delete|remove"},
				Category:      "synthesis",
				Explanation:   "[p113-115] Lifecycle: Registrar nuevo usuario con rol asignado. Usuario opera según permisos del rol. Administrador puede modificar rol. Eliminar usuario cuando ya no necesario.",
			},
			{ // Q19 FIXED: replaced meta-question with document-grounded synthesis
				Question:      "How do the different container parameters interact with the inspection tools during a product changeover?",
				ExpectedFacts: []string{"contenedor|container", "Aprendizaje Global|Global Learning|learning", "ventana|window"},
				Category:      "synthesis",
				Explanation:   "[p126] Container stores height, diameter, cap size. [p87] Global Learning recalibrates píxel/mm using cap diameter. [p131-132] Tracking window position depends on container height. [p166] Fill level tool window adjusts to new container geometry. All inspection windows must be re-validated after changeover.",
			},
			{
				Question:      "Explain how the rejection timing precision is maintained at different line speeds",
				ExpectedFacts: []string{"encoder", "pulsos|pulse", "velocidad|speed"},
				Category:      "synthesis",
				Explanation:   "[p61] Encoder generates pulses proportional to line movement. [p94-96] Rejection distance in encoder pulses (not time), so it's speed-independent for the base distance. [p95-96] Compensador 1 (multiplication) and Compensador 2 (sum) fine-tune at low speeds where timing is more critical.",
			},
			{
				Question:      "What would be the impact of a 10% increase in humidity on system operation?",
				ExpectedFacts: []string{"humedad|humidity", "95%", "condensación|condensation"},
				Category:      "synthesis",
				Explanation:   "[p28] Max humidity: 95% sin condensación. If humidity rises 10% from near-limit, risk of condensation which can damage electronics. System spec explicitly requires non-condensing conditions.",
			},
			{
				Question:      "Design an optimal SKU configuration strategy for a plant with 50 different bottle sizes",
				ExpectedFacts: []string{"SKU", "contenedor|container", "copiar|copy"},
				Category:      "synthesis",
				Explanation:   "[p100] Each product needs SKU. [p126] Group bottles by container dimensions to share configurations. [p102] Use Copy function to duplicate similar SKUs and adjust. Strategy: create base containers by size, copy SKUs within size groups, customize inspection params.",
			},
			{
				Question:      "How do all the electrical protections work together as a defense-in-depth system?",
				ExpectedFacts: []string{"interruptor|switch|breaker", "fusibles|fuse", "UPS", "filtro de ruido|noise filter", "tierra|ground"},
				Category:      "synthesis",
				Explanation:   "[p40] Main switch: first line of defense. [p43] Noise filter: removes electrical interference. [p47] Fuses: protect individual circuits. [p42] UPS: power continuity. [p29] Ground: fault current path. Layers: switch→filter→fuses→UPS→ground.",
			},
			{
				Question:      "What information from the glossary helps understand the inspection parameters?",
				ExpectedFacts: []string{"threshold|umbral", "Teach-In|teach-in"},
				Category:      "synthesis",
				Explanation:   "[p209] Glossary defines key terms: 'Threshold/Umbral' - decision boundary for pass/fail. 'Teach-In' - automated learning process. 'Global Learning' - píxel/mm calibration. These definitions clarify parameter meanings throughout the manual.",
			},
			{ // Q25 FIXED: replaced "opacidad|opacity" with "presencia|presence"
				Question:      "How does the system handle edge cases like bottles without caps?",
				ExpectedFacts: []string{"tapa|cap", "defecto|defect|non-conform", "presencia|presence"},
				Category:      "synthesis",
				Explanation:   "[p131-132] Cap tracking tool detects cap position. [p135] Section 7.3.2 'Presencia/Defecto de Tapa' evaluates cap presence and defects. A bottle without cap triggers the presence check, resulting in rejection.",
			},
			{
				Question:      "What training would an operator need to safely operate and maintain this system?",
				ExpectedFacts: []string{"capacitación|training", "seguridad|safety", "EPP|PPE"},
				Category:      "synthesis",
				Explanation:   "[p25] Safety training: EPP usage, electrical hazards. [p80-81] Operation training: role-specific system usage. [p70-73] Inspection process understanding. [p87] Calibration procedures. Full training covers safety, operations, calibration, and troubleshooting.",
			},
			{
				Question:      "Explain the complete image acquisition pipeline from trigger to analysis",
				ExpectedFacts: []string{"trigger", "tracker", "encoder", "cámara|camera", "estrobo|strobe", "IPC"},
				Category:      "synthesis",
				Explanation:   "[p64] Trigger detects bottle. [p45] Tracker records event. [p61] Encoder counts pulses. [p53] At calculated distance, Tracker fires cámara + estrobo LED simultaneously. [p49] IPC receives and processes image. Complete pipeline: trigger→tracker→encoder count→camera+strobe→IPC analysis.",
			},
			{
				Question:      "How does the system architecture support future upgrades and expansions?",
				ExpectedFacts: []string{"software", "tracker", "micro SD"},
				Category:      "synthesis",
				Explanation:   "[p45] Tracker firmware on micro SD card allows field updates. [p49] Software-based inspection algorithms can be updated on IPC. Ethernet architecture supports adding cameras. Modular design enables component upgrades.",
			},
			{
				Question:      "What are all the parameters that affect the precision of fill level measurement?",
				ExpectedFacts: []string{"píxeles|pixel", "aprendizaje|learning|referencia|reference", "límites|limit|threshold"},
				Category:      "synthesis",
				Explanation:   "[p87] Global Learning establishes píxel/mm reference. [p166-168] Fill level tool: window position, edge size, detection method. [p168] Limits define pass/fail thresholds. [p170-171] Foam compensation adjusts for beer. Precision depends on calibration quality, edge detection params, and threshold settings.",
			},
			{ // Q30 FIXED: rephrase to be document-grounded, no external knowledge
				Question:      "What specific normative standards does the document reference and what aspects of safety do they each govern?",
				ExpectedFacts: []string{"ISO 12100", "IP54", "grupo de riesgo|risk group"},
				Category:      "synthesis",
				Explanation:   "[p22] EN ISO 12100: safety of machinery, risk assessment methodology. [p22] 2006/42/EC: machinery directive, CE conformity. [p22] EN 60204: electrical safety of machinery. [p24] IEC 62471: photobiological safety, risk groups. [p28] IP54: enclosure protection against dust and water. Each standard governs a specific safety domain.",
			},

			// --- NEW: Category A - Graph Multi-Hop Reasoning (5 tests) ---
			{
				Question:      "How does the foam compensation algorithm use the fill level window parameters to adjust measurements?",
				ExpectedFacts: []string{"compensación|compensation|compensar", "ventana|window", "espuma|foam", "milímetro|millimeter|mm"},
				Category:      "graph-multi-hop",
				Explanation:   "[p170-171] Foam compensation: mm of foam per 1mm fill adjustment, typical value 9-10 for beer. [p167] Window height defines detection area in which foam is measured. [p169] Edge size affects boundary detection. The algorithm measures foam height in the window (in mm after calibration) and divides by compensation factor to adjust fill level reading.",
			},
			{
				Question:      "Trace the complete signal path from when a bottle triggers the sensor to when the rejection pulse fires, including all timing parameters",
				ExpectedFacts: []string{"trigger", "encoder", "pulsos|pulse", "rechazador|rejector|reject", "milisegundos|millisecond|ms"},
				Category:      "graph-multi-hop",
				Explanation:   "[p64] Trigger sensor detects bottle. [p45] Tracker registers event. [p61] Encoder counts pulses for distance measurement. [p94-96] Signal path: trigger→tracker counts encoder pulses→at 'Distancia del Trigger al Rechazador' (in pulses) fires rejection→'Ancho de Pulso de Rechazo' 1-999ms duration. Compensador 1 and 2 fine-tune at low speeds.",
			},
			{
				Question:      "How do the three alarm severity levels relate to the different types of system components (tracker, cameras, elevation)?",
				ExpectedFacts: []string{"Tipo 1|Type 1|informativa|informative|verde|green", "Tipo 3|Type 3|crítica|critical|rojo|red", "comunicación|communication|TCP", "elevación|elevation"},
				Category:      "graph-multi-hop",
				Explanation:   "[p105] Three alarm types: Tipo 1 Informativas (green), Tipo 2 Advertencias (yellow), Tipo 3 Críticas (red). [p105-108] Alarm list: TCP communication failure with Tracker = Tipo 3 critical. Elevation movement issues = Tipo 2 warning. Camera loss = Tipo 3 critical. Parameter load success = Tipo 1 informative.",
			},
			{
				Question:      "What is the relationship between the Global Learning process, pixel-to-mm calibration, and the container diameter stored in the SKU?",
				ExpectedFacts: []string{"Aprendizaje Global|Global Learning", "píxel|pixel", "milímetro|millimeter|mm", "diámetro|diameter", "28"},
				Category:      "graph-multi-hop",
				Explanation:   "[p87] Global Learning compares known cap diameter (28mm standard) with pixel count in image to establish píxel/mm ratio. [p209] Glossary: 'Aprendizaje Global' = pixel/mm calibration. [p126] Container stores diameter in physical dimensions. This calibration chains: container diameter→Global Learning→píxel/mm ratio→all measurement tools.",
			},
			{
				Question:      "How do the Compensador 1 (multiplication) and Compensador 2 (sum) parameters interact with encoder pulses to fine-tune rejection at low speeds?",
				ExpectedFacts: []string{"Compensador|compensator", "multiplicación|multiplication", "suma|sum", "velocidad|speed|low speed", "encoder"},
				Category:      "graph-multi-hop",
				Explanation:   "[p95-96] Compensador 1 (Multiplicación): multiplier for fine-tuning rejection timing at low line speeds. Compensador 2 (Suma): additive value used with Compensador 1 for additional precision. [p95] Initial setup: set both to 0, calibrate at high speed first. Then enable compensators for low-speed fine tuning. Both work with encoder pulse counts to adjust the effective rejection distance.",
			},

			// --- NEW: Category B - Anti-Hallucination / Grounding (5 tests) ---
			{
				Question:      "What is the maximum production speed in bottles per minute that the system can handle?",
				ExpectedFacts: []string{"BPM|bottles per minute|botellas por minuto|velocidad|speed", "not|no|cannot|sin|nunca"},
				Category:      "anti-hallucination",
				Explanation:   "The document does NOT specify a maximum bottles/minute throughput anywhere. System speed depends on encoder pulses and line speed, but no BPM number is given. A correct answer must acknowledge this absence rather than fabricate a number.",
			},
			{
				Question:      "What specific camera resolution in megapixels do the inspection cameras use?",
				ExpectedFacts: []string{"camera|cámara|GigE|pixel|resolution|resolución", "not|no|cannot|sin|nunca"},
				Category:      "anti-hallucination",
				Explanation:   "[p53] Document mentions cameras and GigE interface but gives NO megapixel specification. Camera details include mounting and connection but resolution is not specified in the manual.",
			},
			{
				Question:      "Does the system support WiFi or wireless communication between components?",
				ExpectedFacts: []string{"Ethernet", "RS232|RS-232", "no|not|sin"},
				Category:      "anti-hallucination",
				Explanation:   "[p51] Communication protocols: Ethernet and RS232 only. Document makes NO mention of WiFi, wireless, or any non-wired communication. A correct answer states the available protocols (Ethernet, RS232) and explicitly notes absence of wireless.",
			},
			{
				Question:      "What machine learning or artificial intelligence algorithms does the system use for defect detection?",
				ExpectedFacts: []string{"aprendizaje|learning|calibra", "no|not|sin|does not"},
				Category:      "anti-hallucination",
				Explanation:   "[p87] 'Aprendizaje Global' = pixel/mm calibration, NOT machine learning. [p148-150] 'Aprendizaje RGB' = statistical color baseline, NOT AI. Document uses 'aprendizaje' (learning) only for calibration processes. No ML/AI algorithms are described anywhere in the manual.",
			},
			{
				Question:      "What cloud connectivity or remote monitoring capabilities does the system provide?",
				ExpectedFacts: []string{"Ethernet|TCP|network|red", "not|no|cannot|sin|nunca"},
				Category:      "anti-hallucination",
				Explanation:   "Document has ZERO references to cloud, remote monitoring, internet connectivity, or any external network communication. System uses only internal Ethernet (IPC↔Tracker) and RS232. A correct answer must state this absence and describe what IS available.",
			},

			// --- NEW: Category C - Numerical Precision (5 tests) ---
			{
				Question:      "List all the exact current ratings at both 120V and 240V, with and without air conditioning",
				ExpectedFacts: []string{"10", "14", "120", "240"},
				Category:      "numerical",
				Explanation:   "[p28] Electrical specifications: 10A at 120V without AC, 14A at 120V with AC, 5A at 240V without AC, 7A at 240V with AC. All four combinations are specified in the electrical specs table.",
			},
			{
				Question:      "What are the exact weights in kilograms for Model A Standard, Model B, and Model C Standard XL?",
				ExpectedFacts: []string{"153", "80", "300"},
				Category:      "numerical",
				Explanation:   "[p32] Model A Standard: 153 kg. [p34] Model B: 80 kg. [p36] Model C Standard XL: 300 kg. These are the exact weights from the specifications tables of each model.",
			},
			{
				Question:      "What are the exact air pressure range, flow rate, and the recommended edge size values for fill level detection?",
				ExpectedFacts: []string{"75", "85", "11 SCFM|11SCFM", "10", "5"},
				Category:      "numerical",
				Explanation:   "[p31] Air pressure: 75-85 PSIG, flow rate: 11 SCFM. [p169] Edge size (Tamaño de Borde): ~10 for normal contrast, ~5 for low contrast containers. These are the recommended starting values.",
			},
			{
				Question:      "What is the ground connection voltage specification, minimum wire gauge, and the standard cap diameter in mm?",
				ExpectedFacts: []string{"1 VCA|1VCA|1 VAC|1VAC", "14 AWG|AWG 14|14AWG", "28 mm|28mm"},
				Category:      "numerical",
				Explanation:   "[p29] Ground: voltage between ground and neutral must be less than 1 VCA. Minimum wire gauge: 14 AWG. [p126] Standard cap diameter: 28 mm. Three precise specifications from different sections.",
			},
			{
				Question:      "What is the inactivity timeout duration, the rejection pulse width range, and the standard camera debounce value?",
				ExpectedFacts: []string{"20 minuto|20 minute|20 min", "999 ms|999ms|1-999|1–999", "50"},
				Category:      "numerical",
				Explanation:   "[p117] Inactivity timeout: 20 minutes. [p95] Rejection pulse width: 1-999 ms range. [p99] Camera debounce (Inhibición de Activación): 50 μs. Three timing specifications from different system areas.",
			},

			// --- NEW: Category D - Reasoning Chains (5 tests) ---
			{
				Question:      "If the encoder provides 2000 pulses and the pulse divider is set to 2, and the camera is 234mm from the trigger with 1:1 ratio, how many divided pulses correspond to the camera trigger distance?",
				ExpectedFacts: []string{"234", "divisor|divider", "1000", "pulsos|pulse"},
				Category:      "reasoning",
				Explanation:   "[p95-96] Pulse divider halves the count: 2000/2 = 1000 effective pulses. Camera distance is configured as 234 pulses (1:1 means 1 pulse = 1mm, 234mm = 234 pulses). The distance parameter of 234 stays the same because distances are configured based on the example on p95. The divider affects total available resolution (1000 instead of 2000 pulses per revolution).",
			},
			{
				Question:      "If a beer bottle shows 70mm of foam in the inspection window and the compensation parameter is set to 7, what correction is applied to the fill level height?",
				ExpectedFacts: []string{"10 mm|10mm|10 milímetro", "70", "7", "compensación|compensation"},
				Category:      "reasoning",
				Explanation:   "[p170-171] Foam compensation formula: foam_height / compensation_factor = correction. 70mm / 7 = 10mm added to fill level. This example is explicitly described in the document: the compensation value represents mm of foam per 1mm of fill correction.",
			},
			{
				Question:      "Why can't an operator copy an SKU that is in 'En Progreso' status, and what must happen first?",
				ExpectedFacts: []string{"Progreso|Progress", "no puede|cannot|not allowed", "Estático|Static|Dinámico|Dynamic"},
				Category:      "reasoning",
				Explanation:   "[p102] 'SKUs con estado En Progreso no pueden ser copiados' — they are still being configured and may be incomplete. [p124] SKU must first advance to 'Ajuste Estático' or 'Ajuste Dinámico' status before it can be copied, ensuring only validated configurations are duplicated.",
			},
			{
				Question:      "Given the document states noise is <70 dB(A) at 1m and the power measurement is omitted because it's below 80 dB(A), what range does the actual noise fall within?",
				ExpectedFacts: []string{"70", "80", "dB"},
				Category:      "reasoning",
				Explanation:   "[p30] Noise emission: less than 70 dB(A) measured at 1 meter. Sound power measurement omitted because it's below 80 dB(A) per applicable standards. Therefore actual noise is below 70 dB(A) at measurement distance, and sound power is below 80 dB(A).",
			},
			{
				Question:      "If the system is configured with foam detection and the fill level detection is set to 'Borde Inferior', explain why this combination is recommended for beer inspection",
				ExpectedFacts: []string{"inferior|bottom", "espuma|foam", "arriba|up|above", "contraste|contrast"},
				Category:      "reasoning",
				Explanation:   "[p172-173] 'Borde Inferior' (bottom edge) traces from bottom upward. Recommended for beer/foam products because it detects the liquid-air boundary from below, avoiding confusion with foam layer above. Foam creates a low-contrast zone from the top, so searching from bottom provides cleaner edge detection at the actual liquid surface.",
			},
		},
	}
}

// ALTAVisionGraphTestDataset returns 7 targeted test cases for evaluating
// graph-mode retrieval on the ALTAVision manual. These cover electrical specs,
// grounding, environment, anchoring, tracker card, beacon lights, and vortex cooling.
func ALTAVisionGraphTestDataset() Dataset {
	return Dataset{
		Name:       "ALTAVision Graph Test - Targeted Retrieval",
		Difficulty: DifficultyGraphTest,
		Tests: []TestCase{
			{
				Question:      "What are the electrical power requirements?",
				ExpectedFacts: []string{"120|240", "VAC", "THD"},
				Category:      "single-fact",
				Explanation:   "Specified voltage is 120/240 VAC, current 5-14 A depending on A/C, and power quality THD < 5%.",
			},
			{
				Question:      "How should the machine be grounded?",
				ExpectedFacts: []string{"1 VAC|Neutral|neutral", "14 AWG|14AWG"},
				Category:      "single-fact",
				Explanation:   "Proper grounding: < 1 VAC Neutral-Ground, wire gauge 14 AWG.",
			},
			{
				Question:      "What are the environmental operating conditions?",
				ExpectedFacts: []string{"5°|5º|5 °|5 º", "10°|10º|10 °|10 º|40°|40º", "30%|95%"},
				Category:      "single-fact",
				Explanation:   "Temperature 5ºC–10ºC (or up to 40ºC) and humidity limits 30%-95%.",
			},
			{
				Question:      "How do I anchor the machine to the floor?",
				ExpectedFacts: []string{"anchor|bolt|ancla|perno", "4 leg|four leg|4 pata|cuatro pata", "10mm|10 mm"},
				Category:      "single-fact",
				Explanation:   "Altabev provides anchor bolts for all 4 legs and conveyors, with a 10mm drill bit for installation.",
			},
			{
				Question:      "What is the function of the Tracker Card (E1375.1)?",
				ExpectedFacts: []string{"E1375", "camera|cámara|strobe", "reject|rechazo"},
				Category:      "multi-fact",
				Explanation:   "Tracks container position, triggers cameras/strobes, and manages rejection signals.",
			},
			{
				Question:      "What do the beacon light colors mean?",
				ExpectedFacts: []string{"Green|green|Verde|verde", "Blue|blue|Azul|azul", "Red|red|Rojo|rojo"},
				Category:      "multi-fact",
				Explanation:   "Green: Standard Operation, Blue: Reject Detection, Red: Fault or Disabled Rejector.",
			},
			{
				Question:      "How does the Vortex cooling system work?",
				ExpectedFacts: []string{"compressed air|aire comprimido", "vortex|Vortex", "positive pressure|presión positiva|hot|cold"},
				Category:      "multi-hop",
				Explanation:   "Compressed air enters vortex tube, splits into hot/cold streams. Cold air cools cabinet, hot air exhausts. Cabinet maintained at slight positive pressure to prevent outside air entry.",
			},
			{
				Question:      "How do I interpret the main screen counters?",
				ExpectedFacts: []string{"Global|global|Globales|globales", "Specific|specific|Específicos|específicos", "multiple criteria|múltiples criterios|counted once|una sola vez|una vez"},
				Category:      "multi-fact",
				Explanation:   "Two types: Global Counters (total inspected, total rejected, overall rejection rate) and Specific Counters (per-criterion breakdown). When a unit fails multiple criteria, each specific counter increments but the global rejected count only counts the unit once.",
			},
		},
	}
}

// ALTAVisionAllDatasets returns all ALTAVision datasets keyed by difficulty.
func ALTAVisionAllDatasets() map[string]Dataset {
	return map[string]Dataset{
		DifficultyEasy:      ALTAVisionEasyDataset(),
		DifficultyMedium:    ALTAVisionMediumDataset(),
		DifficultyHard:      ALTAVisionHardDataset(),
		DifficultySuperHard: ALTAVisionSuperHardDataset(),
		DifficultyGraphTest: ALTAVisionGraphTestDataset(),
	}
}
