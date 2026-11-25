# Estrategia de pruebas TDD
Cada caso: precondiciones, comando, resultado esperado, errores y evidencia. RED reproducible, GREEN mínimo, refactor, revisión independiente; pruebas unitarias Go/Dart, integración PG real (Testcontainers), contrato OpenAPI, race/fuzz, mutation (herramienta candidata), Flutter widget/golden/integration, k6/Toxiproxy. No tratar cobertura como garantía; para reglas críticas exigir todos los casos de aceptación. Fixtures sintéticos deterministas; nunca datos de cliente.
## Matriz mínima
| ID | Escenario | Resultado esperado |
|---|---|---|
| INV-01 | Dos cajas última unidad | Una confirmación; otra conflicto; kardex y stock coherentes |
| INV-02 | Combo con componente insuficiente | Ningún descuento parcial |
| INV-03 | Caja24, paquete6, unidad y fracción | Equivalencia exacta; no doble stock |
| INV-04 | Mismo serial dos ventas | Una confirmación |
| INV-05 | Lote vencido/bloqueado | No vendible; FEFO solo elegibles |
| INV-06 | Conteo concurrente | Ajuste basado en corte o bloqueo; no sobrescribir movimientos |
| INV-07 | Transferencia/reempaque | Conservación de unidades y valor |
| FX-01 | Cambio de tasa | No altera histórico; tasa vigente/versionada |
| COST-01 | Harinas 5→7, precio 8→10 | Stock 14; valor 90; reposición 98; venta potencial 140 |
| COST-02 | Corrección costo con 30 vendidas | Ajustes 140 inventario y 60 costo venta (ejemplo simple) |
| PRICE-01 | Descuento tras promoción bajo mínimo | Bloquear o autorización válida; no eludir |
| BUY-01 | Recepción concurrente/doble ID | No exceder ni duplicar stock |
| BUY-02 | Factura antes de recibir | No aumenta stock |
| BUY-03 | Devolución sin crédito confirmado | Salida logística; obligación no reducida aún |
| SALE-01 | Timeout durante checkout | Consultar mismo ID; sin duplicar |
| SALE-02 | Carrito abierto cambia tasa/precio | Política y validación backend; sin cambio silencioso |
| SALE-03 | Dos créditos sobre límite | No exceder exposición sin permiso |
| SALE-04 | Apartado vence | Liberación única; anticipo separado |
| SALE-05 | Devolución parcial promo/serial/lote | Límite acumulado, monto original y trazabilidad |
| SALE-06 | Impresora falla tras confirmación | Reimprimir; no nueva venta |
| FIN-01 | Dos abonos a misma deuda | No sobreaplicar |
| FIN-02 | Transferencia, divisas y comisión | Libro balanceado y fondos no duplicados |
| FIN-03 | Arqueo ciego USD/VES | Efectivo por moneda; diferencias auditadas |
| FIN-04 | Liquidación tarjeta 500→490+10 | Fondos pendientes liquidados y comisión registrada |
| FIN-05 | Importar estado dos veces | No duplica movimiento; ambiguos manual |
| FISC-01 | IVA incluido/excluido y mixto | Totales exactos y snapshot de regla validada |
| FISC-02 | Numeración concurrente | Sin duplicados por autoridad; reglas legales |
| FISC-03 | Emisión respuesta incierta | Reconciliar antes de reintentar |
| FISC-04 | Cambio norma/contingencia | Historial inmutable, regularización autorizada |
| SEC-01 | IDOR entre empresas | Denegación en servidor |
| SEC-02 | Reusar permiso supervisor | Denegación; token vinculado/expira |
| OPS-01 | Licencia offline/expirada | Gracia, exportación y obligaciones conservadas |
| OPS-02 | Paquete actualización manipulado | Rechazo antes de instalar |
| OPS-03 | Restauración aislada | Verificar integridad y reconciliación |
| OPS-04 | Telemetría sin consentimiento | Sin fuga comercial/secreto |
## Rendimiento
Hardware y versiones documentados; 100k SKU, 1M movimientos, 50 terminales concurrentes, mezcla de ventas/recepción/precios. p95 búsqueda local <100 ms, confirmación LAN <300 ms excluyendo externos como objetivos, no resultados. Medir p50/p95/p99, throughput, CPU/RAM, bloqueos, recuperación y exactitud. Probar Windows/Android/iOS/web y equipo real periférico. CI con lint, tests, análisis secretos, vulnerabilidades y SBOM; puertas de release con QA, respaldo/restauración y fiscal validado. No afirmar pruebas ejecutadas hasta que existan.

