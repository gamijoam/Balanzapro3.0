import "package:flutter/widgets.dart";

import "bp_badge.dart";
import "design_system.dart";

/// A presentation metric. Counts and actions are supplied by the caller.
class BPMetricCard extends StatelessWidget {
  const BPMetricCard({
    super.key,
    required this.label,
    required this.value,
    required this.actionLabel,
    required this.onPressed,
    this.tone = BPTone.neutral,
    this.actionKey,
  });

  final String label, value, actionLabel;
  final BPTone tone;
  final VoidCallback? onPressed;
  final Key? actionKey;

  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    final (_, foreground) = BPBadge.colors(tone, theme.brightness);
    return BPCard(
      compact: true,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(label, style: theme.muted),
          const SizedBox(height: BPTokens.sm),
          Text(
            value,
            style: theme.heading.copyWith(
              color: foreground,
              fontWeight: FontWeight.w700,
            ),
          ),
          const SizedBox(height: BPTokens.md),
          BPButton(
            key: actionKey,
            label: actionLabel,
            primary: false,
            onPressed: onPressed,
          ),
        ],
      ),
    );
  }
}

class BPDistributionPart {
  const BPDistributionPart(this.label, this.count, this.tone);
  final String label;
  final int count;
  final BPTone tone;
}

/// Accessible summary of a small, mutually exclusive set of counts.
/// The bar is decorative; the accompanying text communicates every count.
class BPDistribution extends StatelessWidget {
  BPDistribution({
    super.key,
    required this.title,
    required List<BPDistributionPart> parts,
  }) : parts = List.unmodifiable(parts) {
    if (parts.isEmpty ||
        parts.length > 8 ||
        parts.any((p) => p.count < 0 || p.label.trim().isEmpty)) {
      throw ArgumentError("Invalid distribution");
    }
  }

  final String title;
  final List<BPDistributionPart> parts;

  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    final total = parts.fold<int>(0, (sum, p) => sum + p.count);
    return BPCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(title, style: theme.title),
          const SizedBox(height: BPTokens.md),
          if (total > 0)
            ExcludeSemantics(
              child: ClipRRect(
                borderRadius: BorderRadius.circular(BPTokens.radius),
                child: SizedBox(
                  height: BPTokens.md,
                  child: Row(
                    children: [
                      for (final part in parts)
                        if (part.count > 0)
                          Expanded(
                            flex: part.count,
                            child: ColoredBox(
                              color: BPBadge.colors(
                                part.tone,
                                theme.brightness,
                              ).$2,
                            ),
                          ),
                    ],
                  ),
                ),
              ),
            )
          else
            Text("Sin registros", style: theme.muted),
          const SizedBox(height: BPTokens.md),
          for (final part in parts)
            Padding(
              padding: const EdgeInsets.only(bottom: BPTokens.xs),
              child: Text(
                "${part.count} de $total ${part.label}",
                style: theme.label,
              ),
            ),
        ],
      ),
    );
  }
}
