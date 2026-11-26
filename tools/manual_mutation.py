#!/usr/bin/env python3
"""Prueba de mutación manual del backend (ADR-173).

Uso, desde backend/ con el árbol limpio (todo commiteado):
    BP_TEST_DATABASE_URL=postgres://usuario@127.0.0.1:5432/bp_test?sslmode=disable \
        python3 ../tools/manual_mutation.py
Aplica cada mutante, corre go test ./..., lo restaura con git checkout y
escribe ../docs/evidence/manual-mutation.txt.
"""
import subprocess, os, sys
if subprocess.run(["git", "status", "--porcelain", "--", "."], capture_output=True, text=True).stdout.strip():
    sys.exit("commit or stash changes first: mutants are restored with git checkout")
if not os.environ.get("BP_TEST_DATABASE_URL"):
    sys.exit("BP_TEST_DATABASE_URL is required")
M=[
 ("M01 descuento condicional permite negativo","internal/inventory/adapters/pgstore/store.go","AND physical + $4::numeric - (reserved + $6::numeric) - blocked >= 0","AND physical + $4::numeric - (reserved + $6::numeric) - blocked >= -1000000"),
 ("M02 sin orden de locks","internal/inventory/application/ports.go","sort.Slice(keys, func(i, j int) bool { return lessKey(keys[i], keys[j]) })","_ = sort.Slice"),
 ("M03 hash ignora payload","internal/inventory/application/service.go","\th.Write(b)\n","\t_ = b\n"),
 ("M04 replay no se marca","internal/inventory/application/service.go","\t\t\tres.Replayed = true\n","\t\t\tres.Replayed = false\n"),
 ("M05 precisión no se valida","internal/inventory/domain/domain.go","if q.Scale() > s.Precision {","if q.Scale() > s.Precision+6 {"),
 ("M06 unidad sin dimensión","internal/inventory/domain/domain.go","case UnitKind(baseUnit) != kind:","case UnitKind(baseUnit) == \"x\":"),
 ("M07 límite de dígitos enteros","internal/platform/qty/qty.go","if intDigits(s) > MaxIntegerDigits {","if intDigits(s) > MaxIntegerDigits+5 {"),
 ("M08 reserva vencida se consume","internal/inventory/application/service.go","\t\tif expired {\n\t\t\treturn OperationResult{}, fmt.Errorf","\t\tif false && expired {\n\t\t\treturn OperationResult{}, fmt.Errorf"),
 ("M09 evento de reserva omitido en release","internal/inventory/application/service.go","func recordEvent(ctx context.Context, tx Tx, sc Scope, r Reservation, typ string, st BalanceState, opType string, opID uuid.UUID) error {\n","func recordEvent(ctx context.Context, tx Tx, sc Scope, r Reservation, typ string, st BalanceState, opType string, opID uuid.UUID) error {\n\tif typ == ReservationEventReleased {\n\t\treturn nil\n\t}\n"),
 ("M10 transferencia sin estado previo","internal/inventory/adapters/pgstore/store.go","WHERE company_id = $1 AND id = $2 AND status = 'shipped'`, company, id, status)","WHERE company_id = $1 AND id = $2`, company, id, status)"),
 ("M11 aislamiento: SKU sin filtro de empresa","internal/inventory/adapters/pgstore/store.go","FROM inventory_skus WHERE company_id = $1 AND id = $2`, company, id).","FROM inventory_skus WHERE ($1::uuid IS NOT NULL) AND id = $2`, company, id)."),
 ("M12 guarda de BD acepta cualquier nombre","internal/platform/pgtest/pgtest.go","func isTestName(db string) bool { return db == \"test\" || strings.HasSuffix(db, \"_test\") }","func isTestName(db string) bool { return true }"),
 ("M13 barrido se detiene tras el primer intento","internal/inventory/jobs/sweeper.go","\t\ts.sweep(ctx)\n","\t\ts.sweep(ctx)\n\t\treturn\n"),
 ("M14 ajuste sin motivo","internal/inventory/application/service.go","&& reason == \"\" {","&& reason == \"__never__\" {"),
]
out=[]
for name,f,a,b in M:
    src=open(f).read()
    if src.count(a)!=1:
        out.append(f"{name}: NO APLICADO (patrón {src.count(a)} veces)"); continue
    open(f,"w").write(src.replace(a,b))
    r=subprocess.run(["go","test","-count=1","./..."],capture_output=True,text=True,timeout=600)
    subprocess.run(["git","checkout","--",f])
    failed=[l.split()[2] for l in r.stdout.splitlines() if l.startswith("--- FAIL")]
    status="MATADO" if r.returncode!=0 else "SOBREVIVE"
    build = "build failed" in r.stdout+r.stderr
    out.append(f"{name}: {status}{' (no compila)' if build else ''}; pruebas que fallan: {', '.join(failed[:4]) or '-'}")
    print(out[-1], flush=True)
open("../docs/evidence/manual-mutation.txt","w").write("Prueba de mutación manual (2026-11-26, Go 1.26.11, PostgreSQL 16 local).\nCada mutante se aplica, se corre go test ./... y se restaura con git checkout.\n\n"+"\n".join(out)+"\n")
