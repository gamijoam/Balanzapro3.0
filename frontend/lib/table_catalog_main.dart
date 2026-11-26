import "package:flutter/widgets.dart";

import "design_system/catalog/table_example.dart";
import "design_system/design_system.dart";

void main() => runApp(const BPApp(home: _TableCatalog()));

class _TableCatalog extends StatelessWidget {
  const _TableCatalog();
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
                "Catálogo BPTable · Datos ficticios",
                style: theme.title,
              ),
            ),
            const Expanded(child: BPTableExample()),
          ],
        ),
      ),
    );
  }
}
