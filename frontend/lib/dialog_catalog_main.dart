import "package:flutter/widgets.dart";

import "design_system/design_system.dart";
import "design_system/bp_dialog.dart";

void main() => runApp(const BPApp(home: BPDialogExample()));

class BPDialogExample extends StatefulWidget {
  const BPDialogExample({super.key});

  @override
  State<BPDialogExample> createState() => _BPDialogExampleState();
}

class _BPDialogExampleState extends State<BPDialogExample> {
  String result = "Sin decisión";
  bool open = false;

  Future<void> _show() async {
    if (open) return;
    setState(() => open = true);
    final confirmed = await BPDialog.confirm(
      context,
      title: "Confirmar ajuste de ejemplo",
      description: "Esta demostración no cambia existencias ni datos reales.",
      confirmLabel: "Confirmar ejemplo",
    );
    if (!mounted) return;
    setState(() {
      open = false;
      result = confirmed ? "Confirmado · Sin cambios reales" : "Cancelado";
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    return ColoredBox(
      color: theme.background,
      child: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(BPTokens.lg),
          children: [
            Text("Catálogo BPDialog · Datos ficticios", style: theme.heading),
            const SizedBox(height: BPTokens.lg),
            BPCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  BPButton(
                    label: "Abrir confirmación",
                    onPressed: open ? null : _show,
                  ),
                  const SizedBox(height: BPTokens.md),
                  Semantics(liveRegion: true, child: Text(result)),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
