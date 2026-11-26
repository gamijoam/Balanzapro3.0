import "package:flutter/widgets.dart";

import "../design_system/bp_badge.dart";
import "../design_system/bp_metrics.dart";
import "../design_system/design_system.dart";
import "inventory_consultation.dart";

/// All numbers are derived from the current synthetic session.
class InventoryDashboard extends StatelessWidget {
  const InventoryDashboard({
    super.key,
    required this.products,
    required this.hasDraft,
    required this.onFilter,
    required this.onProducts,
    required this.onNew,
    required this.onMovements,
  });

  final List<PreviewProduct> products;
  final bool hasDraft;
  final ValueChanged<String> onFilter;
  final VoidCallback onProducts, onNew, onMovements;

  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    final low = products.where((p) => p.stock > 0 && p.stock < 10).length;
    final empty = products.where((p) => p.stock == 0).length;
    final available = products.length - low - empty;
    final cards = [
      BPMetricCard(
        label: "Productos de ejemplo",
        value: products.length.toString(),
        actionKey: const ValueKey("dashboard-all"),
        actionLabel: "Ver todos",
        onPressed: hasDraft ? null : () => onFilter("all"),
      ),
      BPMetricCard(
        label: "Stock bajo de ejemplo",
        value: low.toString(),
        tone: BPTone.warning,
        actionKey: const ValueKey("dashboard-low"),
        actionLabel: "Ver stock bajo",
        onPressed: hasDraft ? null : () => onFilter("low"),
      ),
      BPMetricCard(
        label: "Agotados de ejemplo",
        value: empty.toString(),
        tone: BPTone.danger,
        actionKey: const ValueKey("dashboard-empty"),
        actionLabel: "Ver agotados",
        onPressed: hasDraft ? null : () => onFilter("empty"),
      ),
    ];
    Widget pair(List<Widget> items) => Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (var i = 0; i < items.length; i++) ...[
          if (i > 0) const SizedBox(width: BPTokens.sm),
          Expanded(child: items[i]),
        ],
      ],
    );
    return LayoutBuilder(
      builder: (context, constraints) {
        final normalScale = MediaQuery.textScalerOf(context).scale(14) <= 21;
        final wide =
            normalScale && constraints.maxWidth >= BPTokens.wideBreakpoint;
        final compactPair =
            normalScale &&
            constraints.maxWidth >= BPTokens.sidebarWidth + BPTokens.xxl * 4;
        return ListView(
          key: const ValueKey("home"),
          padding: const EdgeInsets.all(BPTokens.lg),
          children: [
            Text("Resumen de ejemplo", style: theme.heading),
            const SizedBox(height: BPTokens.xs),
            Text(
              "Inventario · Indicadores ficticios de esta sesión",
              style: theme.muted,
            ),
            const SizedBox(height: BPTokens.lg),
            if (hasDraft) ...[
              Text(
                "Tienes un borrador de ejemplo sin guardar.",
                style: theme.muted,
              ),
              const SizedBox(height: BPTokens.sm),
            ],
            BPButton(
              key: const ValueKey("dashboard-new"),
              label: hasDraft ? "Continuar borrador" : "Nuevo producto",
              onPressed: onNew,
            ),
            const SizedBox(height: BPTokens.lg),
            if (wide)
              pair(cards)
            else if (compactPair) ...[
              cards[0],
              const SizedBox(height: BPTokens.sm),
              pair(cards.sublist(1)),
            ] else
              for (final card in cards)
                Padding(
                  padding: const EdgeInsets.only(bottom: BPTokens.sm),
                  child: card,
                ),
            const SizedBox(height: BPTokens.lg),
            BPDistribution(
              title: "Distribución de existencias",
              parts: [
                BPDistributionPart("disponibles", available, BPTone.success),
                BPDistributionPart("con stock bajo", low, BPTone.warning),
                BPDistributionPart("agotados", empty, BPTone.danger),
              ],
            ),
            const SizedBox(height: BPTokens.lg),
            BPCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text("Accesos rápidos", style: theme.title),
                  const SizedBox(height: BPTokens.md),
                  BPButton(
                    label: "Ver productos",
                    primary: false,
                    onPressed: onProducts,
                  ),
                  const SizedBox(height: BPTokens.sm),
                  BPButton(
                    label: "Ver movimientos",
                    primary: false,
                    onPressed: onMovements,
                  ),
                ],
              ),
            ),
          ],
        );
      },
    );
  }
}
