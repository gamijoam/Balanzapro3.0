# Roadmap por entregas
| Fase | Entrega | Puerta de salida |
|---|---|---|
| 00 | Blueprint | Registro revisado y decisiones críticas ratificadas |
| 01 | Fundación | Monorepo real, Go/Flutter/PG/SQLite, contratos, lint/CI, entorno DEV aislado |
| 02 | Spikes | UI Shadcn y plataformas, concurrencia, caché, periféricos, exactitud decimal, fiscal/proveedor |
| 03 | Cascarón | Identidad, ámbito, módulos, configuración, auditoría y dispositivos |
| 04 | Núcleo | SKU/unidades/kardex, autoridad y reservas |
| 05 | Monedas y precios | Tasas, valoración ratificada, listas y reprecificación |
| 06 | Compras | Recepción directa, factura, obligación, pago mínimo, devolución |
| 07 | POS | Carrito, checkout, crédito, cobro, devoluciones y comprobante |
| 08 | Tesorería | Libro, caja, bancos, gastos, conciliación |
| 09 | Fiscal | Impuestos/documentos, adaptador venezolano validado y contingencia |
| 10 | Distribución | Instalador, licencia, actualizador, diagnóstico, restauración |
| 11 | Piloto | Comercios voluntarios, seguridad, pruebas y soporte |
| 12 | Estable | Release firmada y aprobación humana |
| 13 | Expansión | Restaurantes/producción, híbrido, fidelización, contabilidad, otros países |
Dependencias transversales: contratos mínimos de tesorería/fiscal desde compras y ventas; respaldos desde primera persistencia; validación legal inicia en spike, no al final; pruebas/seguridad en todas las fases. Vertical inicial recomendada: bodega local con compra → precio → venta → caja → devolución. No confundir fase con fechas comprometidas.

