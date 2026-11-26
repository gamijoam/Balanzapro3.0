# Runbooks de diseño (no comandos de producción)
## Checkout incierto
Conservar carrito e ID de operación; consultar resultado en autoridad. Si confirmado recuperar comprobante; si pendiente esperar conciliación; si no registrado y backend autoriza, reintentar mismo ID/payload. No cobrar nuevamente si proveedor incierto. Si falla impresión, reimprimir solo comprobante existente.
## Corte de Internet vs servidor
Comprobar conexión LAN; si autoridad accesible continuar operaciones permitidas. Si no, guardar borrador y bloquear confirmación inicial. Fiscal externo caído: activar únicamente contingencia previamente autorizada; si no disponible, bloquear emisión correspondiente.
## Restauración
Aislar instalación, preservar evidencia y respaldo reciente, verificar manifest/clave/versión, restaurar en entorno aislado, comprobar stock/kardex, libro/saldos, obligaciones, secuencias, archivos y operaciones externas pendientes. Autorizar retorno de tráfico después de reconciliar; no duplicar escrituras.
## Actualización fallida
Detener rollout, conservar logs y estado, evaluar compatibilidad de esquema; revertir binarios solo si compatible; restaurar o reparar con plan aprobado y reconciliar operaciones posteriores. Prohibido restaurar a ciegas.
## Incidente de seguridad
Aislar credenciales/dispositivo, revocar tokens al contacto, proteger logs, rotar secretos afectados, revisar ámbito e integridad, notificar según contrato y norma; no extraer datos comerciales indiscriminadamente.

## Migración 00003: restricción de unidades NOT VALID
Si al migrar aparece `WARNING: inventory_skus: hay SKUs con unidad base fuera de su dimensión`, la restricción `inventory_skus_unit_dimension` queda NOT VALID: las filas nuevas sí se validan, las antiguas no.
1. Listar las filas afectadas: `SELECT id, company_id, code, quantity_kind, base_unit FROM inventory_skus WHERE (quantity_kind, base_unit) NOT IN (('unit','und'),('weight','kg'),('weight','g'),('length','m'),('length','cm'),('volume','l'),('volume','ml'));`
2. Corregir cada SKU con el responsable del negocio. Si es un empaque, pasarlo a presentación (ADR-060/171). Hacerlo con un respaldo verificado previo.
3. Validar: `ALTER TABLE inventory_skus VALIDATE CONSTRAINT inventory_skus_unit_dimension;`

