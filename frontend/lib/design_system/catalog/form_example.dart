import "package:flutter/widgets.dart";

import "../bp_form.dart";
import "../design_system.dart";

/// Demonstrates presentation validation; no API or persistent writes.
class BPFormExample extends StatefulWidget {
  const BPFormExample({super.key});
  @override
  State<BPFormExample> createState() => _BPFormExampleState();
}

class _BPFormExampleState extends State<BPFormExample> {
  final name = TextEditingController();
  final code = TextEditingController();
  int revision = 0;
  @override
  void dispose() {
    name.dispose();
    code.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    return ColoredBox(
      color: theme.background,
      child: SafeArea(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.all(BPTokens.lg),
              child: Text(
                "Catálogo BPForm · Datos ficticios",
                style: theme.title,
              ),
            ),
            const Text("El formulario no guarda productos reales."),
            Expanded(
              child: BPForm(
                key: ValueKey(revision),
                sectionTitle: "Identificación de ejemplo",
                onCancel: () {
                  name.clear();
                  code.clear();
                  setState(() => revision++);
                },
                cancelLabel: "Limpiar ejemplo",
                fields: [
                  BPTextField(
                    id: "name",
                    label: "Nombre del producto",
                    controller: name,
                    placeholder: "Ej. Arroz",
                    requiredField: true,
                    help: "Nombre de ejemplo.",
                    validate: (value) =>
                        value.trim().isEmpty ? "Escribe un nombre" : null,
                  ),
                  BPTextField(
                    id: "code",
                    label: "Código",
                    controller: code,
                    placeholder: "Ej. SKU-001",
                    validate: (value) =>
                        value.trim().isEmpty ? "Escribe un código" : null,
                  ),
                ],
                onSubmit: (_) async => true,
                successMessage: "Ejemplo validado · No se guardaron datos",
              ),
            ),
          ],
        ),
      ),
    );
  }
}
