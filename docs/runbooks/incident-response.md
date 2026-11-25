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

