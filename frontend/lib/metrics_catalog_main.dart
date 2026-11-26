import "package:flutter/widgets.dart";

import "design_system/bp_badge.dart";
import "design_system/bp_metrics.dart";
import "design_system/design_system.dart";

void main() => runApp(const BPApp(home: BPMetricsExample()));

class BPMetricsExample extends StatelessWidget {
  const BPMetricsExample({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    return ColoredBox(
      color: theme.background,
      child: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(BPTokens.lg),
          children: [
            Text(
              "Catálogo de indicadores · Datos ficticios",
              style: theme.heading,
            ),
            const SizedBox(height: BPTokens.lg),
            const BPMetricCard(
              label: "Stock bajo de ejemplo",
              value: "2",
              actionLabel: "Acción no disponible",
              onPressed: null,
              tone: BPTone.warning,
            ),
            const SizedBox(height: BPTokens.lg),
            BPDistribution(
              title: "Distribución de ejemplo",
              parts: const [
                BPDistributionPart("disponibles", 3, BPTone.success),
                BPDistributionPart("con stock bajo", 2, BPTone.warning),
              ],
            ),
            const SizedBox(height: BPTokens.lg),
            BPDistribution(
              title: "Ejemplo vacío",
              parts: const [
                BPDistributionPart("disponibles", 0, BPTone.success),
                BPDistributionPart("agotados", 0, BPTone.danger),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
