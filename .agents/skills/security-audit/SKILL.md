---
name: security-audit
description: Auditar permisos, datos, secretos y soporte.
---
# security-audit
Leer AGENTS.md, docs/security/security.md, ADR correspondientes y docs/testing/STRATEGY.md antes de actuar.
## Procedimiento
Threat model, IDOR, replay, autorización supervisor, logs y acceso remoto; reportar severidad y evidencia.
Trabajar con alcance autorizado, datos sintéticos y rama aislada. Reportar archivos, comandos realmente ejecutados, resultados, riesgos, alternativas y pendientes. No desplegar ni modificar producción. Si existe contradicción, proponer ADR y detener cambio afectado.


## CI y secretos
Verificar acciones fijadas por SHA, permisos mínimos, contexto de evento y
procedencia del validador. No ejecutar código de PR en pull_request_target con
credenciales. No publicar hallazgos con secretos ni subir informes crudos.
Comprobar Gitleaks y push protection, excepciones específicas, prueba negativa
sintética y revisión P0–P3 registrada en PR. Un scan limpio no certifica seguridad.
