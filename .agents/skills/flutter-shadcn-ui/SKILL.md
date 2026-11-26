---
name: flutter-shadcn-ui
description: Construir y auditar UX multiplataforma con design system BP y Shadcn encapsulado.
---
# flutter-shadcn-ui
Leer AGENTS.md, docs/design-system/UX.md, ADR-011, ADR-169, docs/testing/STRATEGY.md
y los componentes existentes antes de actuar.
## Procedimiento
1. Identificar tarea del usuario, flujo principal, ancho, teclado/ratón/táctil
   y estados (carga, vacío, error, desconexión). Proponer cambios antes de
   rediseños masivos y justificar decisiones.
2. Reutilizar tokens semánticos y componentes BP; no estilos arbitrarios ni
   importaciones Shadcn en features. Extender biblioteca si falta componente.
3. Crear pruebas RED de comportamiento; validar foco, etiquetas, contraste,
   texto ampliado, temas claro/oscuro y 360/800/1440 px. Acciones táctiles >=48.
4. Implementar GREEN, refactorizar y actualizar catálogo. Usar skeletons
   geométricamente estables, movimiento reducido y mensajes de error accionables.
5. Capturar evidencia real cuando sea posible; golden/widget tests, builds y
   pruebas en dispositivos distinguidos. Revisar regresiones de cambios globales.
6. Reportar comandos, resultados, riesgos, pendientes y revisión independiente.
No duplicar reglas de negocio ni tocar producción, datos reales o secretos.
Trabajar en rama aislada. No instalar dependencias sin revisión; Widgetbook
es candidato y no se adopta automáticamente. Si existe contradicción,
proponer ADR y detener el cambio afectado.
