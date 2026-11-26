# UX y design system
Shadcn UI Flutter es dirección aprobada; prueba de concepto obligatoria en Windows, Android, iOS, web. Encapsular tokens/componentes; contraste, escala tipográfica, teclado, lector de pantalla, foco y estados de carga/error/vacío; light/dark. Modo básico por defecto, opciones avanzadas activables por perfil; no esconder obligaciones legales ni permisos. POS con búsqueda/escaneo permanente, productos y carrito, total USD/VES según configuración, botones grandes, teclado y táctil, F2 búsqueda/F4 cliente/F6 descuento/F12 cobro como propuestas ajustables. Variantes, presentaciones, unidades pesables y listas por línea sin salir del POS; mostrar stock y precio autorizado. En móvil reordenar vistas; no encoger escritorio. Errores accionables: sin servidor conservar carrito, timeout consultar operación original, impresión fallida reimprimir si venta confirmada; nunca mostrar SQL a cajero. Tablas virtualizadas/paginadas, edición masiva con preview y confirmación. Widgetbook candidato; pruebas golden e integración. No usar imágenes como dependencia del escaneo. Adaptadores para balanza, lector, impresora, gaveta, fiscal. Plantillas comerciales personalizadas, fiscales protegidas.


## Estándar ratificado — ADR-011
Fuente de verdad: design_system/tokens, themes, components, layouts, skeletons
y responsive. Usar nombres semánticos (primary, surface, foreground, danger,
warning, success), no colores por módulo. Densidad compacta de escritorio
no debe disminuir los objetivos táctiles. Layouts por restricciones de ancho,
con navegación equivalente y sin funciones inaccesibles en móvil.

Cada componente debe tener contrato, variantes, estados disabled/loading/error
cuando apliquen, foco y etiquetas, ejemplos en catálogo y pruebas. Para skeletons
usar bloques que conserven la geometría; estado de carga anunciado sin anunciar
cada bloque. Texto ampliado y modo oscuro sin recortes ni pérdida de acciones.
Objetivo AA: 4.5:1 texto normal y 3:1 texto grande/controles relevantes.
Probar también casos de nombres largos, lista vacía, error/reintento y sin red.
No inventar estados de éxito ni habilitar operaciones reales en el prototipo.
La paleta y medidas concretas se podrán ajustar desde los tokens; no son marca
definitiva hasta revisión visual. Widgetbook, Shadcn y sus dependencias requieren
auditoría antes de adoptarse definitivamente.
