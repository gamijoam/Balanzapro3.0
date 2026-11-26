# Entregas Alpha por hitos

Decisión del propietario: agrupar cambios importantes de frontend para
evitar descargar/reinstalar después de cada ajuste. No cambia la unidad
de PR ni los controles de seguridad. Un candidato puede generarse en cada
CI; se recomienda probar manualmente solo un hito listo.

## Hito visual A — espacio de trabajo de inventario

- [x] Instalación física previa confirmada por el propietario (APK de prueba).
- [x] Cascarón, tema y tokens, tablas, formularios, diálogos, estados, badges.
- [x] Flujo local crear/editar productos de ejemplo (#187).
- [x] Refinamiento inicial de navegación y densidad (#189).
- [ ] Consulta, filtros combinados, indicadores navegables, edición directa (#191).
- [ ] Revisar consistencia visual del dashboard, catálogo, lista y formulario;
      capturas reales y teclado/texto ampliado, móvil y escritorio.
- [ ] Preparar recorrido de consulta de movimientos de ejemplo, sin confundirlo
      con operaciones reales. No implementar reglas comerciales en el cliente.
- [ ] Artefactos Android y web del mismo commit integrado con CI verde.
- [ ] Prueba física Android y prueba en navegador, registrar versión y feedback.

El hito visual no requiere fingir autenticación, persistencia o acceso fiscal.
Para MVP funcional, completar primero ADR-169 (identidad, autorización,
sesiones, dispositivo y auditoría según decisiones vigentes), API protegida y
operaciones de inventario con persistencia, idempotencia y pruebas de integración.

## Checklist para cada candidato manual

1. Confirmar PRs del hito integrados y aceptación independiente documentada.
2. Registrar commit main y URL de flutter-preview exitosa; comprobar artefactos
   vigentes Android y web del mismo commit, BUILD_COMMIT.txt y SHA256SUMS.
3. APK debug con firma verificada por CI; identificar limitaciones nativas
   y versión. No distribuir como producción ni sugerir datos reales.
4. Probar Inicio, filtros, búsqueda sin resultados, alta/edición, validación,
   cancelación, navegación y cambio de tamaño; comprobar tema oscuro, teclado
   y texto ampliado. Registrar entorno y defectos, no afirmar prueba física
   por widget test.
5. Entregar enlaces y una lista breve de novedades/pendientes. Solo una
   instalación por hito salvo defecto bloqueante que requiera otra prueba.
6. Registrar feedback en issues. Revertir PR si se detecta regresión crítica.

## Web y firma de desarrollo: decisiones pendientes

Web descargable: el ZIP se sirve con HTTP local según frontend/README.md.
Un entorno web alojado debe ser estático, aislado de las instancias actuales,
sin secretos, datos reales ni conexión a endpoints comerciales. Elegir URL,
visibilidad y estrategia de publicación con el propietario; no activar
hosting ni alterar DNS en este PR. El futuro preview puede acelerar revisiones
sin reinstalar Android.

Las llaves debug de CI son efímeras; Android puede rechazar una actualización
con otra firma. No afirmar que una actualización in-place es posible.
Para entregas periódicas evaluar una firma de desarrollo separada, estable
y protegida, con identificación/versionCode coherentes, permisos mínimos,
rotación y revocación. No reutilizar clave de producción ni comprometerla en
Git. Requiere autorización y revisión de seguridad antes de aplicarse.
